package app

import (
	"strings"
	"testing"

	"github.com/openlist/openlist-cli/internal/spec"
)

func TestBuildRouteURL(t *testing.T) {
	got, err := buildRouteURL("http://localhost:5244", "/d", "/阿里云盘/file name.ts", map[string]string{"sign": "abc:0"})
	if err != nil {
		t.Fatal(err)
	}
	want := "http://localhost:5244/d/%E9%98%BF%E9%87%8C%E4%BA%91%E7%9B%98/file%20name.ts?sign=abc%3A0"
	if got != want {
		t.Fatalf("want %s, got %s", want, got)
	}
}

func TestResolveOutputModeRequiresJSONForJQ(t *testing.T) {
	_, err := resolveOutputMode(false, true, ".body")
	if err == nil || !strings.Contains(err.Error(), "--jq requires --json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmbeddedSpecHasFsList(t *testing.T) {
	doc, err := spec.Load()
	if err != nil {
		t.Fatal(err)
	}
	op, ok := spec.FindOperation(doc, "fsList")
	if !ok {
		t.Fatal("fsList not found")
	}
	if op.Method != "POST" || op.Path != "/api/fs/list" {
		t.Fatalf("unexpected op: %+v", op)
	}
}

func TestNormalizeAuthToken(t *testing.T) {
	cases := map[string]string{
		"abc123":        "abc123",
		"Bearer abc123": "abc123",
		"bearer abc123": "abc123",
		"  abc123  ":    "abc123",
	}
	for in, want := range cases {
		if got := normalizeAuthToken(in); got != want {
			t.Fatalf("normalizeAuthToken(%q) = %q, want %q", in, got, want)
		}
	}
}
