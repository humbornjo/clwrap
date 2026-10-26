package zig

import (
	"testing"

	"clwrap/config"
	"clwrap/package/comment"
	"clwrap/package/rule"
)

func reflow(t *testing.T, src string) string {
	t.Helper()
	file, err := (Parser{}).Parse("input.zig", []byte(src))
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

func TestZig_Reflow(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "declaration docs wrap short and retain their marker",
			src: "/// Run documents the declaration in one long line that must wrap at the short width boundary.\n" +
				"pub fn run() void {}\n",
			want: "/// Run documents the declaration in one long line that must\n" +
				"/// wrap at the short width boundary.\n" +
				"pub fn run() void {}\n",
		},
		{
			name: "inner lines join at the long width",
			src: "fn run() void {\n" +
				"    // inner prose written\n" +
				"    // in narrow lines\n" +
				"}\n",
			want: "fn run() void {\n" +
				"    // inner prose written in narrow lines\n" +
				"}\n",
		},
		{
			name: "container docs retain their marker",
			src: "//! Module documentation written\n" +
				"//! in narrow lines.\n\n" +
				"pub fn run() void {}\n",
			want: "//! Module documentation written in narrow lines.\n\n" +
				"pub fn run() void {}\n",
		},
		{
			name: "formatter directives split and preserve prose runs",
			src: "// Prose before the formatter directive\n" +
				"// joins into one line.\n" +
				"// zig fmt: off\n" +
				"// Prose after the directive\n" +
				"// joins into one line.\n" +
				"pub fn run() void {}\n",
			want: "// Prose before the formatter directive joins into one line.\n" +
				"// zig fmt: off\n" +
				"// Prose after the directive joins into one line.\n" +
				"pub fn run() void {}\n",
		},
		{
			name: "trailing and four-slash comments stay untouched",
			src: "//// An ordinary four-slash comment keeps its layout.\n" +
				"const value: i32 = 1; // trailing prose stays exactly here\n",
			want: "//// An ordinary four-slash comment keeps its layout.\n" +
				"const value: i32 = 1; // trailing prose stays exactly here\n",
		},
		{
			name: "comment markers in strings stay untouched",
			src:  "const text = \"// this string is not a comment and must never be reformatted\";\n",
			want: "const text = \"// this string is not a comment and must never be reformatted\";\n",
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

func TestZig_Classification(t *testing.T) {
	src := "// File license.\n\n" +
		"/// Function docs.\npub fn run() void {}\n" +
		"/// Variable docs.\npub const S = struct {\n" +
		"    /// Method docs.\n    pub fn method() void {}\n};\n" +
		"/// Test docs.\ntest \"example\" {}\n"
	file, err := (Parser{}).Parse("input.zig", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var classes []comment.Class
	var markers []string
	for area := file.Next(); area != nil; area = file.Next() {
		classes = append(classes, area.Class())
		markers = append(markers, area.Marker())
	}
	wantClasses := []comment.Class{comment.CLASS_COMMENT, comment.CLASS_DOC, comment.CLASS_DOC, comment.CLASS_COMMENT, comment.CLASS_DOC}
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

func TestZig_SyntaxError(t *testing.T) {
	if _, err := (Parser{}).Parse("broken.zig", []byte("pub fn broken( {\n")); err == nil {
		t.Fatal("a syntax error must reject the whole source")
	}
}

func TestZig_CommentLines(t *testing.T) {
	src := "// one\n// two\nconst value: i32 = 1; // three\n"
	file, err := (Parser{}).Parse("input.zig", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := file.CommentLines(); got != 3 {
		t.Fatalf("comment lines = %d, want 3", got)
	}
}
