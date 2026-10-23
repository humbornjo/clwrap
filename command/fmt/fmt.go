// Package cmdfmt is the gommentlw command action: it walks the positional
// args — files, directories, dir/... recursive patterns, and - for
// standard input — and streams each source's comment areas through the
// rule set.
package cmdfmt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"gommentlw/command/walk"
	"gommentlw/config"
	"gommentlw/package/astx"
	"gommentlw/package/rule"
)

// Run formats every positional arg the command received, reporting the
// failures on stderr and exiting 2 when any arg failed.
func Run(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit("no input: gommentlw [-w] [-dryrun] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/... | ->...", 2)
	}
	write, dry, cfg := cmd.Bool("write"), cmd.Bool("dryrun"), config.Default()
	cfg.Decl, cfg.Inner, cfg.List, cfg.Tolerance = cmd.Int("decl"), cmd.Int("inner"), cmd.Int("list"), cmd.Int("tolerance")

	err := walk.Args(cmd.Args().Slice(), func(path string) error {
		if path == "-" {
			return processStdin(cfg)
		}
		return processFile(path, write, dry, cfg)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.Exit("", 2)
	}
	return nil
}

// processFile formats one file: dry mode lists the files whose comments
// would change on stdout and writes nothing, stdout mode prints the
// reflowed source, write mode rewrites changed files in place and lists
// them on stderr.
func processFile(path string, write, dry bool, cfg config.Config) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := format(path, src, cfg)
	if err != nil {
		return err
	}
	if dry {
		if !bytes.Equal(out, src) {
			fmt.Fprintln(os.Stdout, path)
		}
		return nil
	}
	if !write {
		_, err := os.Stdout.Write(out)
		return err
	}
	if bytes.Equal(out, src) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, info.Mode()); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, path)
	return nil
}

// processStdin formats standard input to standard output, the
// editor-integration path: a buffer with a syntax error fails parse and
// leaves the editor's buffer untouched.
func processStdin(cfg config.Config) error {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	out, err := format("stdin.go", src, cfg)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// format runs the stream pipeline over one source buffer: parse the file,
// pipe its comment areas through the rule set, render the result.
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
