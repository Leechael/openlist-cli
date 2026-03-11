//go:build bdd

package bdd

import (
	"os/exec"
	"strings"
	"testing"
)

func TestListOpsSmoke(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/openlist-cli", "list-ops", "--plain")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "fsList\tPOST\t/api/fs/list") {
		t.Fatalf("unexpected output: %s", out)
	}
}
