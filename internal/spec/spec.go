package spec

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed openapi.json
var rawOpenAPI []byte

type Document struct {
	Info     Info                            `json:"info"`
	Paths    map[string]map[string]Operation `json:"paths"`
	Security []map[string][]string           `json:"security"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Operation struct {
	OperationID string                `json:"operationId"`
	Summary     string                `json:"summary"`
	Description string                `json:"description"`
	Tags        []string              `json:"tags"`
	Security    []map[string][]string `json:"security"`
	RequestBody map[string]any        `json:"requestBody"`
}

type OperationSpec struct {
	Tag                 string `json:"tag"`
	OperationID         string `json:"operation_id"`
	Method              string `json:"method"`
	Path                string `json:"path"`
	Summary             string `json:"summary"`
	SecurityRequired    bool   `json:"security_required"`
	RequestBodyRequired bool   `json:"request_body_required"`
}

func Load() (*Document, error) {
	var doc Document
	if err := json.Unmarshal(rawOpenAPI, &doc); err != nil {
		return nil, fmt.Errorf("parse embedded openapi: %w", err)
	}
	return &doc, nil
}

func Operations(doc *Document) []OperationSpec {
	ops := make([]OperationSpec, 0)
	for path, methods := range doc.Paths {
		for method, op := range methods {
			if !isHTTPMethod(method) {
				continue
			}
			tag := "default"
			if len(op.Tags) > 0 && strings.TrimSpace(op.Tags[0]) != "" {
				tag = op.Tags[0]
			}
			operationID := strings.TrimSpace(op.OperationID)
			if operationID == "" {
				operationID = fallbackID(method, path)
			}
			summary := strings.TrimSpace(op.Summary)
			if summary == "" {
				summary = strings.TrimSpace(op.Description)
			}
			securityRequired := requiresSecurity(op.Security, doc.Security)
			ops = append(ops, OperationSpec{
				Tag:                 tag,
				OperationID:         operationID,
				Method:              strings.ToUpper(method),
				Path:                path,
				Summary:             summary,
				SecurityRequired:    securityRequired,
				RequestBodyRequired: len(op.RequestBody) > 0,
			})
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Tag != ops[j].Tag {
			return ops[i].Tag < ops[j].Tag
		}
		return ops[i].OperationID < ops[j].OperationID
	})
	return ops
}

func FindOperation(doc *Document, operationID string) (OperationSpec, bool) {
	for _, op := range Operations(doc) {
		if op.OperationID == operationID {
			return op, true
		}
	}
	return OperationSpec{}, false
}

func fallbackID(method, path string) string {
	s := strings.ToLower(method + "_" + path)
	var b strings.Builder
	underscore := false
	for _, r := range s {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			underscore = false
			continue
		}
		if !underscore {
			b.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func requiresSecurity(operationSecurity, rootSecurity []map[string][]string) bool {
	security := operationSecurity
	if len(security) == 0 {
		security = rootSecurity
	}
	if len(security) == 0 {
		return false
	}
	for _, item := range security {
		if len(item) == 0 {
			return false
		}
	}
	return true
}

func isHTTPMethod(method string) bool {
	switch strings.ToLower(method) {
	case "get", "post", "put", "patch", "delete", "head", "options":
		return true
	default:
		return false
	}
}
