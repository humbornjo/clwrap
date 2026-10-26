// Package tree shares Tree-sitter comment extraction among language
// adapters.
package tree

import (
	"bytes"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"clwrap/package/comment"
)

// Spec describes the language-specific parts of Tree-sitter comment
// extraction.
type Spec struct {
	// Name identifies the language in parse errors.
	Name string
	// IsComment reports whether a node kind is a line or block comment.
	IsComment func(kind string) bool
	// Marker returns the line-comment marker to preserve, or "" for other comments.
	Marker func(text string) string
	// Directive reports whether a line comment must keep its author's layout.
	Directive func(text string) bool
	// Classify assigns a standalone comment group's width tier using its last content row.
	Classify func(last *sitter.Node, endRow uint) comment.Class
}

// Parse builds a comment stream from src using language and spec. Syntax
// errors reject the whole source so an editor buffer can remain unchanged.
func Parse(name string, src []byte, language *sitter.Language, spec Spec) (*comment.File, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("%s: set %s parser language: %w", name, spec.Name, err)
	}
	parsed := parser.Parse(src, nil)
	defer parsed.Close()
	root := parsed.RootNode()
	if root.HasError() {
		return nil, fmt.Errorf("%s: source contains %s syntax errors", name, spec.Name)
	}

	nodes := commentNodes(root, spec.IsComment)
	commentLines := 0
	for i := range nodes {
		_, endRow := commentEnd(src, &nodes[i])
		commentLines += int(endRow-nodes[i].StartPosition().Row) + 1
	}
	return comment.NewFile(src, extractRegions(src, nodes, spec), commentLines), nil
}

// commentNodes collects every comment in source order. The cursor remains
// valid until all regions and report metadata have been copied out of the
// tree.
func commentNodes(root *sitter.Node, isComment func(string) bool) []sitter.Node {
	cursor := root.Walk()
	defer cursor.Close()
	var nodes []sitter.Node
	for {
		node := cursor.Node()
		if isComment(node.Kind()) {
			nodes = append(nodes, *node)
		}
		if cursor.GotoFirstChild() {
			continue
		}
		for !cursor.GotoNextSibling() {
			if !cursor.GotoParent() {
				return nodes
			}
		}
	}
}

// extractRegions groups consecutive standalone line comments, then splits
// each group around directives and marker changes. Other comments never
// reflow.
func extractRegions(src []byte, nodes []sitter.Node, spec Spec) []comment.Region {
	var regions []comment.Region
	for i := 0; i < len(nodes); {
		if !standaloneLine(src, &nodes[i], spec.Marker) {
			i++
			continue
		}
		end := i + 1
		for end < len(nodes) && standaloneLine(src, &nodes[end], spec.Marker) && adjacentLines(src, &nodes[end-1], &nodes[end]) {
			end++
		}
		regions = append(regions, extractGroup(src, nodes[i:end], spec)...)
		i = end
	}
	return regions
}

// standaloneLine reports whether node is a supported line comment alone on
// its source line.
func standaloneLine(src []byte, node *sitter.Node, marker func(string) string) bool {
	start := int(node.StartByte())
	if marker(node.Utf8Text(src)) == "" {
		return false
	}
	return len(bytes.TrimSpace(src[lineStart(src, start):start])) == 0
}

// adjacentLines reports whether two comments occupy consecutive lines with
// no code between them.
func adjacentLines(src []byte, previous, current *sitter.Node) bool {
	previousEnd, previousRow := commentEnd(src, previous)
	if current.StartPosition().Row != previousRow+1 {
		return false
	}
	return len(bytes.TrimSpace(src[previousEnd:int(current.StartByte())])) == 0
}

// extractGroup turns prose runs between directives into source regions.
// Each run retains its exact line-comment marker for the language-neutral
// rules.
func extractGroup(src []byte, group []sitter.Node, spec Spec) []comment.Region {
	last := &group[len(group)-1]
	_, lastRow := commentEnd(src, last)
	class := spec.Classify(last, lastRow)
	var regions []comment.Region
	from, end := -1, 0
	runMarker := ""
	flush := func() {
		if from < 0 {
			return
		}
		regions = append(regions, comment.Region{Start: from, End: end, Class: class, Marker: runMarker})
		from = -1
	}
	for i := range group {
		text := group[i].Utf8Text(src)
		if spec.Directive != nil && spec.Directive(text) {
			flush()
			continue
		}
		marker := spec.Marker(text)
		if from >= 0 && marker != runMarker {
			flush()
		}
		if from < 0 {
			from = lineStart(src, int(group[i].StartByte()))
			runMarker = marker
		}
		end, _ = commentEnd(src, &group[i])
	}
	flush()
	return regions
}

// commentEnd returns the exclusive byte end and final content row. Some
// grammars include a line comment's trailing newline in its node span.
func commentEnd(src []byte, node *sitter.Node) (int, uint) {
	start, end := int(node.StartByte()), int(node.EndByte())
	row := node.EndPosition().Row
	if end > start && src[end-1] == '\n' {
		end--
		if row > node.StartPosition().Row {
			row--
		}
		if end > start && src[end-1] == '\r' {
			end--
		}
	}
	return end, row
}

// lineStart returns the byte offset immediately after the preceding
// newline.
func lineStart(src []byte, offset int) int {
	return bytes.LastIndexByte(src[:offset], '\n') + 1
}
