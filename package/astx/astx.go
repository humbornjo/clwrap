// Package astx wraps go/ast to stream a Go file's comment areas: ParseFile
// yields a File whose Next hands out each reflowable comment group as an
// Area — a single-shot io.Reader over the original text and a
// single-shot io.Writer whose one write overrides the region — until
// Next returns nil. Source then renders the file with every override
// applied.
package astx

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"regexp"
	"strings"
)

// Class is a comment area's tier under the two-tier width rule.
type Class int

const (
	// CLASS_DECL marks doc comments of package-level named declarations: the
	// package clause, top-level funcs, and the const/var/type specs of
	// top-level declarations, grouped or not.
	CLASS_DECL Class = iota
	// CLASS_INNER marks every other standalone comment: function bodies,
	// struct fields, interface methods.
	CLASS_INNER
)

// Area is one comment group's source region. Read streams the original
// text — leading indent included — exactly once; the drained stream
// reports io.EOF. Write accepts exactly one call, and the write overrides
// the region: the bytes given replace the original text when the file
// renders.
type Area interface {
	io.Reader
	io.Writer
	// Class is the area's comment tier.
	Class() Class
}

// File is a parsed Go source file's comment stream.
type File struct {
	src   []byte
	areas []*area
	next  int
}

var GENERATED_MARKER = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

// ParseFile parses one Go source buffer into a comment stream. A generated
// file parses into a File with no areas: its comments belong to the
// generator, so the stream is drained from the start.
func ParseFile(name string, src []byte) (*File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	f := &File{src: src}
	head := src
	if len(head) > 4096 {
		head = head[:4096]
	}
	if GENERATED_MARKER.Match(head) {
		return f, nil
	}
	for _, group := range file.Comments {
		if a := extract(fset, file, src, group); a != nil {
			f.areas = append(f.areas, a)
		}
	}
	return f, nil
}

// Next returns the file's next comment area, or nil once the stream is
// drained.
func (f *File) Next() Area {
	if f.next >= len(f.areas) {
		return nil
	}
	a := f.areas[f.next]
	f.next++
	return a
}

// Source renders the file with every overridden area's replacement
// applied.
func (f *File) Source() []byte {
	out := f.src
	for i := len(f.areas) - 1; i >= 0; i-- {
		a := f.areas[i]
		if !a.wrote || bytes.Equal(a.repl, a.src) {
			continue
		}
		var b bytes.Buffer
		b.Write(out[:a.start])
		b.Write(a.repl)
		b.Write(out[a.end:])
		out = b.Bytes()
	}
	return out
}

// area is one comment group's source region: the original text plus the
// single override, when written.
type area struct {
	class Class
	start int
	end   int
	src   []byte
	repl  []byte
	off   int
	wrote bool
}

// Class implements Area.
func (a *area) Class() Class { return a.class }

// Read implements Area: the original text streams out once, then the area
// reports io.EOF.
func (a *area) Read(p []byte) (int, error) {
	if a.off >= len(a.src) {
		return 0, io.EOF
	}
	n := copy(p, a.src[a.off:])
	a.off += n
	return n, nil
}

// Write implements Area: one write overrides the region's original text; a
// second write fails.
func (a *area) Write(p []byte) (int, error) {
	if a.wrote {
		return 0, errors.New("astx: comment area already overridden")
	}
	a.wrote = true
	a.repl = bytes.Clone(p)
	return len(p), nil
}

// extract lifts one comment group into an area, or returns nil when the
// group is no reflow candidate: block comments, trailing comments sharing
// a code line, and groups holding directive-style lines all keep their
// author's layout.
func extract(fset *token.FileSet, file *ast.File, src []byte, group *ast.CommentGroup) *area {
	first := group.List[0]
	if strings.HasPrefix(first.Text, "/*") {
		return nil
	}
	pos := fset.Position(first.Slash)
	start := pos.Offset - (pos.Column - 1)
	if strings.TrimSpace(string(src[start:pos.Offset])) != "" {
		return nil
	}
	for _, c := range group.List {
		body := strings.TrimPrefix(c.Text, "//")
		if body != "" && !strings.HasPrefix(body, " ") {
			return nil
		}
	}
	end := fset.Position(group.End()).Offset
	return &area{class: classify(file, group), start: start, end: end, src: src[start:end]}
}

// classify reports the comment group's tier: doc comments attached to
// package-level named declarations — the package clause, top-level
// funcs, and the specs of top-level const/var/type declarations, grouped
// or not — are CLASS_DECL.
func classify(file *ast.File, group *ast.CommentGroup) Class {
	if file.Doc == group {
		return CLASS_DECL
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Doc == group {
				return CLASS_DECL
			}
		case *ast.GenDecl:
			if d.Doc == group {
				return CLASS_DECL
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ImportSpec:
					if s.Doc == group {
						return CLASS_DECL
					}
				case *ast.ValueSpec:
					if s.Doc == group {
						return CLASS_DECL
					}
				case *ast.TypeSpec:
					if s.Doc == group {
						return CLASS_DECL
					}
				}
			}
		}
	}
	return CLASS_INNER
}
