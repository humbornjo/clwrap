// Package walk expands clwrap's positional args — files, directories,
// and dir/... recursive patterns — into the supported source files they
// name, passing "-" through verbatim for the caller's stdin path.
package walk

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"clwrap/package/astx"
)

// Args calls fn for each input in arg order: a file arg as given, every
// supported source file beneath a directory arg, and "-" unchanged. Each
// fn failure joins into the returned error; walking continues past it.
func Args(args []string, fn func(path string) error) error {
	var errs []error
	for _, arg := range args {
		if arg == "-" {
			errs = append(errs, fn("-"))
			continue
		}
		if root, recursive := strings.CutSuffix(arg, "/..."); recursive {
			if root == "" {
				root = "."
			}
			errs = append(errs, dir(root, fn))
			continue
		}
		info, err := os.Stat(arg)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if info.IsDir() {
			errs = append(errs, dir(arg, fn))
			continue
		}
		errs = append(errs, fn(arg))
	}
	return errors.Join(errs...)
}

// dir calls fn for every supported source file beneath root. Dependency
// and test-fixture directories are skipped, along with dot/underscore
// directories per the go tool convention.
func dir(root string, fn func(path string) error) error {
	var errs []error
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				if name := d.Name(); strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "node_modules" || name == "testdata" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if astx.Supports(path) {
			errs = append(errs, fn(path))
		}
		return nil
	})
	return errors.Join(append(errs, err)...)
}
