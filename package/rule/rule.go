// Package rule transforms the areas of a comment.File stream: a Set chains
// rules, and Pipe drives the stream — each area's text reads once
// through the rules and the result writes back over the area.
package rule

import (
	"io"
	"strings"

	"clwrap/config"
	"clwrap/package/comment"
)

// Rule rewrites one comment area's text under its tier and line-comment
// marker.
type Rule interface {
	Rewrite(class comment.Class, marker, text string) string
}

// Set applies rules in order, each receiving the previous rule's output.
type Set []Rule

// Rewrite runs the area text through every rule of the set in order.
func (s Set) Rewrite(class comment.Class, marker, text string) string {
	for _, r := range s {
		text = r.Rewrite(class, marker, text)
	}
	return text
}

// Apply runs the standard List then Width rules over file's comment areas.
func Apply(file *comment.File, cfg config.Config) error {
	rules := Set{
		List{ExemptedPrefixes: cfg.ExemptedPrefixes},
		Width{Decl: cfg.DocWidth, Inner: cfg.CommentWidth, Tolerance: cfg.Tolerance, ExemptedPrefixes: cfg.ExemptedPrefixes},
	}
	return Pipe(file, rules)
}

// Pipe streams the file's comment areas through the rule set, writing each
// rule output back over its area.
func Pipe(file *comment.File, rules Set) error {
	for area := file.Next(); area != nil; area = file.Next() {
		text, err := io.ReadAll(area)
		if err != nil {
			return err
		}
		if _, err := area.Write([]byte(rules.Rewrite(area.Class(), area.Marker(), string(text)))); err != nil {
			return err
		}
	}
	return nil
}

// areaLines splits an area's raw text into its leading indent and its
// marker-stripped lines: each line loses the indent, the marker, and one
// following space.
func areaLines(text, marker string) (string, []string) {
	indent := text[:strings.Index(text, marker)]
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		body := strings.TrimPrefix(strings.TrimPrefix(line, indent), marker)
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
// the indent, marker, and prose spacing restored.
func verbatimLines(indent, marker string, lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, indent+marker)
		} else {
			out = append(out, indent+marker+" "+line)
		}
	}
	return out
}

// plainProse reports whether a line is flowing prose under
// exemptedPrefixes. Numbered list items and link definitions carry their
// own layout regardless.
func plainProse(line string, exemptedPrefixes []string) bool {
	for _, prefix := range exemptedPrefixes {
		if strings.HasPrefix(line, prefix) {
			return false
		}
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
