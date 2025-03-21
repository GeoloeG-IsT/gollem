package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// JSONSchema represents a JSON Schema document
type JSONSchema struct {
	Type                 string                 `json:"type,omitempty"`
	Properties           map[string]JSONSchema  `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	MinLength            *int                   `json:"minLength,omitempty"`
	MaxLength            *int                   `json:"maxLength,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Title                string                 `json:"title,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	OneOf                []JSONSchema           `json:"oneOf,omitempty"`
	AnyOf                []JSONSchema           `json:"anyOf,omitempty"`
	AllOf                []JSONSchema           `json:"allOf,omitempty"`
	Not                  *JSONSchema            `json:"not,omitempty"`
	Definitions          map[string]JSONSchema  `json:"definitions,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
	Examples             []interface{}          `json:"examples,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	AdditionalItems      interface{}            `json:"additionalItems,omitempty"`
	ExtraProperties      map[string]interface{} `json:"-"`
}

// ValidationError represents an error that occurred during validation
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error at %s: %s", e.Path, e.Message)
}

// Validator validates data against a JSON Schema
type Validator struct {
	schema JSONSchema
}

// NewValidator creates a new validator with the given schema
func NewValidator(schema JSONSchema) *Validator {
	return &Validator{schema: schema}
}

// ValidateJSON validates a JSON string against the schema
func (v *Validator) ValidateJSON(jsonData string) ([]ValidationError, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	return v.Validate(data), nil
}

// Validate validates data against the schema
func (v *Validator) Validate(data interface{}) []ValidationError {
	var errors []ValidationError
	validate(data, v.schema, "", &errors)
	return errors
}

// validate recursively validates data against a schema
func validate(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	// Handle null values
	if data == nil {
		if schema.Type != "null" && schema.Type != "" {
			*errors = append(*errors, ValidationError{
				Path:    path,
				Message: "expected " + schema.Type + ", got null",
			})
		}
		return
	}

	// Handle type validation
	if schema.Type != "" {
		validateType(data, schema, path, errors)
	}

	// Handle object validation
	if schema.Type == "object" || (schema.Type == "" && reflect.TypeOf(data).Kind() == reflect.Map) {
		validateObject(data, schema, path, errors)
	}

	// Handle array validation
	if schema.Type == "array" || (schema.Type == "" && reflect.TypeOf(data).Kind() == reflect.Slice) {
		validateArray(data, schema, path, errors)
	}

	// Handle string validation
	if schema.Type == "string" || (schema.Type == "" && reflect.TypeOf(data).Kind() == reflect.String) {
		validateString(data.(string), schema, path, errors)
	}

	// Handle number validation
	if (schema.Type == "number" || schema.Type == "integer") || 
	   (schema.Type == "" && (reflect.TypeOf(data).Kind() == reflect.Float64 || reflect.TypeOf(data).Kind() == reflect.Int)) {
		validateNumber(data, schema, path, errors)
	}

	// Handle enum validation
	if schema.Enum != nil {
		validateEnum(data, schema, path, errors)
	}
}

// validateType validates that the data matches the expected type
func validateType(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	dataType := reflect.TypeOf(data).Kind().String()
	
	// Map Go types to JSON Schema types
	switch dataType {
	case "map":
		dataType = "object"
	case "slice":
		dataType = "array"
	case "float64":
		dataType = "number"
	case "int":
		dataType = "integer"
	case "string":
		dataType = "string"
	case "bool":
		dataType = "boolean"
	}
	
	if schema.Type != dataType {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: "expected " + schema.Type + ", got " + dataType,
		})
	}
}

// validateObject validates an object against a schema
func validateObject(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	// Type assertion
	obj, ok := data.(map[string]interface{})
	if !ok {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: "expected object, got " + reflect.TypeOf(data).String(),
		})
		return
	}
	
	// Check required properties
	for _, req := range schema.Required {
		if _, ok := obj[req]; !ok {
			*errors = append(*errors, ValidationError{
				Path:    path,
				Message: "missing required property: " + req,
			})
		}
	}
	
	// Validate properties
	for name, prop := range schema.Properties {
		if val, ok := obj[name]; ok {
			propPath := path
			if path == "" {
				propPath = name
			} else {
				propPath = path + "." + name
			}
			validate(val, prop, propPath, errors)
		}
	}
	
	// Check additional properties
	if schema.AdditionalProperties != nil {
		// If additionalProperties is false, no additional properties are allowed
		if b, ok := schema.AdditionalProperties.(bool); ok && !b {
			for name := range obj {
				if _, ok := schema.Properties[name]; !ok {
					*errors = append(*errors, ValidationError{
						Path:    path,
						Message: "additional property not allowed: " + name,
					})
				}
			}
		}
		
		// If additionalProperties is a schema, validate additional properties against it
		if s, ok := schema.AdditionalProperties.(JSONSchema); ok {
			for name, val := range obj {
				if _, ok := schema.Properties[name]; !ok {
					propPath := path
					if path == "" {
						propPath = name
					} else {
						propPath = path + "." + name
					}
					validate(val, s, propPath, errors)
				}
			}
		}
	}
}

