// Package comment owns the language-neutral comment stream and source
// editor. Language adapters identify reflowable regions; rules transform
// their text, and File renders each single override over the original
// source.
package comment

import (
	"bytes"
	"errors"
	"io"
)

// Class is a comment area's tier under the two-tier width rule.
type Class int

const (
	// CLASS_DOC marks documentation attached to a top-level named
	// declaration.
	CLASS_DOC Class = iota
	// CLASS_COMMENT marks every other standalone comment.
	CLASS_COMMENT
)

// Area is one comment run's source region. Read streams the original text
// exactly once; Write accepts one replacement for the region.
type Area interface {
	io.Reader
	io.Writer
	// Class is the area's comment tier.
	Class() Class
	// Marker is the line-comment marker, such as // or #.
	Marker() string
}

// Region describes one adapter-extracted run of reflowable comment lines.
type Region struct {
	// Start is the region's inclusive byte offset, including its indentation.
	Start int
	// End is the region's exclusive byte offset.
	End int
	// Class is the region's width tier.
	Class Class
	// Marker is the line-comment marker shared by every line in the region.
	Marker string
}

// File is a parsed source file's comment stream and replacement renderer.
type File struct {
	src          []byte
	areas        []*area
	next         int
	commentLines int
}

// NewFile builds a comment stream from adapter-extracted regions.
// commentLines is the number of source lines occupied by comments,
// including regions the formatter deliberately leaves untouched.
func NewFile(src []byte, regions []Region, commentLines int) *File {
	f := &File{src: src, commentLines: commentLines}
	for _, region := range regions {
		f.areas = append(f.areas, &area{
			class:  region.Class,
			marker: region.Marker,
			start:  region.Start,
			end:    region.End,
			src:    src[region.Start:region.End],
		})
	}
	return f
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

// CommentLines returns the number of lines comments occupied in the parsed
// source, including trailing, block, and directive comments.
func (f *File) CommentLines() int { return f.commentLines }

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

// area is one comment run's original text plus its single override, if
// written.
type area struct {
	class  Class
	marker string
	start  int
	end    int
	src    []byte
	repl   []byte
	off    int
	wrote  bool
}

// Class implements Area.
func (a *area) Class() Class { return a.class }

// Marker implements Area.
func (a *area) Marker() string { return a.marker }

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
		return 0, errors.New("comment: area already overridden")
	}
	a.wrote = true
	a.repl = bytes.Clone(p)
	return len(p), nil
}
