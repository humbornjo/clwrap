// Package cmdfmt is the gommentlw command action: it walks the positional
// args — files, directories, dir/... recursive patterns, and - for
// standard input — and streams each source's comment areas through the
// rule set.
package cmdfmt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"gommentlw/config"
	"gommentlw/package/astx"
	"gommentlw/package/rule"
)

// Run formats every positional arg the command received, reporting each
// failure on stderr and exiting 2 when any arg failed.
func Run(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit("no input: gommentlw [-w] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/... | ->...", 2)
	}
	write, cfg := cmd.Bool("write"), config.Default()
	cfg.Decl, cfg.Inner, cfg.List, cfg.Tolerance = cmd.Int("decl"), cmd.Int("inner"), cmd.Int("list"), cmd.Int("tolerance")

	failed := false
	for _, arg := range cmd.Args().Slice() {
		if arg == "-" {
			if err := processStdin(cfg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
			continue
		}
		if root, recursive := strings.CutSuffix(arg, "/..."); recursive {
			if root == "" {
				root = "."
			}
			if err := walkDir(root, write, cfg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
			continue
		}
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
			continue
		}
		if info.IsDir() {
			if err := walkDir(arg, write, cfg); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
			continue
		}
		if err := processFile(arg, write, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}
	if failed {
		return cli.Exit("", 2)
	}
	return nil
}

// walkDir processes every .go file beneath root and joins the failures.
// Like the go tool, directories starting with a dot or an underscore are
// ignored, vendor and testdata too.
func walkDir(root string, write bool, cfg config.Config) error {
	var errs []error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				if name := d.Name(); strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "testdata" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			if err := processFile(path, write, cfg); err != nil {
				errs = append(errs, err)
			}
		}
		return nil
	})
	return errors.Join(append(errs, err)...)
}

// processFile formats one file: stdout mode prints the reflowed source,
// write mode rewrites changed files in place and lists them on stderr.
func processFile(path string, write bool, cfg config.Config) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := format(path, src, cfg)
	if err != nil {
		return err
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
