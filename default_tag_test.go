package main

import (
	"strings"
	"testing"
)

// ── helper ────────────────────────────────────────────────────────────────────

// parseInline parses a Go source string as a virtual file named "test.pb.go".
func parseInline(src string) ([]textArea, error) {
	return parseFile("test.pb.go", []byte(src), nil)
}

// enrichInline runs enrichAreasWithDefaultTags on an inline source string.
func enrichInline(src string, opts DefaultTagOptions) ([]textArea, error) {
	areas, err := parseFile("test.pb.go", []byte(src), nil)
	if err != nil {
		return nil, err
	}
	return enrichAreasWithDefaultTags("test.pb.go", []byte(src), areas, opts)
}

// areaInjectTag returns the InjectTag for the first area whose CurrentTag
// contains the given key (for convenient assertions).
func findAreaByField(areas []textArea, currentTagSubstr string) *textArea {
	for i := range areas {
		if strings.Contains(areas[i].CurrentTag, currentTagSubstr) {
			return &areas[i]
		}
	}
	return nil
}

// ── minimal protobuf struct source templates ──────────────────────────────────

const protoImport = `
package main
import protoimpl "google.golang.org/protobuf/runtime/protoimpl"
`

// buildSrc wraps fields into a proto-style struct.
func buildSrc(fields string) string {
	return protoImport + `
type MyMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
` + fields + `
}
`
}

// ── Test 1: multiple -default_tag entries are all applied ─────────────────────

func TestDefaultTag_MultipleEntries(t *testing.T) {
	src := buildSrc(`
	UserName string ` + "`" + `protobuf:"bytes,1,opt,name=user_name,proto3" json:"user_name,omitempty"` + "`" + `
`)
	areas, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{`validate:"required"`, `yaml:"{{field_snake_case}}"`},
		InjectType:  DefaultTagPrepend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("expected 1 area, got %d", len(areas))
	}
	inj := areas[0].InjectTag
	// Both default tags must appear
	if !strings.Contains(inj, `validate:`) {
		t.Errorf("expected validate tag in InjectTag, got: %q", inj)
	}
	if !strings.Contains(inj, `yaml:`) {
		t.Errorf("expected yaml tag in InjectTag, got: %q", inj)
	}
}

// ── Test 2: invalid inject_type is caught at validation level ─────────────────

func TestDefaultTag_InvalidInjectType(t *testing.T) {
	// The CLI validation is in main(); here we test the enum values exist.
	validTypes := []DefaultTagInjectType{DefaultTagPrepend, DefaultTagAppend}
	for _, v := range validTypes {
		if v != "prepend" && v != "append" {
			t.Errorf("unexpected enum value: %q", v)
		}
	}
	// Anything else should be invalid
	invalid := DefaultTagInjectType("middle")
	if invalid == DefaultTagPrepend || invalid == DefaultTagAppend {
		t.Error("'middle' should not be a valid inject type")
	}
}

// ── Test 3: all field_* placeholders are resolved correctly ───────────────────

func TestDefaultTag_FieldPlaceholders(t *testing.T) {
	cases := []struct {
		placeholder string
		fieldName   string
		want        string
	}{
		{`yaml:"{{field_camelCase}}"`, "UserName", `yaml:"userName"`},
		{`yaml:"{{field_PascalCase}}"`, "user_name", `yaml:"UserName"`},
		{`yaml:"{{field_snake_case}}"`, "UserName", `yaml:"user_name"`},
		{`yaml:"{{field_UPPER_SNAKE_CASE}}"`, "UserName", `yaml:"USER_NAME"`},
		{`yaml:"{{field_kebab_case}}"`, "UserName", `yaml:"user-name"`},
	}

	for _, tc := range cases {
		got, ok := resolvePlaceholders(tc.placeholder, tc.fieldName, "")
		if !ok {
			t.Errorf("resolvePlaceholders(%q, %q) returned ok=false", tc.placeholder, tc.fieldName)
			continue
		}
		if got != tc.want {
			t.Errorf("placeholder %q, field %q: want %q, got %q", tc.placeholder, tc.fieldName, tc.want, got)
		}
	}
}

// ── Test 4: {{tag_json_name}} picks up the json name from existing tag ─────────

func TestDefaultTag_TagJsonName(t *testing.T) {
	got, ok := resolvePlaceholders(`yaml:"{{tag_json_name}}"`, "UserName", "user_name")
	if !ok {
		t.Fatal("expected ok=true when json name is available")
	}
	if got != `yaml:"user_name"` {
		t.Errorf("want yaml:\"user_name\", got %q", got)
	}
}

// ── Test 5: {{tag_json_name}} skipped when field has no json tag ──────────────

