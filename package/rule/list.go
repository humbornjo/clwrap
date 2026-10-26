package rule

import (
	"strings"

	"clwrap/package/comment"
)

// List joins each markdown list item of a list paragraph into a single
// line; items may exceed the tier width, so the tier boundary never breaks
// them. A paragraph that is not entirely list items keeps its author's
// layout.
type List struct {
	// ExemptedPrefixes identifies continuation lines whose paragraphs keep their layout.
	ExemptedPrefixes []string
}

// Rewrite implements Rule.
func (l List) Rewrite(_ comment.Class, marker, text string) string {
	indent, lines := areaLines(text, marker)
	return strings.Join(verbatimLines(indent, marker, mapParagraphs(lines, func(p []string) []string {
		return reflowList(p, l.ExemptedPrefixes)
	})), "\n")
}

// reflowList joins each list item's continuation lines; a paragraph that
// is not entirely list items keeps its author's layout.
func reflowList(lines []string, exemptedPrefixes []string) []string {
	type item struct {
		indent string
		marker string
		words  []string
	}
	var items []item
	for _, line := range lines {
		if indent, marker, body, ok := bulletItem(line); ok {
			items = append(items, item{indent: indent, marker: marker, words: strings.Fields(body)})
			continue
		}
		if len(items) == 0 {
			return lines
		}
		last := &items[len(items)-1]
		trimmed := strings.TrimLeft(line, " \t")
		if len(line)-len(trimmed) <= len(last.indent) || !plainProse(trimmed, exemptedPrefixes) {
			return lines
		}
		last.words = append(last.words, strings.Fields(trimmed)...)
	}
	var out []string
	for _, it := range items {
		out = append(out, it.indent+it.marker+strings.Join(it.words, " "))
	}
	return out
}

// bulletItem reports the marker indent, marker, and body when the line
// opens a markdown list item: a "- ", "* ", or "+ " bullet, or a "N.
// "/"N)" numbered item, at any indentation.
func bulletItem(line string) (indent, marker, body string, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent = line[:len(line)-len(trimmed)]
	for _, m := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(trimmed, m) {
			return indent, m, trimmed[len(m):], true
		}
	}
	if i := strings.IndexAny(trimmed, ".)"); i > 0 && i+1 < len(trimmed) && trimmed[i+1] == ' ' && digitsOnly(trimmed[:i]) {
		return indent, trimmed[:i+2], trimmed[i+2:], true
	}
	return "", "", "", false
}
