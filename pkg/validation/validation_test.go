package validation_test

import (
	"testing"

	"github.com/GeoloeG-IsT/gollem/pkg/validation"
)

// TestJSONSchemaValidation tests the JSON schema validation
func TestJSONSchemaValidation(t *testing.T) {
	// Create a schema
	schema := validation.JSONSchema{
		Type: "object",
		Properties: map[string]validation.JSONSchema{
			"name": {
				Type:     "string",
				MinLength: intPtr(1),
			},
			"age": {
				Type:    "integer",
				Minimum: float64Ptr(0),
				Maximum: float64Ptr(120),
			},
			"email": {
				Type:   "string",
				Format: "email",
			},
		},
		Required: []string{"name", "age"},
	}

	// Create a validator
	validator := validation.NewValidator(schema)

	// Test valid data
	validJSON := `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com"
	}`

	errors, err := validator.ValidateJSON(validJSON)
	if err != nil {
		t.Fatalf("Failed to validate JSON: %v", err)
	}
	if len(errors) > 0 {
		t.Fatalf("Validation errors for valid data: %v", errors)
	}

	// Test invalid data (missing required field)
	invalidJSON1 := `{
		"name": "John Doe"
	}`

	errors, err = validator.ValidateJSON(invalidJSON1)
	if err != nil {
		t.Fatalf("Failed to validate JSON: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("No validation errors for invalid data (missing required field)")
	}

	// Test invalid data (wrong type)
	invalidJSON2 := `{
		"name": "John Doe",
		"age": "thirty"
	}`

	errors, err = validator.ValidateJSON(invalidJSON2)
	if err != nil {
		t.Fatalf("Failed to validate JSON: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("No validation errors for invalid data (wrong type)")
	}

	// Test invalid data (out of range)
	invalidJSON3 := `{
		"name": "John Doe",
		"age": 150
	}`

	errors, err = validator.ValidateJSON(invalidJSON3)
	if err != nil {
		t.Fatalf("Failed to validate JSON: %v", err)
	}
	if len(errors) == 0 {
		t.Fatal("No validation errors for invalid data (out of range)")
	}

	// Test invalid JSON
	invalidJSON4 := `{
		"name": "John Doe",
		"age": 30,
		"email": "john@example.com"
	`

	_, err = validator.ValidateJSON(invalidJSON4)
	if err == nil {
		t.Fatal("No error for invalid JSON")
	}
}

// TestSchemaGeneration tests the schema generation from Go structs
func TestSchemaGeneration(t *testing.T) {
	// Define a test struct
	type Address struct {
		Street  string `json:"street"`
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Email   string  `json:"email,omitempty"`
		Address Address `json:"address"`
	}

	// Generate a schema
	schema, err := validation.GenerateSchema(Person{})
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Check the schema
	if schema.Type != "object" {
		t.Fatalf("Schema type is incorrect: %s", schema.Type)
	}

	// Check properties
	if len(schema.Properties) != 4 {
		t.Fatalf("Schema has %d properties, expected 4", len(schema.Properties))
	}

	// Check name property
	nameProp, ok := schema.Properties["name"]
	if !ok {
		t.Fatal("Schema missing 'name' property")
	}
	if nameProp.Type != "string" {
		t.Fatalf("Name property type is incorrect: %s", nameProp.Type)
	}

	// Check age property
	ageProp, ok := schema.Properties["age"]
	if !ok {
		t.Fatal("Schema missing 'age' property")
	}
	if ageProp.Type != "integer" {
		t.Fatalf("Age property type is incorrect: %s", ageProp.Type)
	}

	// Check address property
	addressProp, ok := schema.Properties["address"]
	if !ok {
		t.Fatal("Schema missing 'address' property")
	}
	if addressProp.Type != "object" {
		t.Fatalf("Address property type is incorrect: %s", addressProp.Type)
	}
	if len(addressProp.Properties) != 3 {
		t.Fatalf("Address property has %d properties, expected 3", len(addressProp.Properties))
	}

	// Check required fields
	if len(schema.Required) != 3 {
		t.Fatalf("Schema has %d required fields, expected 3", len(schema.Required))
	}
	hasName := false
	hasAge := false
	hasAddress := false
	for _, field := range schema.Required {
		switch field {
		case "name":
			hasName = true
		case "age":
			hasAge = true
		case "address":
			hasAddress = true
		}
	}
	if !hasName || !hasAge || !hasAddress {
		t.Fatal("Schema missing required fields")
	}
}

// Helper functions to create pointers
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