func TestDefaultTag_TagJsonName_Missing(t *testing.T) {
	// {{tag_json_name}} with empty jsonName → skip (ok=false)
	_, ok := resolvePlaceholders(`yaml:"{{tag_json_name}}"`, "NoJsonTag", "")
	if ok {
		t.Error("expected ok=false when json name is empty")
	}

	// Other placeholder entries should still be applied
	src := buildSrc(`
	NoJsonTag string ` + "`" + `protobuf:"bytes,1,opt,name=no_json_tag,proto3"` + "`" + `
	UserName  string ` + "`" + `protobuf:"bytes,2,opt,name=user_name,proto3" json:"user_name,omitempty"` + "`" + `
`)
	areas, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{
			`yaml:"{{tag_json_name}}"`, // skipped for NoJsonTag, applied to UserName
			`validate:"required"`,      // applied to both
		},
		InjectType: DefaultTagPrepend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 {
		t.Fatalf("expected 2 areas, got %d", len(areas))
	}

	// Find the area for NoJsonTag: it has no json tag, so its InjectTag contains
	// only validate (not yaml). The area for UserName has both validate and yaml.
	var noJsonArea, userNameArea *textArea
	for i := range areas {
		if strings.Contains(areas[i].InjectTag, "yaml:") {
			userNameArea = &areas[i]
		} else {
			noJsonArea = &areas[i]
		}
	}
	if noJsonArea == nil {
		t.Fatal("could not find area for NoJsonTag (expected area without yaml:)")
	}
	if userNameArea == nil {
		t.Fatal("could not find area for UserName (expected area with yaml:)")
	}
	// yaml must NOT appear for NoJsonTag (no json tag → {{tag_json_name}} skipped)
	if strings.Contains(noJsonArea.InjectTag, "yaml:") {
		t.Errorf("yaml tag should be skipped for NoJsonTag, got InjectTag: %q", noJsonArea.InjectTag)
	}
	// validate must appear for NoJsonTag
	if !strings.Contains(noJsonArea.InjectTag, "validate:") {
		t.Errorf("validate tag should be applied to NoJsonTag, got InjectTag: %q", noJsonArea.InjectTag)
	}
}

// ── Test 6: priority @gotags > existing tag > default_tag ────────────────────

func TestDefaultTag_Priority(t *testing.T) {
	// Field has existing json tag AND @gotags overrides yaml.
	// default_tag tries to set both json and yaml → both should be blocked.
	src := buildSrc(`
	UserName string ` + "`" + `protobuf:"bytes,1,opt,name=user_name,proto3" json:"existing_json" yaml:"existing_yaml"` + "`" + ` // @gotags: yaml:"gotags_yaml"
`)
	areas, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{`json:"default_json"`, `yaml:"default_yaml"`},
		InjectType:  DefaultTagPrepend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("expected 1 area, got %d", len(areas))
	}
	area := areas[0]

	// Simulate injectTag to get final output
	final := newTagItems(area.CurrentTag).override(newTagItems(area.InjectTag)).format()

	// @gotags yaml wins over existing yaml AND default yaml
	if !strings.Contains(final, `yaml:"gotags_yaml"`) {
		t.Errorf("@gotags yaml should win, got final: %q", final)
	}
	// existing json wins over default json
	if !strings.Contains(final, `json:"existing_json"`) {
		t.Errorf("existing json should be preserved, got final: %q", final)
	}
	// default_json must NOT appear
	if strings.Contains(final, `json:"default_json"`) {
		t.Errorf("default json should be blocked by existing json, got final: %q", final)
	}
}

// ── Test 7: prepend vs append only affects ordering, not priority ─────────────

func TestDefaultTag_PrependAppendOrder(t *testing.T) {
	src := buildSrc(`
	UserName string ` + "`" + `protobuf:"bytes,1,opt,name=user_name,proto3" json:"user_name,omitempty"` + "`" + `
`)

	areasP, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{`validate:"required"`},
		InjectType:  DefaultTagPrepend,
	})
	if err != nil {
		t.Fatal(err)
	}

	areasA, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{`validate:"required"`},
		InjectType:  DefaultTagAppend,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(areasP) != 1 || len(areasA) != 1 {
		t.Fatalf("expected 1 area each, got prepend=%d append=%d", len(areasP), len(areasA))
	}

	finalP := newTagItems(areasP[0].CurrentTag).override(newTagItems(areasP[0].InjectTag)).format()
	finalA := newTagItems(areasA[0].CurrentTag).override(newTagItems(areasA[0].InjectTag)).format()

	// Both must contain validate
	if !strings.Contains(finalP, "validate:") {
		t.Errorf("prepend: missing validate tag in %q", finalP)
	}
	if !strings.Contains(finalA, "validate:") {
		t.Errorf("append: missing validate tag in %q", finalA)
	}

	// prepend: validate appears before json
	if strings.Index(finalP, "validate:") > strings.Index(finalP, "json:") {
		t.Errorf("prepend: expected validate before json, got %q", finalP)
	}

	// append: validate appears after json
	if strings.Index(finalA, "validate:") < strings.Index(finalA, "json:") {
		t.Errorf("append: expected validate after json, got %q", finalA)
	}
}

