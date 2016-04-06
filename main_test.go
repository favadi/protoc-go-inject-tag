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
	}

	for _, test := range tests {
		result := tagFromComment(test.comment)
		if result != test.tag {
			t.Errorf("expected tag: %q, got: %q", test.tag, result)
		}
	}
}

func TestParseWriteFile(t *testing.T) {
	expectedTag := `valid:"ip"`

	areas, err := parseFile(testInputFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 {
		t.Fatalf("expected 1 area to replace, got: %d", len(areas))
	}
	area := areas[0]
	t.Logf("area: %v", area)
	if area.Tag != expectedTag {
		t.Errorf("expected tag: %q, got: %q", expectedTag, area.Tag)
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
	expectedExpr := "Address string `protobuf:\"bytes,1,opt,name=Address,json=address\" json:\"Address,omitempty\" valid:\"ip\"`"
	if !strings.Contains(string(contents), expectedExpr) {
		t.Error("file doesn't contains custom tag after writing")
		t.Log(string(contents))
	}
}
