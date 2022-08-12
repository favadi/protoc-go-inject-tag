package main

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"testing"
)

var (
	testInputFile     = "./pb/test.pb.go"
	testInputFileTemp = "./pb/test.pb.go_tmp"
)

var testsTagFromComment = []struct {
	comment string
	tag     string
}{
	{comment: `//@gotags: valid:"abc"`, tag: `valid:"abc"`},
	{comment: `//   @gotags: valid:"abcd"`, tag: `valid:"abcd"`},
	{comment: `// @gotags:      valid:"xyz"`, tag: `valid:"xyz"`},
	{comment: `// fdsafsa`, tag: ""},
	{comment: `//@gotags:`, tag: ""},
	{comment: `// @gotags: json:"abc" yaml:"abc`, tag: `json:"abc" yaml:"abc`},
	{comment: `// test @gotags: json:"abc" yaml:"abc`, tag: `json:"abc" yaml:"abc`},
	{comment: `// test @inject_tags: json:"abc" yaml:"abc`, tag: `json:"abc" yaml:"abc`},
}

func FuzzTagFromComment(f *testing.F) {
	for _, test := range testsTagFromComment {
		f.Add(test.comment)
	}

	f.Fuzz(func(t *testing.T, orig string) {
		_ = tagFromComment(orig)
	})
}

func TestTagFromComment(t *testing.T) {
	for _, test := range testsTagFromComment {
		if result := tagFromComment(test.comment); result != test.tag {
			t.Errorf("expected tag: %q, got: %q", test.tag, result)
		}
	}
}

func FuzzParseWriteFile(f *testing.F) {
	contents, err := os.ReadFile(testInputFile)
	if err != nil {
		f.Fatal(err)
	}

	f.Add(contents)
	f.Fuzz(func(t *testing.T, orig []byte) {
		areas, err := parseFile("placeholder.pb.go", orig, nil)
		if err == nil {
			for _, area := range areas {
				_ = injectTag(orig, area, false) // Test without annotation removal.
				_ = injectTag(orig, area, true)  // Test with annotation removal.
			}
		}
	})
}

