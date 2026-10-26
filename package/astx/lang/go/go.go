// Package golang uses go/ast to extract Go comment regions into the
// language-neutral comment stream. It preserves Go's declaration-doc
// classification, generated-file policy, and directive handling.
package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"clwrap/package/comment"
)

var GENERATED_MARKER = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

// Parser adapts go/ast to the shared source-parser contract.
type Parser struct{}

// Parse implements astx.Parser for Go source. A generated file has no
// reflowable areas, but its comment lines are still counted for reports.
func (Parser) Parse(name string, src []byte) (*comment.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	head := src
	if len(head) > 4096 {
		head = head[:4096]
	}
	generated := GENERATED_MARKER.Match(head)
	var regions []comment.Region
	commentLines := 0
	for _, group := range file.Comments {
		commentLines += fset.Position(group.End()).Line - fset.Position(group.Pos()).Line + 1
		if !generated {
			regions = append(regions, extract(fset, file, src, group)...)
		}
	}
	return comment.NewFile(src, regions, commentLines), nil
}

// extract lifts one comment group into its reflowable runs: block comments
// and trailing comments sharing a code line keep their author's layout,
// and a directive-style line splits the group into prose runs around it.
func extract(fset *token.FileSet, file *ast.File, src []byte, group *ast.CommentGroup) []comment.Region {
	first := group.List[0]
	if strings.HasPrefix(first.Text, "/*") {
		return nil
	}
	pos := fset.Position(first.Slash)
	if strings.TrimSpace(string(src[pos.Offset-(pos.Column-1):pos.Offset])) != "" {
		return nil
	}
	class := classify(file, group)
	var regions []comment.Region
	from, end := -1, 0
	for _, c := range group.List {
		if body := strings.TrimPrefix(c.Text, "//"); body != "" && !strings.HasPrefix(body, " ") {
			if from >= 0 {
				regions = append(regions, comment.Region{Start: from, End: end, Class: class, Marker: "//"})
				from = -1
			}
			continue
		}
		if from < 0 {
			slash := fset.Position(c.Slash)
			from = slash.Offset - (slash.Column - 1)
		}
		end = fset.Position(c.End()).Offset
	}
	if from >= 0 {
		regions = append(regions, comment.Region{Start: from, End: end, Class: class, Marker: "//"})
	}
	return regions
}

// classify reports the comment group's tier: doc comments attached to
// package-level named declarations are Decl; every other group is Inner.
func classify(file *ast.File, group *ast.CommentGroup) comment.Class {
	if file.Doc == group {
		return comment.Decl
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Doc == group {
				return comment.Decl
			}
		case *ast.GenDecl:
			if d.Doc == group {
				return comment.Decl
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ImportSpec:
					if s.Doc == group {
						return comment.Decl
					}
				case *ast.ValueSpec:
					if s.Doc == group {
						return comment.Decl
					}
				case *ast.TypeSpec:
					if s.Doc == group {
						return comment.Decl
					}
				}
			}
		}
	}
	return comment.Inner
}
