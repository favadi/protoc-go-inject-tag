package main

import (
	"strings"

	"github.com/iancoleman/strcase"
)

// DefaultTagInjectType controls where default tags are placed relative to existing tags.
type DefaultTagInjectType string

const (
	DefaultTagPrepend DefaultTagInjectType = "prepend"
	DefaultTagAppend  DefaultTagInjectType = "append"
)

// DefaultTagOptions holds the CLI parameters for default tag injection.
type DefaultTagOptions struct {
	DefaultTags []string             // each entry is like key:"value", may contain placeholders
	InjectType  DefaultTagInjectType // prepend | append
}

// multiStringFlag allows a flag to be specified multiple times on the CLI.
type multiStringFlag []string

func (f *multiStringFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *multiStringFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// resolvePlaceholders replaces placeholder tokens in tagValue with field-derived values.
//
// Supported placeholders:
//
//	{{field_camelCase}}      → strcase.ToLowerCamel(fieldName)
//	{{field_PascalCase}}     → strcase.ToCamel(fieldName)
//	{{field_snake_case}}     → strcase.ToSnake(fieldName)
//	{{field_UPPER_SNAKE_CASE}} → strcase.ToScreamingSnake(fieldName)
//	{{field_kebab_case}}     → strcase.ToKebab(fieldName)
//	{{tag_json_name}}        → json name from existing tag (if missing → skip)
//
// Returns (resolved, true) on success, ("", false) if a required placeholder has no value.
func resolvePlaceholders(tagValue string, fieldName string, jsonTagName string) (string, bool) {
	// If the tag references json name but it's not available, skip this entry
	if strings.Contains(tagValue, "{{tag_json_name}}") {
		if jsonTagName == "" {
			return "", false
		}
		tagValue = strings.ReplaceAll(tagValue, "{{tag_json_name}}", jsonTagName)
	}

	tagValue = strings.ReplaceAll(tagValue, "{{field_camelCase}}", strcase.ToLowerCamel(fieldName))
	tagValue = strings.ReplaceAll(tagValue, "{{field_PascalCase}}", strcase.ToCamel(fieldName))
	tagValue = strings.ReplaceAll(tagValue, "{{field_snake_case}}", strcase.ToSnake(fieldName))
	tagValue = strings.ReplaceAll(tagValue, "{{field_UPPER_SNAKE_CASE}}", strcase.ToScreamingSnake(fieldName))
	tagValue = strings.ReplaceAll(tagValue, "{{field_kebab_case}}", strcase.ToKebab(fieldName))

	return tagValue, true
}

// isProtobufField returns true if the struct tag marks a real protobuf business field.
// Internal fields (state, sizeCache, unknownFields) don't have protobuf tags.
func isProtobufField(tag string) bool {
	return strings.Contains(tag, `protobuf:"`) || strings.Contains(tag, `protobuf_oneof:"`)
}

// extractJSONName extracts the logical name from a json struct tag value.
// e.g. `json:"foo_bar,omitempty"` → "foo_bar"
// Returns "" if there is no json tag.
func extractJSONName(currentTag string) string {
	// Find json:"..."
	const prefix = `json:"`
	idx := strings.Index(currentTag, prefix)
	if idx == -1 {
		return ""
	}
	rest := currentTag[idx+len(prefix):]
	end := strings.Index(rest, `"`)
	if end == -1 {
		return ""
	}
	raw := rest[:end]
	// The json name is the part before the first comma
	if comma := strings.Index(raw, ","); comma != -1 {
		raw = raw[:comma]
	}
	// "-" means the field is intentionally hidden; treat as no json name
	if raw == "-" {
		return ""
	}
	return raw
}

// mergeWithDefault merges defaultItems into existing using prepend/append ordering.
// Priority rule: existing keys always win (default only fills missing keys).
func mergeWithDefault(existing tagItems, defaults tagItems, injectType DefaultTagInjectType) tagItems {
	// Collect keys already present
	existingKeys := map[string]bool{}
	for _, item := range existing {
		existingKeys[item.key] = true
	}

	// Only keep default items whose key is not already present
	var extra tagItems
	for _, d := range defaults {
		if !existingKeys[d.key] {
			extra = append(extra, d)
		}
	}

	if len(extra) == 0 {
		return existing
	}

	if injectType == DefaultTagAppend {
		return append(existing, extra...)
	}
	// prepend (default)
	return append(extra, existing...)
}
