package requests

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func messagesByField(invalidFields validator.ValidationErrors) map[string][]string {
	messages := make(map[string][]string, len(invalidFields))
	for _, invalid := range invalidFields {
		field := fieldPath(invalid)
		messages[field] = append(messages[field], message(invalid))
	}

	return messages
}

// fieldPath drops the struct name from the namespace, so nested fields read
// "address.city" and list items "items[0].name".
func fieldPath(invalid validator.FieldError) string {
	_, path, found := strings.Cut(invalid.Namespace(), ".")
	if !found {
		return invalid.Field()
	}

	return path
}

// message never quotes the submitted value, so a response cannot echo back
// what the client sent.
func message(invalid validator.FieldError) string {
	switch invalid.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "uuid", "uuid4", "uuid7":
		return "must be a valid UUID"
	case "url", "http_url":
		return "must be a valid URL"
	case "oneof":
		return "must be one of: " + invalid.Param()
	case "min", "gte":
		return "must be at least " + sized(invalid)
	case "max", "lte":
		return "must be at most " + sized(invalid)
	case "len":
		return "must be exactly " + sized(invalid)
	default:
		return "is invalid"
	}
}

func sized(invalid validator.FieldError) string {
	switch invalid.Kind() {
	case reflect.String:
		return counted(invalid.Param(), "character")
	case reflect.Slice, reflect.Array, reflect.Map:
		return counted(invalid.Param(), "item")
	default:
		return invalid.Param()
	}
}

func counted(count, unit string) string {
	if count == "1" {
		return fmt.Sprintf("%s %s", count, unit)
	}

	return fmt.Sprintf("%s %ss", count, unit)
}
