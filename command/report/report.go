// Package cmdreport is the gommentlw report command: it walks the
// positional args the way the fmt command does and, without writing
// anything, summarizes the comment-line change the reflow would make,
// grouped by Go package — tokei style, the most changed packages first.
package cmdreport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"

	"gommentlw/command/walk"
	"gommentlw/config"
	"gommentlw/package/astx"
	"gommentlw/package/rule"
)

// stat accumulates one package's comment hygiene: the .go files seen, the
// files the reflow would rewrite, and the comment lines before and after
// it.
type stat struct {
	files  int
	dirty  int
	before int
	after  int
}

// add folds another package's tally into s.
func (s *stat) add(o *stat) {
	s.files, s.dirty, s.before, s.after = s.files+o.files, s.dirty+o.dirty, s.before+o.before, s.after+o.after
}

// Run tallies every positional arg the command received and renders the
// per-package table, reporting the failures on stderr and exiting 2 when
// any arg failed.
func Run(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit("no input: gommentlw report [-limit N] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/...>", 2)
	}
	cfg := config.Default()
	cfg.Decl, cfg.Inner, cfg.List, cfg.Tolerance = cmd.Int("decl"), cmd.Int("inner"), cmd.Int("list"), cmd.Int("tolerance")

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
	render(os.Stdout, stats, int(cmd.Int("limit")))
	return nil
}

// collect formats one file in memory and folds its comment-line change
// into its package's tally; the package key is the file's directory.
func collect(path string, cfg config.Config, stats map[string]*stat) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	before, err := commentLines(path, src)
	if err != nil {
		return err
	}
	out, err := format(path, src, cfg)
	if err != nil {
		return err
	}
	after, err := commentLines(path, out)
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

// commentLines counts the lines comments occupy in one source buffer: each
// comment group contributes its line span, doc to end.
func commentLines(name string, src []byte) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, group := range file.Comments {
		n += fset.Position(group.End()).Line - fset.Position(group.Pos()).Line + 1
	}
	return n, nil
}

// format composes the same stream pipeline the fmt command runs: parse the
// file, pipe its comment areas through the rule set, render.
func format(name string, src []byte, cfg config.Config) ([]byte, error) {
	file, err := astx.ParseFile(name, src)
	if err != nil {
		return nil, err
	}
	if err := rule.Pipe(file, rule.Set{rule.List{Width: cfg.List}, rule.Width{Decl: cfg.Decl, Inner: cfg.Inner, Tolerance: cfg.Tolerance}}); err != nil {
		return nil, err
	}
	return file.Source(), nil
}

// render prints the per-package table, tokei style: the limit most changed
// packages in descending order, an "other" row folding the rest away, and
// the total over every package seen. A limit of 0 shows all.
func render(w io.Writer, stats map[string]*stat, limit int) {
	type row struct {
		name string
		s    *stat
	}
	rows := make([]row, 0, len(stats))
	for name, s := range stats {
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

	var other, total stat
	shown := rows
	if limit > 0 && len(rows) > limit {
		for _, r := range rows[limit:] {
			other.add(r.s)
		}
		shown = rows[:limit]
	}
	for _, r := range rows {
		total.add(r.s)
	}
	otherName := fmt.Sprintf("other (%d packages)", len(rows)-len(shown))
	if len(rows)-len(shown) == 1 {
		otherName = "other (1 package)"
	}

	nw := len("Package")
	for _, name := range []string{otherName, "Total"} {
		nw = max(nw, len(name))
	}
	for _, r := range shown {
		nw = max(nw, len(r.name))
	}
	line := strings.Repeat("=", nw+49)
	fmt.Fprintln(w, line)
	fmt.Fprintf(w, " %-*s %7s %7s %7s %7s %7s\n", nw, "Package", "Files", "Dirty", "Before", "After", "Diff")
	fmt.Fprintln(w, line)
	for _, r := range shown {
		printRow(w, nw, r.name, r.s)
	}
	if other.files > 0 {
		printRow(w, nw, otherName, &other)
	}
	fmt.Fprintln(w, line)
	printRow(w, nw, "Total", &total)
	fmt.Fprintln(w, line)
}

// printRow prints one table line: the package name and its tallies, the
// diff signed when nonzero.
func printRow(w io.Writer, nw int, name string, s *stat) {
	diff := "      0"
	if d := s.after - s.before; d != 0 {
		diff = fmt.Sprintf("%+7d", d)
	}
	fmt.Fprintf(w, " %-*s %7d %7d %7d %7d %s\n", nw, name, s.files, s.dirty, s.before, s.after, diff)
}
