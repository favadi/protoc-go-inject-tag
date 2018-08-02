package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	testInputFile     = "./pb/test.pb.go"
	testInputFileTemp = "./pb/test.pb.go_tmp"
)

func TestTagFromComment(t *testing.T) {
	var tests = []struct {
		comment string
		tag     string
	}{
		{comment: `//@inject_tag: valid:"abc"`, tag: `valid:"abc"`},
		{comment: `//   @inject_tag: valid:"abcd"`, tag: `valid:"abcd"`},
		{comment: `// @inject_tag:      valid:"xyz"`, tag: `valid:"xyz"`},
		{comment: `// fdsafsa`, tag: ""},
		{comment: `//@inject_tag:`, tag: ""},
		{comment: `// @inject_tag: json:"abc" yaml:"abc`, tag: `json:"abc" yaml:"abc`},
	}
	for _, test := range tests {
		result := tagFromComment(test.comment)
		if result != test.tag {
			t.Errorf("expected tag: %q, got: %q", test.tag, result)
		}
	}
}

func TestParseWriteFile(t *testing.T) {
	expectedTag := `valid:"ip" yaml:"ip" json:"overrided"`

	areas, err := parseFile(testInputFile, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 3 {
		t.Fatalf("expected 3 area to replace, got: %d", len(areas))
	}
	area := areas[0]
	t.Logf("area: %v", area)
	if area.InjectTag != expectedTag {
		t.Errorf("expected tag: %q, got: %q", expectedTag, area.InjectTag)
	}

	// make a copy of test file
	contents, err := ioutil.ReadFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = ioutil.WriteFile(testInputFileTemp, contents, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testInputFileTemp)

	if err = writeFile(testInputFileTemp, areas); err != nil {
		t.Fatal(err)
	}

	// check if file contains custom tag
	contents, err = ioutil.ReadFile(testInputFileTemp)
	if err != nil {
		t.Fatal(err)
	}
	expectedExpr := "Address string `protobuf:\"bytes,1,opt,name=Address\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`"
	if !strings.Contains(string(contents), expectedExpr) {
		t.Error("file doesn't contains custom tag after writing")
		t.Log(string(contents))
	}
}

func TestNewTagItems(t *testing.T) {
	var tests = []struct {
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

	for _, test := range tests {
		for i, item := range newTagItems(test.tag) {
			if item.key != test.items[i].key || item.value != test.items[i].value {
				t.Errorf("wrong tag item for tag %s, expected %v, got: %v",
					test.tag, test.items[i], item)
			}
		}
	}
}

func TestContinueParsingWhenSkippingFields(t *testing.T) {
	expectedTags := []string{`valid:"ip" yaml:"ip" json:"overrided"`, `valid:"http|https"`, `valid:"nonzero"`}

	areas, err := parseFile(testInputFile, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 3 {
		t.Fatalf("expected 3 areas to replace, got: %d", len(areas))
	}

	for i, a := range areas {
		if a.InjectTag != expectedTags[i] {
			t.Errorf("expected tag: %q, got: %q", expectedTags[i], a.InjectTag)
		}
	}

	// make a copy of test file
	contents, err := ioutil.ReadFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = ioutil.WriteFile(testInputFileTemp, contents, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testInputFileTemp)

	if err = writeFile(testInputFileTemp, areas); err != nil {
		t.Fatal(err)
	}

	// check if file contains 3 three custom tags
	contents, err = ioutil.ReadFile(testInputFileTemp)
	if err != nil {
		t.Fatal(err)
	}

	expectedExprs := []string{
		"Address string `protobuf:\"bytes,1,opt,name=Address\" json:\"overrided\" valid:\"ip\" yaml:\"ip\"`",
		"Scheme string `protobuf:\"bytes,1,opt,name=scheme\" json:\"scheme,omitempty\" valid:\"http|https\"`",
		"Port int32 `protobuf:\"varint,3,opt,name=port\" json:\"port,omitempty\" valid:\"nonzero\"`",
	}

	for i, expr := range expectedExprs {
		if !strings.Contains(string(contents), expr) {
			t.Errorf("file doesn't contains custom tag #%d after writing", i+1)
			t.Log(string(contents))
			break
		}
	}
}
