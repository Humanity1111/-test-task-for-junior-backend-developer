package docs

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestOpenAPISpecIsValidSingleJSONDocument(t *testing.T) {
	h := NewHandler()

	dec := json.NewDecoder(bytes.NewReader(h.spec))

	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode openapi spec: %v", err)
	}

	if version, ok := doc["openapi"].(string); !ok || version == "" {
		t.Fatalf("openapi version field is missing or invalid")
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		t.Fatalf("openapi spec must contain exactly one JSON document")
	}
}