func TestParseWriteFile(t *testing.T) {
	expectedTag := `valid:"ip" yaml:"ip" json:"overrided"`

	areas, err := parseFile(testInputFile, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 9 {
		t.Fatalf("expected 9 areas to replace, got: %d", len(areas))
	}
	area := areas[0]
	if area.InjectTag != expectedTag {
		t.Errorf("expected tag: %q, got: %q", expectedTag, area.InjectTag)
	}

	// make a copy of test file
	contents, err := os.ReadFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(testInputFileTemp, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testInputFileTemp)

	if err = writeFile(testInputFileTemp, areas, false); err != nil {
		t.Fatal(err)
	}

	newAreas, err := parseFile(testInputFileTemp, nil, nil)
	if len(newAreas) != len(areas) {
		t.Errorf("the comment tag has error")
	}
	if err != nil {
		t.Fatal(err)
	}

	// check if file contains custom tag
	contents, err = os.ReadFile(testInputFileTemp)
	if err != nil {
		t.Fatal(err)
	}
	expectedExpr := "Address[ \t]+string[ \t]+`protobuf:\"bytes,1,opt,name=Address,proto3\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`"
	matched, err := regexp.Match(expectedExpr, contents)
	if err != nil || matched != true {
		t.Error("file doesn't contains custom tag after writing")
		t.Log(string(contents))
	}
}

func TestParseWriteFileClearCommon(t *testing.T) {
	expectedTag := `valid:"ip" yaml:"ip" json:"overrided"`

	areas, err := parseFile(testInputFile, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 9 {
		t.Fatalf("expected 9 areas to replace, got: %d", len(areas))
	}
	area := areas[0]
	if area.InjectTag != expectedTag {
		t.Errorf("expected tag: %q, got: %q", expectedTag, area.InjectTag)
	}

	// make a copy of test file
	contents, err := os.ReadFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(testInputFileTemp, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testInputFileTemp)

	if err = writeFile(testInputFileTemp, areas, true); err != nil {
		t.Fatal(err)
	}
	newAreas, err := parseFile(testInputFileTemp, nil, nil)
	if newAreas != nil {
		t.Errorf("not clear tag")
	}
	if err != nil {
		t.Fatal(err)
	}

	// check if file contains custom tag
	contents, err = os.ReadFile(testInputFileTemp)
	if err != nil {
		t.Fatal(err)
	}
	expectedExpr := "Address[ \t]+string[ \t]+`protobuf:\"bytes,1,opt,name=Address,proto3\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`"
	matched, err := regexp.Match(expectedExpr, contents)
	if err != nil || matched != true {
		t.Error("file doesn't contains custom tag after writing")
		t.Log(string(contents))
	}
}

var testsNewTagItems = []struct {
	tag   string
	items tagItems
}{
	{
		tag: `valid:"ip" yaml:"ip, required" json:"overrided"`,
		items: []tagItem{
			{key: "valid", value: `"ip"`},
			{key: "yaml", value: `"ip, required"`},
			{key: "json", value: `"overrided"`},
		},
	},
	{
		tag: `validate:"omitempty,oneof=a b c d"`,
		items: []tagItem{
			{key: "validate", value: `"omitempty,oneof=a b c d"`},
		},
	},
}

func FuzzNewTagItems(f *testing.F) {
	for _, test := range testsNewTagItems {
		f.Add(test.tag)
	}

	f.Fuzz(func(t *testing.T, orig string) {
		_ = newTagItems(orig)
	})
}

func TestNewTagItems(t *testing.T) {
	for _, test := range testsNewTagItems {
		for i, item := range newTagItems(test.tag) {
			if item.key != test.items[i].key || item.value != test.items[i].value {
				t.Errorf("wrong tag item for tag %s, expected %v, got: %v",
					test.tag, test.items[i], item)
			}
		}
	}
}

func TestContinueParsingWhenSkippingFields(t *testing.T) {
	expectedTags := []string{
		`valid:"ip" yaml:"ip" json:"overrided"`,
		`valid:"-"`,
		`valid:"http|https"`,
		`valid:"nonzero"`,
		`validate:"omitempty"`,
		`xml:"-"`,
		`validate:"omitempty"`,
		`tag:"foo_bar"`,
		`tag:"foo"`,
		`tag:"bar"`,
	}

	areas, err := parseFile(testInputFile, nil, []string{"xml"})
	if err != nil {
		t.Fatal(err)
	}

	if len(areas) != len(expectedTags) {
		t.Fatalf("expected %d areas to replace, got: %d", len(expectedTags), len(areas))
	}

	for i, a := range areas {
		if a.InjectTag != expectedTags[i] {
			t.Errorf("expected tag: %q, got: %q", expectedTags[i], a.InjectTag)
		}
	}

	// make a copy of test file
	contents, err := os.ReadFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(testInputFileTemp, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testInputFileTemp)

	if err = writeFile(testInputFileTemp, areas, false); err != nil {
		t.Fatal(err)
	}

	// check if file contains custom tags
	contents, err = os.ReadFile(testInputFileTemp)
	if err != nil {
		t.Fatal(err)
	}

	expectedExprs := []string{
		"Address[ \t]+string[ \t]+`protobuf:\"[^\"]+\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`",
		"Address[ \t]+string[ \t]+`protobuf:\"[^\"]+\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`",
		"Scheme[ \t]+string[ \t]+`protobuf:\"[^\"]+\" json:\"scheme,omitempty\" valid:\"http|https\"`",
		"Port[ \t]+int32[ \t]+`protobuf:\"[^\"]+\" json:\"port,omitempty\" valid:\"nonzero\"`",
		"FooBar[ \t]+isOneOfObject_FooBar[ \t]+`protobuf_oneof:\"[^\"]+\" tag:\"foo_bar\"`",
		"Foo[ \t]+string[ \t]+`protobuf:\"[^\"]+\" tag:\"foo\"`",
		"Bar[ \t]+int64[ \t]+`protobuf:\"[^\"]+\" tag:\"bar\"`",
		"XXX_Deprecated[ \t]+string[ \t]+`protobuf:\"[^\"]+\" json:\"XXX__deprecated,omitempty\" xml:\"-\"`",
	}

	for i, expr := range expectedExprs {
		matched, err := regexp.Match(expr, contents)
		if err != nil || matched != true {
			t.Errorf("file doesn't contains custom tag #%d after writing", i+1)
			t.Log(string(contents))
			break
		}
	}
}

func TestVerbose(t *testing.T) {
	b := new(bytes.Buffer)
	log.SetOutput(b)
	verbose = false
	logf("test")
	if len(b.Bytes()) > 0 {
		t.Errorf("verbose should be off")
	}
	verbose = true
	logf("test")
	if len(b.Bytes()) == 0 {
		t.Errorf("verbose should be on")
	}
}
