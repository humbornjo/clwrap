package rule

import (
	"strings"

	"gommentlw/package/astx"
)

// List joins each markdown list item of a list paragraph into a single
// line: bullet items may exceed the tier width, so the tier boundary never
// breaks them. When the rule's Width is positive the item instead wraps at
// Width with the continuation aligned under the item text. A paragraph
// that is not entirely list items keeps its author's layout.
type List struct {
	// Width is the item budget in columns; 0, the default, means no constraint.
	Width int
}

// Rewrite implements Rule.
func (l List) Rewrite(_ astx.Class, text string) string {
	indent, lines := areaLines(text)
	budget := 0
	if l.Width > 0 {
		if budget = l.Width - len(indent) - len("// "); budget < 1 {
			budget = 1
		}
	}
	return strings.Join(verbatimLines(indent, mapParagraphs(lines, func(p []string) []string {
		return reflowList(p, budget)
	})), "\n")
}

// reflowList joins a list paragraph's items, each into one line or wrapped
// under the budget; a paragraph that is not entirely list items keeps its
// author's layout.
func reflowList(lines []string, budget int) []string {
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
		if len(line)-len(trimmed) <= len(last.indent) || !plainProse(trimmed) {
			return lines
		}
		last.words = append(last.words, strings.Fields(trimmed)...)
	}
	var out []string
	for _, it := range items {
		body := strings.Join(it.words, " ")
		if avail := budget - len(it.indent) - len(it.marker); budget > 0 && len(body) > avail {
			out = append(out, wrapItem(it.indent, it.marker, it.words, avail)...)
			continue
		}
		out = append(out, it.indent+it.marker+body)
	}
	return out
}

// wrapItem wraps an item's words under the available body width, aligning
// continuation lines under the item text.
func wrapItem(indent, marker string, words []string, avail int) []string {
	if avail < 1 {
		avail = 1
	}
	var out []string
	var b strings.Builder
	flush := func() {
		if len(out) == 0 {
			out = append(out, indent+marker+b.String())
		} else {
			out = append(out, indent+strings.Repeat(" ", len(marker))+b.String())
		}
		b.Reset()
	}
	for _, word := range words {
		if b.Len() > 0 {
			if b.Len()+1+len(word) > avail {
				flush()
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(word)
	}
	if b.Len() > 0 {
		flush()
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
