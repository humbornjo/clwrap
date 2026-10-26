// Package config carries clwrap's rule configuration: comment widths and
// structured-line exemptions.
package config

// Config is clwrap's rule configuration: the two width constraints of the
// comment tiers plus the structured-line exemptions.
type Config struct {
	// DocWidth bounds doc comments of package-level named declarations; a word straddling the boundary may end
	// DocWidth+Tolerance columns out.
	DocWidth int
	// CommentWidth bounds every other standalone comment.
	CommentWidth int
	// ExemptedPrefixes identifies lines whose paragraphs keep their author's layout.
	ExemptedPrefixes []string
	// Tolerance is the doc tier's straddle allowance.
	Tolerance int
}

// Default is the house configuration: doc width 75, comment width 120, no
// tolerance, and the standard code-block and markdown bullet prefixes
// exempt.
func Default() Config {
	return Config{
		DocWidth:         75,
		CommentWidth:     120,
		ExemptedPrefixes: []string{" ", "\t", "- ", "* ", "+ "},
	}
}
