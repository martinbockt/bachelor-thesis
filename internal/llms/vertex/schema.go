package vertex

import (
	"reflect"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

func generateSchema(v interface{}) *genai.Schema {
	t := reflect.TypeOf(v)
	schema := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: make(map[string]*genai.Schema),
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("json")
		enum := parseEnumTag(field)

		var fieldType genai.Type
		switch field.Type.Kind() {
		case reflect.String:
			fieldType = genai.TypeString
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldType = genai.TypeInteger
		case reflect.Slice:
			elemKind := field.Type.Elem().Kind()
			var itemsSchema *genai.Schema

			if elemKind == reflect.Struct {
				itemsSchema = generateSchema(reflect.New(field.Type.Elem()).Interface())
			} else {
				var itemType genai.Type
				switch elemKind {
				case reflect.String:
					itemType = genai.TypeString
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					itemType = genai.TypeInteger
				case reflect.Bool:
					itemType = genai.TypeBoolean
				default:
					itemType = genai.TypeString
				}
				itemsSchema = &genai.Schema{
					Type: itemType,
				}
			}

			schema.Properties[fieldName] = &genai.Schema{
				Type:        genai.TypeArray,
				Items:       itemsSchema,
				Description: field.Tag.Get("description"),
			}

			continue
		case reflect.Bool:
			fieldType = genai.TypeBoolean
		default:
			fieldType = genai.TypeString
		}

		if field.Tag.Get("required") == "true" {
			required = append(required, fieldName)
		}

		schema.Properties[fieldName] = &genai.Schema{
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
