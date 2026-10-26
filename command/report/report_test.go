package cmdreport

import (
	"bytes"
	"strings"
	"testing"
)

func TestCmdReport_Render(t *testing.T) {
	stats := map[string]*stat{
		"changed":    {files: 1, dirty: 1, before: 4, after: 2},
		"same lines": {files: 1, dirty: 1, before: 3, after: 3},
		"unchanged":  {files: 2, before: 5, after: 5},
	}
	var out bytes.Buffer
	render(&out, stats, 0, 80, false)
	got := out.String()
	if strings.Contains(got, "unchanged") {
		t.Fatalf("report contains an unchanged directory:\n%s", got)
	}
	for _, name := range []string{"changed", "same lines"} {
		if !strings.Contains(got, name) {
			t.Errorf("report does not contain changed directory %q:\n%s", name, got)
		}
	}
}
