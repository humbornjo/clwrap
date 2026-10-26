// Package typescript uses Tree-sitter to extract standalone TypeScript and
// TSX line comments. Block comments, trailing comments, and tooling
// directives keep their author's layout.
package typescript

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	treesittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"clwrap/package/astx/lang/tree"
	"clwrap/package/comment"
)

var spec = tree.Spec{
	Name:      "TypeScript",
	IsComment: commentKind,
	Marker:    marker,
	Directive: directive,
	Classify:  classify,
}

// Parser adapts the TypeScript Tree-sitter grammar to the shared parser
// contract.
type Parser struct{}

// Parse implements astx.Parser for TypeScript source.
func (Parser) Parse(name string, src []byte) (*comment.File, error) {
	return tree.Parse(name, src, sitter.NewLanguage(treesittertypescript.LanguageTypescript()), spec)
}

// TSXParser adapts the TSX Tree-sitter grammar to the shared parser
// contract.
type TSXParser struct{}

// Parse implements astx.Parser for TSX source.
func (TSXParser) Parse(name string, src []byte) (*comment.File, error) {
	return tree.Parse(name, src, sitter.NewLanguage(treesittertypescript.LanguageTSX()), spec)
}

func commentKind(kind string) bool {
	return kind == "comment"
}

// marker retains TypeScript's historical // treatment; triple-slash lines
// are excluded as directives below.
func marker(text string) string {
	if strings.HasPrefix(text, "//") {
		return "//"
	}
	return ""
}

// directive reports whether changing or joining a TypeScript comment could
// alter compiler or tooling behavior. Directive lines split prose groups
// just as Go directives do.
func directive(text string) bool {
	if strings.HasPrefix(text, "///") {
		return true
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	return strings.HasPrefix(body, "@ts-") ||
		body == "eslint" || strings.HasPrefix(body, "eslint-") || strings.HasPrefix(body, "eslint ") ||
		strings.HasPrefix(body, "tslint:") ||
		strings.HasPrefix(body, "prettier-ignore") ||
		strings.HasPrefix(body, "biome-ignore") ||
		strings.HasPrefix(body, "istanbul ignore") ||
		strings.HasPrefix(body, "c8 ignore")
}

// classify marks a comment group Decl only when it is a root-level group
// immediately followed by a named declaration. A blank line or an
// intervening comment prevents doc attachment, as it does with go/ast.
func classify(last *sitter.Node, endRow uint) comment.Class {
	parent := last.Parent()
	if parent == nil || parent.Kind() != "program" {
		return comment.Inner
	}
	next := last.NextNamedSibling()
	if next == nil || next.StartPosition().Row != endRow+1 || !namedDeclaration(next) {
		return comment.Inner
	}
	return comment.Decl
}

// namedDeclaration recognizes top-level named TypeScript declarations. The
// grammar wraps namespaces in expression_statement and exported
// declarations in export_statement, so those wrappers are unwrapped before
// checking.
func namedDeclaration(node *sitter.Node) bool {
	kind := node.Kind()
	if kind == "export_statement" {
		node = node.ChildByFieldName("declaration")
		if node == nil {
			return false
		}
		kind = node.Kind()
	}
	if kind == "expression_statement" {
		node = node.NamedChild(0)
		if node == nil {
			return false
		}
		kind = node.Kind()
	}
	switch kind {
	case "abstract_class_declaration",
		"ambient_declaration",
		"class_declaration",
		"enum_declaration",
		"function_declaration",
		"function_signature",
		"generator_function_declaration",
		"import_alias",
		"interface_declaration",
		"internal_module",
		"lexical_declaration",
		"module",
		"type_alias_declaration",
		"variable_declaration":
		return true
	}
	return false
}
