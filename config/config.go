// Package config carries gommentlw's rule configuration: the two width
// constraints of the comment tiers.
package config

// Config is gommentlw's rule configuration: the two width constraints of
// the comment tiers plus the markdown list budget.
type Config struct {
	// Decl bounds doc comments of package-level named declarations; a word straddling the boundary may end Decl+Tolerance
	// columns out.
	Decl int
	// Tolerance is the Decl tier's straddle allowance.
	Tolerance int
	// Inner bounds every other standalone comment.
	Inner int
	// List bounds markdown list items; 0 leaves them unbounded past the tier width.
	List int
}

// Default is the house configuration: decl 75, inner 120, no tolerance,
// inner 120, lists unbounded.
func Default() Config {
	return Config{Decl: 75, Inner: 120}
}
