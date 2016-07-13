package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

var (
	rComment = regexp.MustCompile(`^//\s*@inject_tag:\s*(?P<tag>.*)$`)
	rInject  = regexp.MustCompile("`$")
)

type textArea struct {
	Start int
	End   int
	Tag   string
}

func parseFile(inputPath string) (areas []textArea, err error) {
	log.Printf("parsing file %q for inject tag comments", inputPath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for _, decl := range f.Decls {
		// check if is generic declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		var typeSpec *ast.TypeSpec
		for _, spec := range genDecl.Specs {
			if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
				typeSpec = ts
				break
			}
		}

		// skip if can't get type spec
		if typeSpec == nil {
			continue
		}

		// not a struct, skip
		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range structDecl.Fields.List {
			// skip if field has no doc
			if field.Doc == nil {
				continue
			}
			for _, comment := range field.Doc.List {
				tag := tagFromComment(comment.Text)
				area := textArea{
					Start: int(field.Pos()),
					End:   int(field.End()),
					Tag:   tag,
				}
				areas = append(areas, area)
			}
		}
	}
	log.Printf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(areas))
	return
}

func writeFile(inputPath string, areas []textArea) (err error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	if err = f.Close(); err != nil {
		return
	}

	// inject custom tags from tail of file first to preserve order
	for i := range areas {
		area := areas[len(areas)-i-1]
		log.Printf("inject custom tag %q to expression %q", area.Tag, string(contents[area.Start-1:area.End-1]))
		contents = injectTag(contents, area)
	}
	if err = ioutil.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}

	log.Printf("file %q is injected with custom tags", inputPath)
	return
}
