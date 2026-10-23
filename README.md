# gommentlw

Reflows Go line comments under two width constraints, using go/ast for classification:

- **Doc comments on package-level named declarations** — the package clause, top-level funcs, and const/var/type specs of top-level declarations, grouped or not — wrap to the **short** width (default 75, `-decl`) with an optional **straddle tolerance** (`-tolerance N`, default 0): a word that starts inside the boundary may end up to N columns past it; a word starting past the boundary wraps.
- **Every other standalone comment** — function bodies, struct fields, interface methods — joins and wraps to the **long** width (default 120, `-inner`).

- **Markdown list items** (`- `, `* `, `+ `, `N. `) join their continuation lines into one line per item and may exceed the tier width; `-list N` bounds them (default 0: no constraint), wrapping with the continuation aligned under the item text.

Generated files (`// Code generated … DO NOT EDIT.`) are never rewritten; stdout mode passes them through whole.

Left untouched: trailing comments on code lines, `/* */` block comments, directive-style lines themselves (`//go:…`, `//nolint`, …) — a directive splits its group and the prose runs around it still reflow — and structured paragraphs inside a reflowed group (code blocks, doc link definitions, prose-led paragraphs that mix text with list items).

## Usage

```
gommentlw file.go            # print formatted source to stdout (gofmt style)
gommentlw -w ./...           # rewrite every .go file beneath the current directory
gommentlw -dryrun ./...      # list the files -w would rewrite, writing nothing
gommentlw -w service/...     # same, rooted at service/
gommentlw - < file.go        # stdin → stdout, the editor-integration path
gommentlw report ./...       # per-package comment hygiene table, tokei style
```

The `report` subcommand formats in memory, writes nothing, and tallies the comment-line change per Go package: files seen, files the reflow would rewrite, comment lines before/after, and the signed diff — the most changed packages first, `--limit N` rows (default 10, 0 = all) with the rest folded into an `other` row. It accepts the same width flags; stdin is not a report input.

Args are files, directories, `dir/...` recursive patterns, or `-` for stdin. Directory walks ignore dot/underscore directories, `vendor`, and `testdata`, per the go tool convention. Flags: `--decl N` (75), `--tolerance N` (0), `--inner N` (120), `--list N` (0), `-w`/`--write`, `-d`/`--dryrun`. Output is idempotent and gofmt-stable.

## Layout

A stream pipeline: the ast wrapper extracts comment areas, the rule set transforms them, and each area's single write overrides its region.

- `main.go` — CLI wiring only (urfave/cli/v3, the fuss command pattern)
- `config/` — the rule configuration: Decl width + Tolerance, Inner width, List budget
- `command/walk` — positional-arg expansion: files, directories, `dir/...` patterns, `-` passthrough
- `command/fmt` — the fmt action: file/stdin IO, exit codes, pipeline composition
- `command/report` — the report subcommand: in-memory format, per-package tally, tokei-style table
- `package/astx` — the go/ast wrapper: `ParseFile` yields a `File` whose `Next()` streams each reflowable comment run — a prose stretch between directive lines — as an `Area` (a single-shot `io.Reader`; its one `Write` overrides the region) until nil; `Source()` renders the result
- `package/rule` — `Rule` + `Set` chain and the `Pipe` stream driver; `Width` carries the tier boundaries with the straddle tolerance, `List` owns markdown list items

## Neovim

Runs after the normal formatter (gopls) as a comment bonus: astrolsp's `format_on_save` ignores `go`, and a `BufWritePre *.go` autocmd in astrocore pipes the buffer through `gommentlw -` after `vim.lsp.buf.format`. A buffer that fails to parse keeps the LSP-formatted text.
