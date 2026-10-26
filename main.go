// Command clwrap reflows Go, TypeScript, Rust, and Zig line comments under
// two width constraints. Documentation attached to top-level named
// declarations wraps to the short width, with an optional straddle
// tolerance; every other standalone comment wraps to the long width.
// Markdown list items join into one line each. Trailing comments, block
// comments, and language-specific directive lines keep their author's
// layout. The report subcommand summarizes comment-line changes per
// directory.
package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	cmdfmt "clwrap/command/fmt"
	cmdreport "clwrap/command/report"
)

func main() {
	(&cli.Command{
		Name:      "clwrap",
		Usage:     "reflow Go, TypeScript, Rust, and Zig line comments under the two-tier width rule",
		UsageText: "clwrap [-w] [-dryrun] [--lang go|typescript|tsx|rust|zig] [-decl N] [-inner N] [-tolerance N] <file | dir | dir/... | ->...",
		Flags: append([]cli.Flag{
			&cli.BoolFlag{
				Name:    "write",
				Aliases: []string{"w"},
				Usage:   "write result to source file instead of stdout",
			},
			&cli.BoolFlag{
				Name:    "dryrun",
				Aliases: []string{"d"},
				Usage:   "list files whose comments would change, without writing",
			},
			&cli.StringFlag{
				Name:  "lang",
				Usage: "stdin language: go, typescript (ts), tsx, rust, or zig (default go)",
			},
		}, widthFlags()...),
		Commands: []*cli.Command{
			{
				Name:      "report",
				Usage:     "report comment hygiene: the line change the reflow would make, per directory",
				UsageText: "clwrap report [-limit N] [-decl N] [-inner N] [-tolerance N] <file | dir | dir/...>",
				Flags: append([]cli.Flag{
					&cli.IntFlag{
						Name:  "limit",
						Value: 10,
						Usage: "show at most N directories, the most changed first (0 = all)",
					},
				}, widthFlags()...),
				Action: cmdreport.Run,
			},
		},
		Action: cmdfmt.Run,
	}).Run(context.Background(), os.Args) // nolint:errcheck // Run reports its own failures
}

// widthFlags are the rule configuration flags the fmt action and report
// command share.
func widthFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:  "decl",
			Value: 75,
			Usage: "wrap width for top-level declaration doc comments",
		},
		&cli.IntFlag{
			Name:  "tolerance",
			Value: 0,
			Usage: "columns a boundary-straddling word may run past the decl width",
		},
		&cli.IntFlag{
			Name:  "inner",
			Value: 120,
			Usage: "wrap width for all other comments",
		},
	}
}
