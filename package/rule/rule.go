// Package rule transforms the comment areas of an astx.File stream: a Set
// chains rules, and Pipe drives the stream — each area's text reads once
// through the rules and the result writes back over the area.
package rule

import (
	"io"
	"strings"

	"gommentlw/package/astx"
)

// Rule rewrites one comment area's text under the area's tier.
type Rule interface {
	Rewrite(class astx.Class, text string) string
}

// Set applies rules in order, each receiving the previous rule's output.
type Set []Rule

// Rewrite runs the area text through every rule of the set in order.
func (s Set) Rewrite(class astx.Class, text string) string {
	for _, r := range s {
		text = r.Rewrite(class, text)
	}
	return text
}

// Pipe streams the file's comment areas through the rule set, writing each
// rule output back over its area.
func Pipe(file *astx.File, rules Set) error {
	for area := file.Next(); area != nil; area = file.Next() {
		text, err := io.ReadAll(area)
		if err != nil {
			return err
		}
		if _, err := area.Write([]byte(rules.Rewrite(area.Class(), string(text)))); err != nil {
			return err
		}
	}
	return nil
}

// areaLines splits an area's raw text into its leading indent and its
// marker-stripped lines: each line loses the indent, the "//" marker, and
// one following space.
func areaLines(text string) (string, []string) {
	indent := text[:strings.Index(text, "//")]
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		body := strings.TrimPrefix(strings.TrimPrefix(line, indent), "//")
		lines = append(lines, strings.TrimPrefix(body, " "))
	}
	return indent, lines
}

// mapParagraphs splits the marker-stripped lines on blank lines and
// reassembles them with each paragraph replaced by fn's output, preserving
// the blank separators.
func mapParagraphs(lines []string, fn func([]string) []string) []string {
	var out []string
	for len(lines) > 0 {
		if lines[0] == "" {
			out = append(out, "")
			lines = lines[1:]
			continue
		}
		i := 0
		for i < len(lines) && lines[i] != "" {
			i++
		}
		out = append(out, fn(lines[:i])...)
		lines = lines[i:]
	}
	return out
}

// verbatimLines renders marker-stripped lines back into comment text with
// the indent and "// " marker restored.
func verbatimLines(indent string, lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, indent+"//")
		} else {
			out = append(out, indent+"// "+line)
		}
	}
	return out
}

// plainProse reports whether a line is flowing prose: doc-comment code
// blocks (indent beyond the marker), list items, and link definitions
// carry their own layout.
func plainProse(line string) bool {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return false
	}
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "+ ") {
		return false
	}
	if i := strings.IndexAny(line, ".)"); i > 0 && i+1 < len(line) && line[i+1] == ' ' && digitsOnly(line[:i]) {
		return false
	}
	if strings.HasPrefix(line, "[") && strings.Contains(line, "]: ") {
		return false
	}
	return true
}

func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
