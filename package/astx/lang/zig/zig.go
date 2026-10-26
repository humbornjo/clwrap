// Package zig uses Tree-sitter to extract standalone Zig line comments.
// Declaration and container doc comments retain their markers; trailing
// comments and zig fmt directives keep their author's layout.
package zig

import (
	"strings"

	treesitterzig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"

	"clwrap/package/astx/lang/tree"
	"clwrap/package/comment"
)

var spec = tree.Spec{
	Name:      "Zig",
	IsComment: commentKind,
	Marker:    marker,
	Directive: directive,
	Classify:  classify,
}

// Parser adapts the Zig Tree-sitter grammar to the shared parser contract.
type Parser struct{}

// Parse implements astx.Parser for Zig source.
func (Parser) Parse(name string, src []byte) (*comment.File, error) {
	return tree.Parse(name, src, sitter.NewLanguage(treesitterzig.Language()), spec)
}

func commentKind(kind string) bool {
	return kind == "comment"
}

// marker distinguishes Zig declaration and container doc comments from
// ordinary // comments. Four or more slashes are ordinary and stay verbatim.
func marker(text string) string {
	switch {
	case strings.HasPrefix(text, "////"):
		return ""
	case strings.HasPrefix(text, "///"):
		return "///"
	case strings.HasPrefix(text, "//!"):
		return "//!"
	case strings.HasPrefix(text, "//"):
		return "//"
	default:
		return ""
	}
}

// directive preserves Zig formatter controls without treating doc comments
// as tooling directives.
func directive(text string) bool {
	if marker(text) != "//" {
		return false
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	return strings.HasPrefix(body, "zig fmt:")
}

// classify marks a comment group Decl only when a named root-level
// declaration immediately follows it. Container-level //! comments remain
// Inner.
func classify(last *sitter.Node, endRow uint) comment.Class {
	parent := last.Parent()
	if parent == nil || parent.Kind() != "source_file" {
		return comment.Inner
	}
	next := last.NextNamedSibling()
	if next == nil || next.StartPosition().Row != endRow+1 || !namedDeclaration(next) {
		return comment.Inner
	}
	return comment.Decl
}

// namedDeclaration recognizes named root-level Zig declarations.
func namedDeclaration(node *sitter.Node) bool {
	switch node.Kind() {
	case "function_declaration", "test_declaration", "variable_declaration":
		return true
	}
	return false
}
