package jamba

import (
	"reflect"
	"strings"
)

func generateSchemaMap(v interface{}) map[string]any {
	t := reflect.TypeOf(v)
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	required := []string{}
	properties := schema["properties"].(map[string]any)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Name // fallback to the field name if no json tag is provided
		}
		enum := parseEnumTag(field)

		var fieldType string
		switch field.Type.Kind() {
		case reflect.String:
			fieldType = "string"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = "integer"
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.Struct {
				nestedSchema := generateSchemaMap(reflect.New(field.Type.Elem()).Elem().Interface())
				properties[fieldName] = map[string]any{
					"type":        "array",
					"items":       nestedSchema,
					"description": field.Tag.Get("description"),
				}

				continue
			}
			fieldType = "array"
		case reflect.Bool:
			fieldType = "boolean"
		default:
			fieldType = "string"
		}

		property := map[string]any{
			"type":        fieldType,
			"description": field.Tag.Get("description"),
		}

		if len(enum) > 0 {
			property["enum"] = enum
		}

		if field.Tag.Get("required") == "true" {
			required = append(required, fieldName)
		}

		properties[fieldName] = property
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func parseEnumTag(field reflect.StructField) []string {
	enumTag := field.Tag.Get("enum")
	if enumTag == "" {
		return nil
	}

	return strings.Split(enumTag, ",")
}
