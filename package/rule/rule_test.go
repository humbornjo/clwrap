package rule

import (
	"strings"
	"testing"

	"gommentlw/package/astx"
)

// pipeString runs one source fragment through the stream pipeline,
// injecting a package clause when the fragment does not carry one.
func pipeString(t *testing.T, src string, shortW, longW, listW int) string {
	t.Helper()
	full := strings.HasPrefix(src, "package ") || strings.Contains(src, "\npackage ")
	if !full {
		src = "package p\n\n" + src
	}
	file, err := astx.ParseFile("x.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	rules := Set{List{Width: listW}, Width{Decl: shortW, Inner: longW, Tolerance: 2}}
	if err := Pipe(file, rules); err != nil {
		t.Fatal(err)
	}
	text := string(file.Source())
	if !full {
		text = strings.TrimPrefix(text, "package p\n\n")
	}
	return text
}

func TestRule_Pipe(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		want  string
		short int
		long  int
		list  int
	}{
		{
			name: "decl doc wraps short",
			src: "// Run documents the declaration in one long line that must wrap at the short width boundary.\n" +
				"func Run() {}\n",
			want: "// Run documents the declaration in one long line that must\n" +
				"// wrap at the short width boundary.\n" +
				"func Run() {}\n",
			short: 60,
			long:  100,
		},
		{
			name: "decl doc tolerance holds a straddling word",
			src: "// Run documents the declaration in one long line of w words.\n" +
				"func Run() {}\n",
			want: "// Run documents the declaration in one long line of w words.\n" +
				"func Run() {}\n",
			short: 60,
			long:  100,
		},
		{
			// The word "as" would end within the two-column allowance but starts past the boundary: a full word in the tolerance
			// zone wraps instead of riding it.
			name: "tolerance refuses a full word past the boundary",
			src: "// Run documents it as follows.\n" +
				"func Run() {}\n",
			want: "// Run documents it\n" +
				"// as follows.\n" +
				"func Run() {}\n",
			short: 20,
			long:  100,
		},
		{
			name: "tolerance lets a straddling word over",
			src: "// Run documents this as follows.\n" +
				"func Run() {}\n",
			want: "// Run documents this\n" +
				"// as follows.\n" +
				"func Run() {}\n",
			short: 20,
			long:  100,
		},
		{
			name: "decl doc joins narrow lines",
			src: "// Run documents the declaration\n" +
				"// in narrowly wrapped lines that\n" +
				"// the tool joins.\n" +
				"func Run() {}\n",
			want: "// Run documents the declaration in narrowly wrapped lines\n" +
				"// that the tool joins.\n" +
				"func Run() {}\n",
			short: 60,
			long:  100,
		},
		{
			name: "body comment wraps long",
			src: "func Run() {\n" +
				"\t// A body comment reflows to the longer width, staying one line past the short width.\n" +
				"}\n",
			want: "func Run() {\n" +
				"\t// A body comment reflows to the longer width, staying one line past the short width.\n" +
				"}\n",
			short: 60,
			long:  100,
		},
		{
			name: "field doc joins to long",
			src: "type C struct {\n" +
				"\t// Path is a field doc comment\n" +
				"\t// written narrowly by hand.\n" +
				"\tPath string\n" +
				"}\n",
			want: "type C struct {\n" +
				"\t// Path is a field doc comment written narrowly by hand.\n" +
				"\tPath string\n" +
				"}\n",
			short: 60,
			long:  100,
		},
		{
			name: "grouped const spec doc is short",
			src: "const (\n" +
				"\t// A grouped const spec is still a package-level named declaration, so its doc comment wraps short.\n" +
				"\tA = 1\n" +
				")\n",
			want: "const (\n" +
				"\t// A grouped const spec is still a package-level named\n" +
				"\t// declaration, so its doc comment wraps short.\n" +
				"\tA = 1\n" +
				")\n",
			short: 60,
			long:  100,
		},
		{
			name: "grouped type spec doc is short",
			src: "type (\n" +
				"\t// C is a named type of the package even inside a grouped declaration, so its doc comment wraps short.\n" +
				"\tC struct{}\n" +
				")\n",
			want: "type (\n" +
				"\t// C is a named type of the package even inside a grouped\n" +
				"\t// declaration, so its doc comment wraps short.\n" +
				"\tC struct{}\n" +
				")\n",
			short: 60,
			long:  100,
		},
		{
			name:  "directive group untouched",
			src:   "func Run() {\n\t//go:generate echo this stays exactly as written even when it runs past any width at all\n}\n",
			want:  "func Run() {\n\t//go:generate echo this stays exactly as written even when it runs past any width at all\n}\n",
			short: 60,
			long:  100,
		},
		{
			name: "directive splits the doc group",
			src: "// D embeds the asset for the binary build, produced by the make step.\n" +
				"//\n" +
				"//go:embed all:dist\n" +
				"var D embed.FS\n",
			want: "// D embeds the asset for the binary build, produced by the\n" +
				"// make step.\n" +
				"//\n" +
				"//go:embed all:dist\n" +
				"var D embed.FS\n",
			short: 60,
			long:  100,
		},
		{
			name:  "trailing comment untouched",
			src:   "func Run() {\n\tx := 1 // a trailing comment keeps its place even when it runs past any width at all\n\t_ = x\n}\n",
			want:  "func Run() {\n\tx := 1 // a trailing comment keeps its place even when it runs past any width at all\n\t_ = x\n}\n",
			short: 60,
			long:  100,
		},
		{
			name: "list items keep their layout",
			src: "// List does things.\n" +
				"//\n" +
				"//   - first item keeps its layout\n" +
				"//   - second item keeps its layout\n" +
				"func List() {}\n",
			want: "// List does things.\n" +
				"//\n" +
				"//   - first item keeps its layout\n" +
				"//   - second item keeps its layout\n" +
				"func List() {}\n",
			short: 60,
			long:  100,
		},
		{
			name: "list item joins its continuation past the tier width",
			src: "// List does things.\n" +
				"//\n" +
				"//   - first item spread over\n" +
				"//     two lines joins past the tier width\n" +
				"//   - second item\n" +
				"func List() {}\n",
			want: "// List does things.\n" +
				"//\n" +
				"//   - first item spread over two lines joins past the tier width\n" +
				"//   - second item\n" +
				"func List() {}\n",
			short: 60,
			long:  100,
		},
		{
			name: "list item wraps at the list constraint",
			src: "//   - first item spread over two lines joins\n" +
				"func f() {}\n",
			want: "//   - first item spread over\n" +
				"//     two lines joins\n" +
				"func f() {}\n",
			short: 60,
			long:  100,
			list:  30,
		},
		{
			name: "prose-led paragraph keeps its layout",
			src: "// Does things:\n" +
				"//   - a sub item\n" +
				"func f() {}\n",
			want: "// Does things:\n" +
				"//   - a sub item\n" +
				"func f() {}\n",
			short: 60,
			long:  100,
		},
		{
			name: "package doc wraps short",
			src: "// Package sample documents the package clause in one long line that must wrap at the short width boundary.\n" +
				"package sample\n",
			want: "// Package sample documents the package clause in one long\n" +
				"// line that must wrap at the short width boundary.\n" +
				"package sample\n",
			short: 60,
			long:  100,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pipeString(t, tc.src, tc.short, tc.long, tc.list)
			if got != tc.want {
				t.Fatalf("pipe:\n%s\nwant:\n%s", got, tc.want)
			}
			if again := pipeString(t, got, tc.short, tc.long, tc.list); again != got {
				t.Fatalf("not idempotent:\n%s\nthen:\n%s", got, again)
			}
		})
	}
}
