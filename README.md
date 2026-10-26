# clwrap

Reflows standalone `//` line comments in Go, TypeScript, TSX, Rust, and Zig under two width constraints. Go uses `go/ast`; the other languages use Tree-sitter behind the same language-neutral comment pipeline. Rust and Zig also preserve their `///` and `//!` doc-comment markers.

- **Doc comments on top-level named declarations** wrap to the **short** width (default 75, `-doc-width`) with an optional **straddle tolerance** (`-tolerance N`, default 0): a word that starts inside the boundary may end up to N columns past it; a word starting past the boundary wraps. For Go this includes the package clause and top-level funcs and specs; for TypeScript it includes adjacent comments on top-level functions, classes, interfaces, types, enums, variables, and namespaces, exported or not. Rust covers top-level items, including declarations after attributes; Zig covers top-level functions, variables, and tests.
- **Every other standalone comment** — function bodies, methods, fields, interface members, and detached file-header comments — joins and wraps to the **long** width (default 120, `-comment-width`).
- **Markdown list items** (`- `, `* `, `+ `, `N. `) join their continuation lines into one line per item and may exceed the tier width.

Left untouched: trailing comments on code lines, `/* */` block comments (including JSDoc), language-specific directive lines, and structured paragraphs inside a reflowed group. `ExemptedPrefixes` configures the prefix-based exemptions; its defaults are space, tab, `- `, `* `, and `+ `. Numbered list items, doc link definitions, and prose-led paragraphs that mix text with list items also keep their layout. A directive splits its group, and the prose runs around it still reflow. TypeScript directives include triple-slash lines, `@ts-` comments, ESLint/TSLint, Prettier, Biome, Istanbul, and c8 annotations. Rust preserves `rustfmt::` controls; Zig preserves `zig fmt:` controls. In both languages `///` and `//!` runs reflow with their exact markers, while `////` comments stay verbatim.

Go files with the standard generated marker (`// Code generated … DO NOT EDIT.`) are never rewritten; stdout mode passes them through whole.

## Usage

```
clwrap file.go             # print formatted Go source to stdout
clwrap file.ts             # TypeScript is detected by extension
clwrap file.tsx            # TSX uses the TSX grammar
clwrap file.rs             # Rust uses Tree-sitter
clwrap file.zig            # Zig uses Tree-sitter
clwrap -w ./...            # rewrite every supported file beneath the current directory
clwrap -dryrun ./...       # list the files -w would rewrite, writing nothing
clwrap -w service/...      # same, rooted at service/
clwrap - < file.go         # stdin defaults to Go
clwrap --lang typescript - < file.ts
clwrap --lang tsx - < file.tsx
clwrap --lang rust - < file.rs
clwrap --lang zig - < file.zig
clwrap report ./...        # per-directory comment hygiene table, tokei style
```

Supported extensions are `.go`, `.ts`, `.mts`, `.cts`, `.tsx`, `.rs`, and `.zig`. Explicit files with other extensions are rejected rather than parsed as Go. `--lang` selects the stdin grammar and accepts `go`, `typescript` (or `ts`), `tsx`, `rust`, and `zig`.

The `report` subcommand formats in memory, writes nothing, and tallies the comment-line change per directory: files seen, files the reflow would rewrite, comment lines before/after, and the signed diff — the most changed directories first. Only directories containing files the reflow would rewrite get rows; the total still covers every supported file. `--limit N` selects the number of rows (default 10, 0 = all), with the rest folded into an `other` row. The table is styled after tokei: `━` rules spanning the terminal width, ANSI colors on a terminal (`NO_COLOR` disables). It accepts the same width flags; stdin is not a report input.

Args are files, directories, `dir/...` recursive patterns, or `-` for stdin. Directory walks ignore dot/underscore directories, `vendor`, `node_modules`, and `testdata`. Flags: `--doc-width N` (75), `--tolerance N` (0), `--comment-width N` (120), `-w`/`--write`, `-d`/`--dryrun`, and `--lang` for stdin. Output is idempotent; Go output remains gofmt-stable.

Building requires CGo and a C compiler because the Tree-sitter runtime and language grammars compile as C.

## Layout

A stream pipeline: a language adapter extracts comment areas, the rule set transforms them, and each area's single write overrides its region.

- `main.go` — CLI wiring only (urfave/cli/v3, the fuss command pattern)
- `config/` — the rule configuration: DocWidth + Tolerance, CommentWidth, ExemptedPrefixes
- `command/internal/walk` — positional-arg expansion and supported-file discovery
- `command/fmt` — file/stdin IO, exit codes, and pipeline invocation
- `command/report` — in-memory format, per-directory tally, tokei-style table
- `package/comment` — language-neutral classes, areas, comment streams, and byte-range replacement
- `package/astx` — the `Parser` interface, adapter registry, extension detection, and explicit language selection
- `package/astx/lang/go` — the Go adapter using `go/ast`
- `package/astx/lang/internal/tree` — grammar-neutral Tree-sitter traversal, grouping, and span handling; adapters inject language policy
- `package/astx/lang/typescript` — the TypeScript and TSX adapter using Tree-sitter
- `package/astx/lang/rust` — the Rust adapter, including doc markers, rustfmt directives, and declaration classification
- `package/astx/lang/zig` — the Zig adapter, including doc markers, zig fmt directives, and declaration classification
- `package/rule` — marker-aware `Rule` + `Set` chain and the `Pipe` stream driver; `Apply` configures the standard `List` → `Width` pipeline, `Width` carries tier boundaries and prefix exemptions, and `List` owns markdown list items

## Neovim

For Go, run after the normal formatter (gopls) as a comment bonus: astrolsp's `format_on_save` ignores `go`, and a `BufWritePre *.go` autocmd in astrocore pipes the buffer through `clwrap -` after `vim.lsp.buf.format`. A buffer that fails to parse keeps the LSP-formatted text. For Tree-sitter-backed buffers, pass `--lang typescript`, `--lang tsx`, `--lang rust`, or `--lang zig` explicitly.
