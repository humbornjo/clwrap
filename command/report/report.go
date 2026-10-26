// Package cmdreport is the clwrap report command: it walks the positional
// args the way the fmt command does and, without writing anything,
// summarizes the comment-line change the reflow would make, grouped by
// directory — tokei style, the most changed directories first.
package cmdreport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"clwrap/command/internal/walk"
	"clwrap/config"
	"clwrap/package/astx"
	"clwrap/package/rule"
)

// stat accumulates one directory's comment hygiene: the supported source
// files seen, the files the reflow would rewrite, and the comment lines
// before and after it.
type stat struct {
	files  int
	dirty  int
	before int
	after  int
}

// add folds another directory's tally into s.
func (s *stat) add(o *stat) {
	s.files, s.dirty, s.before, s.after = s.files+o.files, s.dirty+o.dirty, s.before+o.before, s.after+o.after
}

// Run tallies every positional arg the command received and renders the
// per-directory table, reporting failures on stderr and exiting 2 when any
// arg failed.
func Run(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit("no input: clwrap report [-limit N] [-decl N] [-inner N] [-tolerance N] <file | dir | dir/...>", 2)
	}
	cfg := config.Default()
	cfg.DocWidth, cfg.CommentWidth, cfg.Tolerance = cmd.Int("decl"), cmd.Int("inner"), cmd.Int("tolerance")

	stats := map[string]*stat{}
	err := walk.Args(cmd.Args().Slice(), func(path string) error {
		if path == "-" {
			return errors.New("report: - (stdin) is not a report input")
		}
		return collect(path, cfg, stats)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.Exit("", 2)
	}
	fd := int(os.Stdout.Fd())
	columns := 80
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		columns = w
	}
	render(os.Stdout, stats, int(cmd.Int("limit")), columns, term.IsTerminal(fd) && os.Getenv("NO_COLOR") == "")
	return nil
}

// collect formats one file in memory and folds its comment-line change
// into its directory's tally.
func collect(path string, cfg config.Config, stats map[string]*stat) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, err := astx.Parse(path, src)
	if err != nil {
		return err
	}
	before := file.CommentLines()
	if err := rule.Apply(file, cfg); err != nil {
		return err
	}
	out := file.Source()
	after, err := astx.CommentLines(path, out)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	s := stats[dir]
	if s == nil {
		s = &stat{}
		stats[dir] = s
	}
	s.files++
	if !bytes.Equal(out, src) {
		s.dirty++
	}
	s.before += before
	s.after += after
	return nil
}

// Column widths, tokei's: Files is narrower than the rest, and the name
// column takes the terminal width left over by the numbers and gaps (63).
const (
	_FILES_WIDTH = 8
	_NUM_WIDTH   = 12
)

// table renders the report tokei style: ━ separators spanning the
// terminal width, the header in bold blue, directory names in bold
// magenta, and the total's numbers in blue — colors only on a terminal.
type table struct {
	w     io.Writer
	color bool
	nw    int
	sep   string
}

// newTable sizes the name column so the table fills columns wide.
func newTable(w io.Writer, columns int, color bool) *table {
	return &table{w: w, color: color, nw: max(columns-63, len("Directory")), sep: strings.Repeat("━", columns)}
}

// paint wraps s in an ANSI color when the table renders to a terminal.
func (t *table) paint(code, s string) string {
	if !t.color || code == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// header prints the column titles in bold blue between two rules.
func (t *table) header() {
	fmt.Fprintln(t.w, t.sep)
	fmt.Fprintf(t.w, " %s %s %s %s %s %s\n",
		t.paint("1;34", fmt.Sprintf("%-*s", t.nw, "Directory")),
		t.paint("1;34", fmt.Sprintf("%*s", _FILES_WIDTH, "Files")),
		t.paint("1;34", fmt.Sprintf("%*s", _NUM_WIDTH, "Dirty")),
		t.paint("1;34", fmt.Sprintf("%*s", _NUM_WIDTH, "Before")),
		t.paint("1;34", fmt.Sprintf("%*s", _NUM_WIDTH, "After")),
		t.paint("1;34", fmt.Sprintf("%*s", _NUM_WIDTH, "Diff")),
	)
	fmt.Fprintln(t.w, t.sep)
}

// line prints one tally row: the directory name in bold magenta, truncated
// with a trailing | when it overflows as tokei does, the numbers plain —
// or blue on the total row — and the diff signed when nonzero.
func (t *table) line(name string, s *stat, total bool) {
	if r := []rune(name); len(r) > t.nw {
		name = string(r[:t.nw-1]) + "|"
	}
	code := ""
	if total {
		code = "34"
	}
	diff := fmt.Sprintf("%*d", _NUM_WIDTH, 0)
	if d := s.after - s.before; d != 0 {
		diff = fmt.Sprintf("%+*d", _NUM_WIDTH, d)
	}
	fmt.Fprintf(t.w, " %s %s %s %s %s %s\n",
		t.paint("1;35", fmt.Sprintf("%-*s", t.nw, name)),
		t.paint(code, fmt.Sprintf("%*d", _FILES_WIDTH, s.files)),
		t.paint(code, fmt.Sprintf("%*d", _NUM_WIDTH, s.dirty)),
		t.paint(code, fmt.Sprintf("%*d", _NUM_WIDTH, s.before)),
		t.paint(code, fmt.Sprintf("%*d", _NUM_WIDTH, s.after)),
		t.paint(code, diff),
	)
}

// render prints the most changed directories in descending order, an "other"
// row folding away those beyond limit, and the total over every directory
// seen. Unchanged directories only contribute to the total. A limit of 0 shows
// all changed directories.
func render(w io.Writer, stats map[string]*stat, limit, columns int, color bool) {
	type row struct {
		name string
		s    *stat
	}
	rows := make([]row, 0, len(stats))
	var total stat
	for name, s := range stats {
		total.add(s)
		if s.dirty == 0 {
			continue
		}
		rows = append(rows, row{name, s})
	}
	sort.Slice(rows, func(i, j int) bool {
		di, dj := rows[i].s.after-rows[i].s.before, rows[j].s.after-rows[j].s.before
		if di < 0 {
			di = -di
		}
		if dj < 0 {
			dj = -dj
		}
		if di != dj {
			return di > dj
		}
		if rows[i].s.dirty != rows[j].s.dirty {
			return rows[i].s.dirty > rows[j].s.dirty
		}
		return rows[i].name < rows[j].name
	})

	var other stat
	shown := rows
	if limit > 0 && len(rows) > limit {
		for _, r := range rows[limit:] {
			other.add(r.s)
		}
		shown = rows[:limit]
	}
	otherName := fmt.Sprintf("other (%d directories)", len(rows)-len(shown))
	if len(rows)-len(shown) == 1 {
		otherName = "other (1 directory)"
	}

	t := newTable(w, columns, color)
	t.header()
	for _, r := range shown {
		t.line(r.name, r.s, false)
	}
	if other.files > 0 {
		t.line(otherName, &other, false)
	}
	fmt.Fprintln(t.w, t.sep)
	t.line("Total", &total, true)
	fmt.Fprintln(t.w, t.sep)
}
