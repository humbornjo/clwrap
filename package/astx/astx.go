// Package astx defines the source-parser contract and selects registered
// language adapters by file extension or explicit language name.
package astx

import (
	"fmt"
	"path/filepath"
	"strings"

	golang "clwrap/package/astx/lang/go"
	"clwrap/package/astx/lang/rust"
	"clwrap/package/astx/lang/typescript"
	"clwrap/package/astx/lang/zig"
	"clwrap/package/comment"
)

// Parser extracts a language-neutral comment stream from one source
// buffer. Language adapters under lang implement this interface.
type Parser interface {
	Parse(name string, src []byte) (*comment.File, error)
}

// registration associates one Parser implementation with the names and
// file extensions that select it.
type registration struct {
	name       string
	aliases    []string
	extensions []string
	parser     Parser
}

var registrations = []registration{
	{name: "go", extensions: []string{".go"}, parser: golang.Parser{}},
	{name: "typescript", aliases: []string{"ts"}, extensions: []string{".ts", ".mts", ".cts"}, parser: typescript.Parser{}},
	{name: "tsx", extensions: []string{".tsx"}, parser: typescript.TSXParser{}},
	{name: "rust", extensions: []string{".rs"}, parser: rust.Parser{}},
	{name: "zig", extensions: []string{".zig"}, parser: zig.Parser{}},
}

// Supports reports whether path has an extension handled by a registered
// language adapter.
func Supports(path string) bool {
	_, ok := forExtension(path)
	return ok
}

// Parse selects an adapter from name's extension and parses src.
func Parse(name string, src []byte) (*comment.File, error) {
	registration, ok := forExtension(name)
	if !ok {
		return nil, fmt.Errorf("unsupported source file %q", name)
	}
	return registration.parser.Parse(name, src)
}

// ParseAs selects an adapter by explicit language name and parses src. The
// canonical names are go, typescript, tsx, rust, and zig; ts aliases
// typescript.
func ParseAs(languageName, name string, src []byte) (*comment.File, error) {
	languageName = strings.ToLower(languageName)
	for i := range registrations {
		registration := &registrations[i]
		if languageName == registration.name {
			return registration.parser.Parse(name, src)
		}
		for _, alias := range registration.aliases {
			if languageName == alias {
				return registration.parser.Parse(name, src)
			}
		}
	}
	return nil, fmt.Errorf("unsupported language %q (want go, typescript, tsx, rust, or zig)", languageName)
}

// CommentLines counts all lines occupied by comments in one source buffer,
// including comments that are not reflowable.
func CommentLines(name string, src []byte) (int, error) {
	file, err := Parse(name, src)
	if err != nil {
		return 0, err
	}
	return file.CommentLines(), nil
}

// forExtension finds the registration responsible for name's final
// extension.
func forExtension(name string) (*registration, bool) {
	extension := strings.ToLower(filepath.Ext(name))
	for i := range registrations {
		registration := &registrations[i]
		for _, candidate := range registration.extensions {
			if extension == candidate {
				return registration, true
			}
		}
	}
	return nil, false
}