// ── Test 8: internal protobuf fields (state/sizeCache/unknownFields) skipped ──

func TestDefaultTag_InternalFieldsSkipped(t *testing.T) {
	// The internal fields have no protobuf:" tag, so isProtobufField returns false.
	if isProtobufField(``) {
		t.Error("empty tag should not be a protobuf field")
	}
	if isProtobufField(`json:"foo"`) {
		t.Error("json-only tag should not be a protobuf field")
	}
	if !isProtobufField(`protobuf:"bytes,1,opt" json:"foo"`) {
		t.Error("tag with protobuf: should be a protobuf field")
	}
	if !isProtobufField(`protobuf_oneof:"foo"`) {
		t.Error("tag with protobuf_oneof: should be a protobuf field")
	}

	// End-to-end: internal fields should produce 0 areas
	src := buildSrc(``) // only state/sizeCache/unknownFields which have no protobuf tag
	areas, err := enrichInline(src, DefaultTagOptions{
		DefaultTags: []string{`validate:"required"`},
		InjectType:  DefaultTagPrepend,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 0 {
		t.Errorf("expected 0 areas for internal-only struct, got %d", len(areas))
	}
}

// ── Test 9: idempotent — running twice produces the same result ───────────────

func TestDefaultTag_Idempotent(t *testing.T) {
	src := buildSrc(`
	UserName string ` + "`" + `protobuf:"bytes,1,opt,name=user_name,proto3" json:"user_name,omitempty"` + "`" + `
`)

	opts := DefaultTagOptions{
		DefaultTags: []string{`validate:"required"`, `yaml:"{{field_snake_case}}"`},
		InjectType:  DefaultTagPrepend,
	}

	// First pass
	areas1, err := enrichInline(src, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas1) == 0 {
		t.Fatal("expected areas after first pass")
	}

	// Simulate what the file would look like after writeFile (apply inject)
	// by computing the effective final tag and building new source.
	area1 := areas1[0]
	finalTag := newTagItems(area1.CurrentTag).override(newTagItems(area1.InjectTag)).format()

	// Build "already-injected" source
	src2 := buildSrc(`
	UserName string ` + "`" + finalTag + "`" + `
`)

	// Second pass: should produce no new areas (or areas with identical InjectTag)
	areas2, err := enrichInline(src2, opts)
	if err != nil {
		t.Fatal(err)
	}

	// After injection the current tag already contains all default keys,
	// so enrichAreasWithDefaultTags should produce no new areas (idempotent).
	if len(areas2) != 0 {
		for _, a := range areas2 {
			// Any remaining area should have an empty or unchanged InjectTag
			finalTag2 := newTagItems(a.CurrentTag).override(newTagItems(a.InjectTag)).format()
			if finalTag2 != a.CurrentTag {
				t.Errorf("second pass changed tag: before=%q after=%q", a.CurrentTag, finalTag2)
			}
		}
	}
}

// ── Test helpers unit tests ───────────────────────────────────────────────────

func TestExtractJSONName(t *testing.T) {
	cases := []struct {
		tag  string
		want string
	}{
		{`protobuf:"bytes,1" json:"user_name,omitempty"`, "user_name"},
		{`protobuf:"bytes,1" json:"foo"`, "foo"},
		{`protobuf:"bytes,1"`, ""},
		{`protobuf:"bytes,1" json:"-"`, ""},
		{`protobuf:"bytes,1" json:",omitempty"`, ""},
	}
	for _, tc := range cases {
		got := extractJSONName(tc.tag)
		if got != tc.want {
			t.Errorf("extractJSONName(%q): want %q got %q", tc.tag, tc.want, got)
		}
	}
}

func TestMergeWithDefault(t *testing.T) {
	existing := newTagItems(`json:"foo" protobuf:"bytes,1"`)
	defaults := newTagItems(`validate:"required" json:"should_be_blocked"`)

	// prepend: validate before json
	merged := mergeWithDefault(existing, defaults, DefaultTagPrepend)
	f := merged.format()
	if !strings.Contains(f, `validate:"required"`) {
		t.Errorf("missing validate in %q", f)
	}
	if strings.Contains(f, `json:"should_be_blocked"`) {
		t.Errorf("blocked json appeared in %q", f)
	}
	if strings.Index(f, "validate:") > strings.Index(f, "json:") {
		t.Errorf("prepend: validate should come before json in %q", f)
	}

	// append: validate after json
	mergedA := mergeWithDefault(existing, defaults, DefaultTagAppend)
	fA := mergedA.format()
	if strings.Index(fA, "validate:") < strings.Index(fA, "json:") {
		t.Errorf("append: validate should come after json in %q", fA)
	}
}
