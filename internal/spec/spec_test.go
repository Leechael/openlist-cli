package spec

import "testing"

func TestRequiresSecurity(t *testing.T) {
	if requiresSecurity([]map[string][]string{{"BearerAuth": {}}, {}}, nil) {
		t.Fatal("optional auth should not be treated as required")
	}
	if !requiresSecurity([]map[string][]string{{"BearerAuth": {}}}, nil) {
		t.Fatal("explicit bearer auth should be treated as required")
	}
	if requiresSecurity(nil, nil) {
		t.Fatal("no security should not be treated as required")
	}
}
