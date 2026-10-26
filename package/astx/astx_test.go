package astx

import (
	"bytes"
	"testing"
)

func TestAstx_Parse(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{name: "file.go", src: "package p\n"},
		{name: "file.ts", src: "export const value = 1;\n"},
		{name: "file.mts", src: "export const value = 1;\n"},
		{name: "file.cts", src: "export const value = 1;\n"},
		{name: "file.d.ts", src: "export declare function run(): void;\n"},
		{name: "file.tsx", src: "export const view = <div />;\n"},
		{name: "file.rs", src: "fn main() {}\n"},
		{name: "file.zig", src: "pub fn main() void {}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !Supports(tc.name) {
				t.Fatalf("Supports(%q) = false", tc.name)
			}
			file, err := Parse(tc.name, []byte(tc.src))
			if err != nil {
				t.Fatal(err)
			}
			if got := file.Source(); !bytes.Equal(got, []byte(tc.src)) {
				t.Fatalf("unwritten source changed:\n%s", got)
			}
		})
	}
}

func TestAstx_ParseAs(t *testing.T) {
	cases := []struct {
		name     string
		language string
		src      string
	}{
		{name: "go", language: "go", src: "package p\n"},
		{name: "typescript", language: "typescript", src: "export const value = 1;\n"},
		{name: "typescript alias", language: "ts", src: "export const value = 1;\n"},
		{name: "case insensitive", language: "TypeScript", src: "export const value = 1;\n"},
		{name: "tsx", language: "tsx", src: "export const view = <div />;\n"},
		{name: "rust", language: "rust", src: "fn main() {}\n"},
		{name: "zig", language: "zig", src: "pub fn main() void {}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAs(tc.language, "stdin", []byte(tc.src)); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := ParseAs("javascript", "stdin", nil); err == nil {
		t.Fatal("an unknown explicit language must fail")
	}
}

func TestAstx_UnsupportedFile(t *testing.T) {
	if Supports("file.js") {
		t.Fatal("an unregistered extension must not be supported")
	}
	if _, err := Parse("file.js", []byte("const value = 1;\n")); err == nil {
		t.Fatal("an unregistered extension must not default to another parser")
	}
}
