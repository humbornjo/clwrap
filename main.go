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
// link-definition paragraphs.
package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	cmdfmt "gommentlw/command/fmt"
)

func main() {
	(&cli.Command{
		Name:      "gommentlw",
		Usage:     "reflow Go line comments under the two-tier width rule",
		UsageText: "gommentlw [-w] [-decl N] [-inner N] [-tolerance N] [-list N] <file.go | dir | dir/... | ->...",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "write",
				Aliases: []string{"w"},
				Usage:   "write result to source file instead of stdout",
			},
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
		},
		Action: cmdfmt.Run,
	}).Run(context.Background(), os.Args) // nolint:errcheck // Run reports its own failures
}
