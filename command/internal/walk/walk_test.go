package walk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWalk_Args(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"a.go",
		"b.ts",
		"c.tsx",
		"d.mts",
		"e.cts",
		"rust.rs",
		"zig.zig",
		"ignore.js",
		"README.md",
		"nested/f.go",
		"node_modules/dependency.ts",
		"vendor/dependency.go",
		"testdata/fixture.ts",
		".hidden/hidden.go",
		"_private/private.ts",
	}
	for _, name := range paths {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var got []string
	if err := Args([]string{root}, func(path string) error {
		got = append(got, path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "a.go"),
		filepath.Join(root, "b.ts"),
		filepath.Join(root, "c.tsx"),
		filepath.Join(root, "d.mts"),
		filepath.Join(root, "e.cts"),
		filepath.Join(root, "rust.rs"),
		filepath.Join(root, "zig.zig"),
		filepath.Join(root, "nested/f.go"),
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("walked files = %v, want %v", got, want)
	}
}