// validateArray validates an array against a schema
func validateArray(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	// Type assertion
	arr, ok := data.([]interface{})
	if !ok {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: "expected array, got " + reflect.TypeOf(data).String(),
		})
		return
	}
	
	// Validate items
	if schema.Items != nil {
		for i, item := range arr {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			validate(item, *schema.Items, itemPath, errors)
		}
	}
}

// validateString validates a string against a schema
func validateString(data string, schema JSONSchema, path string, errors *[]ValidationError) {
	// Check minLength
	if schema.MinLength != nil && len(data) < *schema.MinLength {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d is less than minimum %d", len(data), *schema.MinLength),
		})
	}
	
	// Check maxLength
	if schema.MaxLength != nil && len(data) > *schema.MaxLength {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("string length %d is greater than maximum %d", len(data), *schema.MaxLength),
		})
	}
	
	// Check pattern (not implemented in this simplified version)
	// Would require a regex library
}

// validateNumber validates a number against a schema
func validateNumber(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	var num float64
	
	// Convert to float64 for comparison
	switch v := data.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	default:
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: "expected number, got " + reflect.TypeOf(data).String(),
		})
		return
	}
	
	// Check minimum
	if schema.Minimum != nil && num < *schema.Minimum {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("value %f is less than minimum %f", num, *schema.Minimum),
		})
	}
	
	// Check maximum
	if schema.Maximum != nil && num > *schema.Maximum {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("value %f is greater than maximum %f", num, *schema.Maximum),
		})
	}
	
	// Check integer type
	if schema.Type == "integer" && float64(int(num)) != num {
		*errors = append(*errors, ValidationError{
			Path:    path,
			Message: fmt.Sprintf("value %f is not an integer", num),
		})
	}
}

// validateEnum validates that the data is one of the enum values
func validateEnum(data interface{}, schema JSONSchema, path string, errors *[]ValidationError) {
	for _, enum := range schema.Enum {
		if reflect.DeepEqual(data, enum) {
			return
		}
	}
	
	*errors = append(*errors, ValidationError{
		Path:    path,
		Message: "value does not match any enum value",
	})
}

// GenerateSchema generates a JSON Schema from a Go struct
func GenerateSchema(v interface{}) (JSONSchema, error) {
	t := reflect.TypeOf(v)
	
	// If v is a pointer, get the underlying type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	// Only structs are supported
	if t.Kind() != reflect.Struct {
		return JSONSchema{}, errors.New("only structs are supported")
	}
	
	return generateSchemaFromType(t), nil
}

// generateSchemaFromType generates a JSON Schema from a reflect.Type
func generateSchemaFromType(t reflect.Type) JSONSchema {
	schema := JSONSchema{}
	
	switch t.Kind() {
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]JSONSchema)
		var required []string
		
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			
			// Skip unexported fields
			if field.PkgPath != "" {
				continue
			}
			
			// Get the JSON tag
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			
			// Parse the JSON tag
			name, opts := parseTag(tag)
			if name == "" {
				name = field.Name
			}
			
			// Generate schema for the field
			fieldSchema := generateSchemaFromType(field.Type)
			
			// Add description from doc tag
			if doc := field.Tag.Get("doc"); doc != "" {
				fieldSchema.Description = doc
			}
			
			// Check if the field is required
			if !opts.Contains("omitempty") {
				required = append(required, name)
			}
			
			schema.Properties[name] = fieldSchema
		}
		
		if len(required) > 0 {
			schema.Required = required
		}
		
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
		schema.Items = new(JSONSchema)
		*schema.Items = generateSchemaFromType(t.Elem())
		
	case reflect.Map:
		schema.Type = "object"
		schema.AdditionalProperties = generateSchemaFromType(t.Elem())
		
	case reflect.String:
		schema.Type = "string"
		
	case reflect.Bool:
		schema.Type = "boolean"
		
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
		
	case reflect.Interface:
		// For interface{}, we don't specify a type
		
	case reflect.Ptr:
		return generateSchemaFromType(t.Elem())
	}
	
	return schema
}

// tagOptions represents options in a JSON tag
type tagOptions string

// Contains checks if the options contain the specified option
func (o tagOptions) Contains(option string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var next string
		i := strings.Index(s, ",")
		if i >= 0 {
			s, next = s[:i], s[i+1:]
		}
		if s == option {
			return true
		}
		s = next
	}
	return false
}

// parseTag parses a JSON tag into a name and options
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, ""
}
