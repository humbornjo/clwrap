package rule

import (
	"strings"

	"gommentlw/package/astx"
)

// Width joins a comment area's prose paragraphs and rewraps them to the
// tier's boundary: CLASS_DECL holds to Decl, CLASS_INNER to Inner. A word
// straddling the Decl boundary may end up to Tolerance columns past it; a
// word starting past the boundary wraps. Paragraphs carrying their own
// layout — code blocks, list items, link definitions — stay verbatim.
type Width struct {
	Decl      int
	Inner     int
	Tolerance int
}

// Rewrite implements Rule.
func (w Width) Rewrite(class astx.Class, text string) string {
	boundary, tolerance := w.Inner, 0
	if class == astx.CLASS_DECL {
		boundary, tolerance = w.Decl, w.Tolerance
	}
	indent, lines := areaLines(text)
	textWidth := boundary - len(indent) - len("// ")
	if textWidth < 1 {
		textWidth = 1
	}
	return strings.Join(verbatimLines(indent, mapParagraphs(lines, func(p []string) []string {
		return reflowProse(p, textWidth, tolerance)
	})), "\n")
}

// reflowProse joins a paragraph's lines and greedy-wraps the words under
// the text width; a paragraph holding any structured line keeps its
// author's layout.
func reflowProse(lines []string, textWidth, tolerance int) []string {
	for _, line := range lines {
		if !plainProse(line) {
			return lines
		}
	}
	var out []string
	var b strings.Builder
	for _, word := range strings.Fields(strings.Join(lines, " ")) {
		if b.Len() > 0 {
			if end := b.Len() + 1 + len(word); end > textWidth && !tolerated(b.Len(), len(word), textWidth, tolerance) {
				out = append(out, b.String())
				b.Reset()
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(word)
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// tolerated reports whether the tolerance lets the next word end past the
// boundary: the word must start within the boundary — a full word past
// it, however small, wraps instead — and end within the extra allowance.
func tolerated(lineLen, wordLen, boundary, tolerance int) bool {
	return lineLen+1 < boundary && lineLen+1+wordLen <= boundary+tolerance
}
