// Command gommentlw reflows Go line comments under two width constraints:
// doc comments on package-level named declarations — the package clause,
// top-level funcs, and const/var/type specs of top-level declarations,
// grouped or not — wrap to the short width, optionally letting a
// boundary-straddling word run past it (-tolerance), and every other
// standalone comment (function bodies, struct fields, interface methods)
// wraps to the long width. Markdown list items join into one line each,
// exceeding the tier width unless -list bounds them. Trailing comments,
// block comments, and groups holding directive-style lines (//go:…,
// //nolint…) keep their author's layout, as do code-block and
// link-definition paragraphs. The report subcommand summarizes, per Go
// package and tokei style, the comment-line change the reflow would make.
package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	cmdfmt "gommentlw/command/fmt"
	cmdreport "gommentlw/command/report"
)

func main() {
	(&cli.Command{
		Name:      "gommentlw",
		Usage:     "reflow Go line comments under the two-tier width rule",
		UsageText: "gommentlw [-w] [-dryrun] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/... | ->...",
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
		}, widthFlags()...),
		Commands: []*cli.Command{
			{
				Name:      "report",
				Usage:     "report the repo's comment hygiene: the line change the reflow would make, per Go package",
				UsageText: "gommentlw report [-limit N] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/...>",
				Flags: append([]cli.Flag{
					&cli.IntFlag{
						Name:  "limit",
						Value: 10,
						Usage: "show at most N packages, the most changed first (0 = all)",
					},
				}, widthFlags()...),
				Action: cmdreport.Run,
			},
		},
		Action: cmdfmt.Run,
	}).Run(context.Background(), os.Args) // nolint:errcheck // Run reports its own failures
}

// widthFlags are the rule configuration flags the fmt action and the
// report command share.
func widthFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:  "decl",
			Value: 75,
			Usage: "wrap width for package-level declaration doc comments",
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
		&cli.IntFlag{
			Name:  "list",
			Value: 0,
			Usage: "wrap markdown list items at N columns (0 = no constraint)",
		},
	}
}
