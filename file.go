package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

var (
	rComment       = regexp.MustCompile(`^//\s*@inject_tag:\s*(.*)$`)
	rBeegoOrmTable = regexp.MustCompile(`^//\s*@inject_beego_orm_table:\s*"(.*)".*$`)
	rGoInterface   = regexp.MustCompile(`^//\s*@inject_go_interface:\s*"(.*)".*$`)
	rInject        = regexp.MustCompile("`$")
)

type textArea struct {
	Start int
	End   int
	Tag   string
}

func parseFile(inputPath string) (areas []textArea, beegoOrmTbls [][2]string, goInterfaceMap map[string][]string, err error) {
	log.Printf("parsing file %q for inject tag comments", inputPath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	goInterfaceMap = make(map[string][]string)

	for _, decl := range f.Decls {
		// check if is generic declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		var beegoOrmTblName string
		var goInterfaceName string

		if genDecl != nil && genDecl.Doc != nil {
			for _, spec := range genDecl.Doc.List {
				beegoOrmTblName = beegoOrmTblNameFromComment(spec.Text)
				if beegoOrmTblName != "" {
					break
				}
			}
			for _, spec := range genDecl.Doc.List {
				goInterfaceName = goInterfaceFromComment(spec.Text)
				if goInterfaceName != "" {
					break
				}
			}
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

		if beegoOrmTblName != "" {
			beegoOrmTbls = append(beegoOrmTbls, [2]string{typeSpec.Name.Name, beegoOrmTblName})
		}

		if goInterfaceName != "" {
			goIfcSlice, ok := goInterfaceMap[goInterfaceName]
			if !ok {
				goIfcSlice = make([]string, 0)
				goInterfaceMap[goInterfaceName] = goIfcSlice
			}
			goInterfaceMap[goInterfaceName] = append(goInterfaceMap[goInterfaceName], typeSpec.Name.Name)
		}

		for _, field := range structDecl.Fields.List {
			// skip if field has no doc
			if field.Doc == nil {
				continue
			}
			for _, comment := range field.Doc.List {
				tag := tagFromComment(comment.Text)
				if tag == "" {
					continue
				}
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

func writeFile(inputPath string, areas []textArea, beegoOrmTbls [][2]string, goIfcMap map[string][]string) (err error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return
	}
	defer f.Close()

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

	// append beego orm TableName
	for _, tbl := range beegoOrmTbls {
		txt := fmt.Sprintf("\nfunc (_ *%s) TableName() string {\n    return \"%s\"\n}\n", tbl[0], tbl[1])
		contents = append(contents, []byte(txt)...)
	}

	// append go interface
	for ifcName, sts := range goIfcMap {
		ifcFunc := ifcName + "Func"
		txt := fmt.Sprintf("\ntype %s interface {\n    %s()\n}\n", ifcName, ifcFunc)
		contents = append(contents, []byte(txt)...)

		for _, st := range sts {
			txt := fmt.Sprintf("\nfunc (_ *%s) %s() {}\n", st, ifcFunc)
			contents = append(contents, []byte(txt)...)
		}
	}

	if err = ioutil.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}

	if len(areas) > 0 {
		log.Printf("file %q is injected with custom tags", inputPath)
	}

	return
}
