package schema

import (
	"encoding/json"
	"testing"
)

type SimpleStruct struct {
	Name string `json:"name" jsonschema:"required,description=The name"`
	Age  int    `json:"age" jsonschema:"required,minimum=0"`
}

type StructWithOptional struct {
	Required string `json:"required" jsonschema:"required"`
	Optional string `json:"optional,omitempty"`
}

type StructWithNested struct {
	Name    string      `json:"name" jsonschema:"required"`
	Details NestedEntry `json:"details" jsonschema:"required"`
}

type NestedEntry struct {
	ID    int    `json:"id" jsonschema:"required"`
	Value string `json:"value"`
}

type StructWithEnum struct {
	Status string `json:"status" jsonschema:"required,enum=active,enum=inactive,enum=pending"`
}

type StructWithSlice struct {
	Tags []string `json:"tags" jsonschema:"description=List of tags"`
}

func TestFromStruct(t *testing.T) {
	s := FromStruct[SimpleStruct]()
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	if s.Properties.Len() != 2 {
		t.Errorf("expected 2 properties, got %d", s.Properties.Len())
	}

	nameProp, ok := s.Properties.Get("name")
	if !ok {
		t.Fatal("expected name property")
	}
	if nameProp.Description != "The name" {
		t.Errorf("expected description 'The name', got %q", nameProp.Description)
	}
}

func TestFromStructWithOptional(t *testing.T) {
	s := FromStruct[StructWithOptional]()
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	if len(s.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(s.Required))
	}
}

func TestFromStructWithNested(t *testing.T) {
	s := FromStruct[StructWithNested]()
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	detailsProp, ok := s.Properties.Get("details")
	if !ok {
		t.Fatal("expected details property")
	}
	if detailsProp == nil {
		t.Fatal("expected details property to be non-nil")
	}
}

func TestFromStructWithEnum(t *testing.T) {
	s := FromStruct[StructWithEnum]()
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	statusProp, ok := s.Properties.Get("status")
	if !ok {
		t.Fatal("expected status property")
	}
	if len(statusProp.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(statusProp.Enum))
	}
}

func TestFromStructWithSlice(t *testing.T) {
	s := FromStruct[StructWithSlice]()
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	tagsProp, ok := s.Properties.Get("tags")
	if !ok {
		t.Fatal("expected tags property")
	}
	if tagsProp.Type != "array" {
		t.Errorf("expected array type, got %q", tagsProp.Type)
	}
	if tagsProp.Items == nil {
		t.Fatal("expected items for array")
	}
	if tagsProp.Items.Type != "string" {
		t.Errorf("expected string items, got %q", tagsProp.Items.Type)
	}
}

func TestFromStructWithExamples(t *testing.T) {
	examples := []SimpleStruct{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}
	s := FromStructWithExamples[SimpleStruct](examples...)
	if s == nil {
		t.Fatal("expected non-nil schema")
	}

	if len(s.Examples) != 2 {
		t.Errorf("expected 2 examples, got %d", len(s.Examples))
	}
}

func TestToRawMessage(t *testing.T) {
	s := FromStruct[SimpleStruct]()
	raw, err := ToRawMessage(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}

	props, ok := m["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties object")
	}
	if len(props) != 2 {
		t.Errorf("expected 2 properties in raw JSON, got %d", len(props))
	}
}

func TestMustToRawMessage(t *testing.T) {
	s := FromStruct[SimpleStruct]()
	raw := MustToRawMessage(s)
	if len(raw) == 0 {
		t.Error("expected non-empty raw message")
	}
}
