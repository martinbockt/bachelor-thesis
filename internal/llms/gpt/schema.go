package gpt

import (
	"reflect"
	"strings"

	"github.com/sashabaranov/go-openai/jsonschema"
)

func generateSchema(v interface{}) *jsonschema.Definition {
	t := reflect.TypeOf(v)
	schema := &jsonschema.Definition{
		Type:       jsonschema.Object,
		Properties: make(map[string]jsonschema.Definition),
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("json")
		enum := parseEnumTag(field)

		var fieldType jsonschema.DataType
		switch field.Type.Kind() {
		case reflect.String:
			fieldType = jsonschema.String
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = jsonschema.Integer
		case reflect.Slice:
			elemKind := field.Type.Elem().Kind()
			var itemsSchema *jsonschema.Definition

			if elemKind == reflect.Struct {
				itemsSchema = generateSchema(reflect.New(field.Type.Elem()).Interface())
			} else {
				var itemType jsonschema.DataType
				switch elemKind {
				case reflect.String:
					itemType = jsonschema.String
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					itemType = jsonschema.Integer
				case reflect.Bool:
					itemType = jsonschema.Boolean
				default:
					itemType = jsonschema.String
				}
				itemsSchema = &jsonschema.Definition{
					Type: itemType,
				}
			}

			schema.Properties[fieldName] = jsonschema.Definition{
				Type:        jsonschema.Array,
				Items:       itemsSchema,
				Description: field.Tag.Get("description"),
			}

			continue
		case reflect.Bool:
			fieldType = jsonschema.Boolean
		default:
			fieldType = jsonschema.String
		}

		if field.Tag.Get("required") == "true" {
			required = append(required, fieldName)
		}

		schema.Properties[fieldName] = jsonschema.Definition{
			Type:        fieldType,
			Description: field.Tag.Get("description"),
			Enum:        enum,
		}
	}
	schema.Required = required

	return schema
}

func parseEnumTag(field reflect.StructField) []string {
	enumTag := field.Tag.Get("enum")
	if enumTag == "" {
		return nil
	}

	return strings.Split(enumTag, ",")
}
