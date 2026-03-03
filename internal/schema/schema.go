package schema

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// FromStruct generates a JSON Schema from a struct type T.
// The struct should use json tags for field names and jsonschema tags
// for schema properties like required, description, enum, minimum, etc.
//
// Example:
//
//	type Person struct {
//	    Name string `json:"name" jsonschema:"required,description=The person's name"`
//	    Age  int    `json:"age" jsonschema:"required,minimum=0,description=The person's age"`
//	}
//
//	schema := FromStruct[Person]()
func FromStruct[T any]() *jsonschema.Schema {
	var zero T
	r := new(jsonschema.Reflector)
	r.ExpandedStruct = true
	return r.Reflect(&zero)
}

// FromStructWithExamples generates a JSON Schema with examples.
// Examples are included in the schema's "examples" field.
func FromStructWithExamples[T any](examples ...T) *jsonschema.Schema {
	var zero T
	r := new(jsonschema.Reflector)
	r.ExpandedStruct = true
	s := r.Reflect(&zero)
	if len(examples) > 0 {
		s.Examples = make([]any, len(examples))
		for i, ex := range examples {
			s.Examples[i] = ex
		}
	}
	return s
}

// ToRawMessage converts a JSON Schema to json.RawMessage for use in API requests.
func ToRawMessage(s *jsonschema.Schema) (json.RawMessage, error) {
	return json.Marshal(s)
}

// MustToRawMessage converts a JSON Schema to json.RawMessage, panicking on error.
// Use this for schemas that are known to be valid.
func MustToRawMessage(s *jsonschema.Schema) json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}
