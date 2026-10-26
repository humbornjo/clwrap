package typescript

import (
	"io"
	"testing"

	"clwrap/config"
	"clwrap/package/comment"
	"clwrap/package/rule"
)

// pipeString runs one source buffer through an adapter and the production
// rule order.
func pipeString(t *testing.T, parse func(string, []byte) (*comment.File, error), src string, shortW, longW int) string {
	t.Helper()
	file, err := parse("input.ts", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.DocWidth, cfg.CommentWidth, cfg.Tolerance = shortW, longW, 2
	if err := rule.Apply(file, cfg); err != nil {
		t.Fatal(err)
	}
	return string(file.Source())
}

func TestTypescript_Reflow(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "exported declaration doc wraps short",
			src: "// Run documents the declaration in one long line that must wrap at the short width boundary.\n" +
				"export function Run() {}\n",
			want: "// Run documents the declaration in one long line that must\n" +
				"// wrap at the short width boundary.\n" +
				"export function Run() {}\n",
		},
		{
			name: "inner lines join at the long width",
			src: "function run() {\n" +
				"  // inner prose written\n" +
				"  // in narrow lines\n" +
				"}\n",
			want: "function run() {\n" +
				"  // inner prose written in narrow lines\n" +
				"}\n",
		},
		{
			name: "directives split and preserve prose runs",
			src: "// Prose before the compiler directive\n" +
				"// joins into one line.\n" +
				"// @ts-expect-error keep this compiler directive unchanged\n" +
				"// Prose before the lint directive\n" +
				"// joins into one line.\n" +
				"// eslint-disable-next-line keep-this-rule-name\n" +
				"/// <reference path=\"./types.d.ts\" />\n" +
				"// Prose after the directives\n" +
				"// joins into one line.\n" +
				"export function Run() {}\n",
			want: "// Prose before the compiler directive joins into one line.\n" +
				"// @ts-expect-error keep this compiler directive unchanged\n" +
				"// Prose before the lint directive joins into one line.\n" +
				"// eslint-disable-next-line keep-this-rule-name\n" +
				"/// <reference path=\"./types.d.ts\" />\n" +
				"// Prose after the directives joins into one line.\n" +
				"export function Run() {}\n",
		},
		{
			name: "block and trailing comments stay untouched",
			src: "/** JSDoc keeps its own layout even when this line is long. */\n" +
				"export function run() {\n" +
				"  const value = 1; // trailing prose also stays exactly here\n" +
				"  /* and so does this block comment */\n" +
				"}\n",
			want: "/** JSDoc keeps its own layout even when this line is long. */\n" +
				"export function run() {\n" +
				"  const value = 1; // trailing prose also stays exactly here\n" +
				"  /* and so does this block comment */\n" +
				"}\n",
		},
		{
			name: "comment markers in strings and templates stay untouched",
			src: "const text = \"// this string is not a comment and must never be reformatted\";\n" +
				"const template = `// this template line is not a comment\n" +
				"// and neither is this one`;\n",
			want: "const text = \"// this string is not a comment and must never be reformatted\";\n" +
				"const template = `// this template line is not a comment\n" +
				"// and neither is this one`;\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pipeString(t, (Parser{}).Parse, tc.src, 60, 100)
			if got != tc.want {
				t.Fatalf("pipe:\n%s\nwant:\n%s", got, tc.want)
			}
			if again := pipeString(t, (Parser{}).Parse, got, 60, 100); again != got {
				t.Fatalf("not idempotent:\n%s\nthen:\n%s", got, again)
			}
		})
	}
}

func TestTypescript_Classification(t *testing.T) {
	src := "// File license.\n\n" +
		"// Function docs.\nexport function run() {}\n" +
		"// Class docs.\nexport default class C {\n  // Method docs.\n  method(): void {}\n}\n" +
		"// Interface docs.\ninterface Options {}\n" +
		"// Type docs.\ntype ID = string;\n" +
		"// Enum docs.\nenum Color { red }\n" +
		"// Variable docs.\nconst answer = 42;\n" +
		"// Namespace docs.\nnamespace Models {\n  // Namespace member docs.\n  interface Item {}\n}\n" +
		"// Expression docs.\nrun();\n"
	file, err := (Parser{}).Parse("input.ts", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var got []comment.Class
	for area := file.Next(); area != nil; area = file.Next() {
		if area.Marker() != "//" {
			t.Fatalf("marker = %q, want //", area.Marker())
		}
		got = append(got, area.Class())
	}
	want := []comment.Class{
		comment.Inner,
		comment.Decl,
		comment.Decl,
		comment.Inner,
		comment.Decl,
		comment.Decl,
		comment.Decl,
		comment.Decl,
		comment.Decl,
		comment.Inner,
		comment.Inner,
	}
	if len(got) != len(want) {
		t.Fatalf("classes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("area %d class = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestTypescript_TSX(t *testing.T) {
	src := "// View documents the declaration in one long line that must wrap at the short width boundary.\n" +
		"export function View() {\n" +
		"  return (\n" +
		"    <div>\n" +
		"      // JSX text, not a comment\n" +
		"      {/* keep this block comment exactly as written */}\n" +
		"    </div>\n" +
		"  );\n" +
		"}\n"
	want := "// View documents the declaration in one long line that must\n" +
		"// wrap at the short width boundary.\n" +
		"export function View() {\n" +
		"  return (\n" +
		"    <div>\n" +
		"      // JSX text, not a comment\n" +
		"      {/* keep this block comment exactly as written */}\n" +
		"    </div>\n" +
		"  );\n" +
		"}\n"
	got := pipeString(t, (TSXParser{}).Parse, src, 60, 100)
	if got != want {
		t.Fatalf("pipe:\n%s\nwant:\n%s", got, want)
	}
	if again := pipeString(t, (TSXParser{}).Parse, got, 60, 100); again != got {
		t.Fatalf("not idempotent:\n%s\nthen:\n%s", got, again)
	}
}

func TestTypescript_SyntaxError(t *testing.T) {
	if _, err := (Parser{}).Parse("broken.ts", []byte("export function broken( {\n")); err == nil {
		t.Fatal("a syntax error must reject the whole source")
	}
}

func TestTypescript_CommentLines(t *testing.T) {
	src := "// one\n/* two\nthree */\nconst value = 1; // four\n"
	file, err := (Parser{}).Parse("input.ts", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if got := file.CommentLines(); got != 4 {
		t.Fatalf("comment lines = %d, want 4", got)
	}
	for area := file.Next(); area != nil; area = file.Next() {
		if _, err := io.Copy(io.Discard, area); err != nil {
			t.Fatal(err)
		}
	}
}
