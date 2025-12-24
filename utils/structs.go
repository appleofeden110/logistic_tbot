package utils

import (
	"fmt"
	"reflect"
	"strings"
)

func validateStructTypes(data reflect.Type, fields ...string) error {
	if data.Kind() == reflect.Ptr {
		data = data.Elem()
	}
	if data.Kind() != reflect.Struct {
		return fmt.Errorf("data must be a struct, got %v", data.Kind())
	}
	for i := 0; i < data.NumField(); i++ {
		field := data.Field(i)
		fieldType := field.Type
		kind := fieldType.Kind()
		for i, v := range fields {
			if v == field.Name {
				fields = append(fields[:i], fields[i+1:]...)
			}
			fmt.Println(fields)
		}

		if kind == reflect.Ptr {
			fieldType = fieldType.Elem()
			kind = fieldType.Kind()
		}

		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Float32, reflect.Float64, reflect.Bool,
			reflect.String:
		case reflect.Array:
			typeName := fieldType.String()
			if typeName == "uuid.UUID" || fieldType.Name() == "UUID" {
				continue
			}
			return fmt.Errorf("field %s has invalid array type %v", field.Name, field.Type)
		case reflect.Struct:
			typeName := fieldType.Name()
			switch typeName {
			case "Time", "User", "PendingMessage":
				continue
			default:
				return fmt.Errorf("field %s has invalid type %v (only int, float, string, or allowed structs)",
					field.Name, field.Type)
			}
		default:
			return fmt.Errorf("field %s has invalid type %v (only int, float, string allowed)",
				field.Name, field.Type)
		}
	}
	if len(fields) > 0 {
		return fmt.Errorf("Certain fields are not actually in the struct: %v\n", fields)
	}
	return nil
}

// should be a struct with a `form:""` beside the field
func GetFormTagValue[T any](formStruct T, field string) (string, error) {
	t := reflect.TypeOf(formStruct)
	err := validateStructTypes(t, field)
	if err != nil {
		return "", err
	}

	// Check if field contains dot notation (e.g., "User.Name")
	parts := strings.Split(field, ".")
	currentType := t

	for i, part := range parts {
		structField, found := currentType.FieldByName(part)
		if !found {
			availableFields := make([]string, currentType.NumField())
			for j := 0; j < currentType.NumField(); j++ {
				availableFields[j] = currentType.Field(j).Name
			}
			return "", fmt.Errorf("field %s not found at level %d, available fields: %v\n", part, i, availableFields)
		}

		// If this is the last part, return the tag
		if i == len(parts)-1 {
			return structField.Tag.Get("form"), nil
		}

		// Otherwise, continue into the nested struct
		currentType = structField.Type
		if currentType.Kind() == reflect.Ptr {
			currentType = currentType.Elem()
		}
		if currentType.Kind() != reflect.Struct {
			return "", fmt.Errorf("field %s is not a struct, cannot traverse further", part)
		}
	}

	return "", fmt.Errorf("unexpected error traversing field path")
}

// same as with GetFormTagValue, but gets you a map of ["fieldName"]: "value of the tag"
// Now recursively traverses nested structs
func GetAllFormTags[T any](formStruct T) (map[string]string, error) {
	t := reflect.TypeOf(formStruct)
	err := validateStructTypes(t)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	collectFormTags(t, "", tags)
	return tags, nil
}

// Helper function to recursively collect form tags
func collectFormTags(t reflect.Type, prefix string, tags map[string]string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Build the field path
		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		// Check if this field has a form tag
		if value, ok := field.Tag.Lookup("form"); ok && len(value) > 0 {
			tags[fieldPath] = value
		}

		// If the field is a struct (or pointer to struct), recurse into it
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			// Skip time.Time and uuid.UUID to avoid infinite recursion
			if fieldType.PkgPath() == "time" || fieldType.PkgPath() == "github.com/gofrs/uuid" {
				continue
			}
			collectFormTags(fieldType, fieldPath, tags)
		}
	}
}
