// Package rust uses Tree-sitter to extract standalone Rust line comments.
// Outer and inner doc comments retain their markers; trailing comments,
// block comments, and rustfmt directives keep their author's layout.
package rust

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	treesitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"

	"clwrap/package/astx/lang/internal/tree"
	"clwrap/package/comment"
)

var _SPEC = tree.Spec{
	Name:      "Rust",
	IsComment: commentKind,
	Marker:    marker,
	Directive: directive,
	Classify:  classify,
}

// Parser adapts the Rust Tree-sitter grammar to the shared parser
// contract.
type Parser struct{}

// Parse implements astx.Parser for Rust source.
func (Parser) Parse(name string, src []byte) (*comment.File, error) {
	return tree.Parse(name, src, sitter.NewLanguage(treesitterrust.Language()), _SPEC)
}

func commentKind(kind string) bool {
	return kind == "line_comment" || kind == "block_comment"
}

// marker distinguishes Rust outer and inner doc comments from ordinary //
// comments. Four or more slashes are ordinary comments and stay verbatim.
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

// directive preserves rustfmt controls without treating documentation
// about rustfmt as a tooling directive.
func directive(text string) bool {
	if marker(text) != "//" {
		return false
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	return strings.HasPrefix(body, "rustfmt::")
}

// classify marks a root-level comment group CLASS_DOC when a named item follows
// it. Outer attributes may sit between the documentation and its item.
func classify(last *sitter.Node, endRow uint) comment.Class {
	parent := last.Parent()
	if parent == nil || parent.Kind() != "source_file" {
		return comment.CLASS_COMMENT
	}
	row := endRow
	next := last.NextNamedSibling()
	for next != nil && next.Kind() == "attribute_item" {
		if next.StartPosition().Row != row+1 {
			return comment.CLASS_COMMENT
		}
		row = next.EndPosition().Row
		next = next.NextNamedSibling()
	}
	if next == nil || next.StartPosition().Row != row+1 || !namedDeclaration(next) {
		return comment.CLASS_COMMENT
	}
	return comment.CLASS_DOC
}

// namedDeclaration recognizes root-level Rust items that introduce a name.
func namedDeclaration(node *sitter.Node) bool {
	switch node.Kind() {
	case "const_item",
		"enum_item",
		"extern_crate_declaration",
		"function_item",
		"function_signature_item",
		"macro_definition",
		"mod_item",
		"static_item",
		"struct_item",
		"trait_item",
		"type_item",
		"union_item":
		return true
	}
	return false
}
