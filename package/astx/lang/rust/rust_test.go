package rust

import (
	"testing"

	"clwrap/config"
	"clwrap/package/comment"
	"clwrap/package/rule"
)

func reflow(t *testing.T, src string) string {
	t.Helper()
	file, err := (Parser{}).Parse("input.rs", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.DocWidth, cfg.CommentWidth, cfg.Tolerance = 60, 100, 2
	if err := rule.Apply(file, cfg); err != nil {
		t.Fatal(err)
	}
	return string(file.Source())
}

func TestRust_Reflow(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "outer doc comments wrap short and retain their marker",
			src: "/// Run documents the declaration in one long line that must wrap at the short width boundary.\n" +
				"pub fn run() {}\n",
			want: "/// Run documents the declaration in one long line that must\n" +
				"/// wrap at the short width boundary.\n" +
				"pub fn run() {}\n",
		},
		{
			name: "inner lines join at the long width",
			src: "fn run() {\n" +
				"    // inner prose written\n" +
				"    // in narrow lines\n" +
				"}\n",
			want: "fn run() {\n" +
				"    // inner prose written in narrow lines\n" +
				"}\n",
		},
		{
			name: "inner doc comments retain their marker",
			src: "//! Module documentation written\n" +
				"//! in narrow lines.\n\n" +
				"pub fn run() {}\n",
			want: "//! Module documentation written in narrow lines.\n\n" +
				"pub fn run() {}\n",
		},
		{
			name: "rustfmt directives split and preserve prose runs",
			src: "// Prose before the formatter directive\n" +
				"// joins into one line.\n" +
				"// rustfmt::skip\n" +
				"// Prose after the directive\n" +
				"// joins into one line.\n" +
				"pub fn run() {}\n",
			want: "// Prose before the formatter directive joins into one line.\n" +
				"// rustfmt::skip\n" +
				"// Prose after the directive joins into one line.\n" +
				"pub fn run() {}\n",
		},
		{
			name: "block trailing and four-slash comments stay untouched",
			src: "//// An ordinary four-slash comment keeps its layout.\n" +
				"/* A block comment also keeps its layout. */\n" +
				"fn run() {\n" +
				"    let value = 1; // trailing prose stays exactly here\n" +
				"    let _ = value;\n" +
				"}\n",
			want: "//// An ordinary four-slash comment keeps its layout.\n" +
				"/* A block comment also keeps its layout. */\n" +
				"fn run() {\n" +
				"    let value = 1; // trailing prose stays exactly here\n" +
				"    let _ = value;\n" +
				"}\n",
		},
		{
			name: "comment markers in strings stay untouched",
			src:  "const TEXT: &str = \"// this string is not a comment and must never be reformatted\";\n",
			want: "const TEXT: &str = \"// this string is not a comment and must never be reformatted\";\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflow(t, tc.src)
			if got != tc.want {
				t.Fatalf("pipe:\n%s\nwant:\n%s", got, tc.want)
			}
			if again := reflow(t, got); again != got {
				t.Fatalf("not idempotent:\n%s\nthen:\n%s", got, again)
			}
		})
	}
}

func TestRust_Classification(t *testing.T) {
	src := "// File license.\n\n" +
		"/// Function docs.\npub fn run() {}\n" +
		"/// Struct docs.\n#[derive(Debug)]\npub struct S;\n" +
		"impl S {\n    /// Method docs.\n    fn method(&self) {}\n}\n" +
		"/// Expression docs.\nrun();\n"
	file, err := (Parser{}).Parse("input.rs", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var classes []comment.Class
	var markers []string
	for area := file.Next(); area != nil; area = file.Next() {
		classes = append(classes, area.Class())
		markers = append(markers, area.Marker())
	}
	wantClasses := []comment.Class{comment.Inner, comment.Decl, comment.Decl, comment.Inner, comment.Inner}
	wantMarkers := []string{"//", "///", "///", "///", "///"}
	if len(classes) != len(wantClasses) {
		t.Fatalf("areas = %d, want %d; classes = %v", len(classes), len(wantClasses), classes)
	}
	for i := range wantClasses {
		if classes[i] != wantClasses[i] {
			t.Errorf("area %d class = %v, want %v", i, classes[i], wantClasses[i])
		}
		if markers[i] != wantMarkers[i] {
			t.Errorf("area %d marker = %q, want %q", i, markers[i], wantMarkers[i])
		}
	}
}

func TestRust_SyntaxError(t *testing.T) {
	if _, err := (Parser{}).Parse("broken.rs", []byte("pub fn broken( {\n")); err == nil {
		t.Fatal("a syntax error must reject the whole source")
	}
}

func TestRust_CommentLines(t *testing.T) {
	src := "//! module\n\n// one\n/* two\nthree */\n/// docs\nconst VALUE: i32 = 1; // four\n"
	file, err := (Parser{}).Parse("input.rs", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := file.CommentLines(); got != 6 {
		t.Fatalf("comment lines = %d, want 6", got)
	}
}
