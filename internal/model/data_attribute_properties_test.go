package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDataAttributePropertiesJSONIncludesExtensibleList(t *testing.T) {
	marshaled, err := json.Marshal(DataAttributeProperties{})
	if err != nil {
		t.Fatalf("marshal DataAttributeProperties: %v", err)
	}

	var properties map[string]any
	if err := json.Unmarshal(marshaled, &properties); err != nil {
		t.Fatalf("unmarshal DataAttributeProperties JSON: %v", err)
	}

	value, ok := properties["extensibleList"]
	if !ok {
		t.Fatalf("expected extensibleList key in marshaled JSON, got %v", properties)
	}

	boolValue, ok := value.(bool)
	if !ok {
		t.Fatalf("expected extensibleList to be bool, got %T", value)
	}

	if boolValue != false {
		t.Fatalf("expected extensibleList false, got %v", boolValue)
	}
}

func TestOpenAPISchemasDefineExtensibleList(t *testing.T) {
	const (
		startMarker   = "    DataAttributeProperties:"
		endMarker     = "\n    DateAttributeContent:"
		requiredEntry = "      - extensibleList"
		propertyBlock = `        extensibleList:
          type: boolean
          description: Boolean determining if a list Attribute can have values other than predefined options
          default: false`
	)

	for _, fileName := range []string{
		"doc-openapi-connector-authority-provider-v2.yaml",
		"doc-openapi-connector-discovery-provider.yaml",
	} {
		t.Run(fileName, func(t *testing.T) {
			path := filepath.Join("..", "..", "api", "connector-api", fileName)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
			start := strings.Index(normalized, startMarker)
			if start == -1 {
				t.Fatalf("missing %q section start in %s", startMarker, fileName)
			}

			end := strings.Index(normalized[start:], endMarker)
			if end == -1 {
				t.Fatalf("missing %q section end in %s", endMarker, fileName)
			}

			section := normalized[start : start+end]

			if !strings.Contains(section, requiredEntry) {
				t.Fatalf("section missing required entry %q in %s", requiredEntry, fileName)
			}

			if !strings.Contains(section, propertyBlock) {
				t.Fatalf("section missing extensibleList property block in %s", fileName)
			}
		})
	}
}
