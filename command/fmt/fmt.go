// Package cmdfmt is the clwrap command action: it walks the positional
// args — files, directories, dir/... recursive patterns, and - for
// standard input — and streams each supported source's comment areas
// through the rules.
package cmdfmt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"clwrap/command/internal/walk"
	"clwrap/config"
	"clwrap/package/astx"
	"clwrap/package/rule"
)

// Run formats every positional arg the command received, reporting
// failures on stderr and exiting 2 when any arg failed.
func Run(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		return cli.Exit("no input: clwrap [-w] [-dryrun] [-doc-width N] [-comment-width N] [-tolerance N] [--lang go|typescript|tsx|rust|zig] <file | dir | dir/... | ->...", 2)
	}
	write, dry, cfg := cmd.Bool("write"), cmd.Bool("dryrun"), config.Default()
	cfg.DocWidth, cfg.CommentWidth, cfg.Tolerance = cmd.Int("doc-width"), cmd.Int("comment-width"), cmd.Int("tolerance")
	languageName := cmd.String("lang")

	err := walk.Args(cmd.Args().Slice(), func(path string) error {
		if path == "-" {
			return processStdin(cfg, languageName)
		}
		return processFile(path, write, dry, cfg)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.Exit("", 2)
	}
	return nil
}

// processFile formats one file: dry mode lists files whose comments would
// change on stdout and writes nothing, stdout mode prints reflowed source,
// and write mode rewrites changed files in place and lists them on stderr.
func processFile(path string, write, dry bool, cfg config.Config) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, err := astx.Parse(path, src)
	if err != nil {
		return err
	}
	if err := rule.Apply(file, cfg); err != nil {
		return err
	}
	out := file.Source()
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

// processStdin formats standard input to standard output, the editor
// integration path. Stdin remains Go by default; --lang selects another
// registered grammar. A syntax error leaves the editor's buffer untouched.
func processStdin(cfg config.Config, languageName string) error {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if languageName == "" {
		languageName = "go"
	}
	file, err := astx.ParseAs(languageName, "stdin", src)
	if err != nil {
		return err
	}
	if err := rule.Apply(file, cfg); err != nil {
		return err
	}
	_, err = os.Stdout.Write(file.Source())
	return err
}
