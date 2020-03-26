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
	"strings"
)

var (
	rInjectingTag  = regexp.MustCompile(`^//\s*@inject_tag:\s*(.*)$`)
	rValidationTag = regexp.MustCompile(`^//\s*@validation:\s*(.*)$`)
	rInject        = regexp.MustCompile("`.+`$")
	rTags          = regexp.MustCompile(`[\w_]+:"[^"]+"`)
)

type tagTextArea struct {
	Start      int
	End        int
	CurrentTag string
	InjectTag  string
	StructName string
	FieldName  string
}

func parseFile(inputPath string, xxxSkip []string) (injectingTags []*tagTextArea, validationTags []*tagTextArea, err error) {
	log.Printf("parsing file %q for inject tag comments", inputPath)
	f, err := setupAst(inputPath)
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

		builder := strings.Builder{}
		if len(xxxSkip) > 0 {
			for i, skip := range xxxSkip {
				builder.WriteString(fmt.Sprintf("%s:\"-\"", skip))
				if i > 0 {
					builder.WriteString(",")
				}
			}
		}

		for _, field := range structDecl.Fields.List {
			// skip if field has no doc
			if len(field.Names) == 0 {
				continue
			}

			fieldName := field.Names[0].Name
			if len(xxxSkip) > 0 && strings.HasPrefix(fieldName, "XXX") {
				currentTag := field.Tag.Value
				area := &tagTextArea{
					Start:      int(field.Pos()),
					End:        int(field.End()),
					CurrentTag: currentTag[1 : len(currentTag)-1],
					InjectTag:  builder.String(),
				}

				injectingTags = append(injectingTags, area)
			}

			if field.Doc == nil {
				continue
			}

			for _, comment := range field.Doc.List {
				area := getTagArea(rInjectingTag, field, comment.Text)
				if area != nil {
					area.StructName = typeSpec.Name.Name
					injectingTags = append(injectingTags, area)
					continue
				}

				area = getTagArea(rValidationTag, field, comment.Text)
				if area != nil {
					area.StructName = typeSpec.Name.Name
					area.FieldName = fieldName
					validationTags = append(validationTags, area)
				}
			}
		}
	}

	log.Printf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(injectingTags))
	return
}

func getTagArea(regex *regexp.Regexp, field *ast.Field, text string) (area *tagTextArea) {
	tag := tagFromComment(regex, text)
	if tag == "" {
		return
	}

	currentTag := field.Tag.Value
	area = &tagTextArea{
		Start:      int(field.Pos()),
		End:        int(field.End()),
		CurrentTag: currentTag[1 : len(currentTag)-1],
		InjectTag:  tag,
	}

	return
}

func modifyTags(inputPath string, areas []*tagTextArea) (err error) {
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
		log.Printf("inject custom tag %q to expression %q", area.InjectTag, string(contents[area.Start-1:area.End-1]))
		contents = injectTag(contents, area)
	}
	if err = ioutil.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}

	if len(areas) > 0 {
		log.Printf("file %q is injected with custom tags", inputPath)
	}
	return
}

func addValidateFunctions(inputPath string, tags []*tagTextArea) (err error) {
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

	fileAST, err := setupAst(inputPath)
	if err != nil {
		return
	}

	tagByStructName := map[string][]*tagTextArea{}
	for _, tag := range tags {
		tagByStructName[tag.StructName] = append(tagByStructName[tag.StructName], tag)
	}

	var injected []byte
	for name, tags := range tagByStructName {
		funcs := getFunctions(name, fileAST)
		if len(funcs) < 2 {
			continue
		}

		startPos := funcs[0].Body.Rbrace + 1
		endPos := funcs[1].Type.Func - 1
		validateFunc := setupValidateFunc(name, tags)

		injected = append(injected, contents[:startPos]...)
		injected = append(injected, validateFunc...)
		injected = append(injected, contents[endPos:]...)
		if err = ioutil.WriteFile(inputPath, injected, 0644); err != nil {
			return
		}
	}

	contents = injected
	imports := fileAST.Imports
	if len(imports) == 0 {
		pos := int(fileAST.Name.NamePos) + len(fileAST.Name.Name)
		injected = append([]byte{}, contents[:pos]...)
		injected = append(injected, "\nimport rgx \"regexp\"\n"...)
		injected = append(injected, contents[pos:]...)
		if err = ioutil.WriteFile(inputPath, injected, 0644); err != nil {
			return
		}
		return
	}

	lastImport := imports[len(imports)-1]
	pos := int(lastImport.Path.ValuePos)+len(lastImport.Path.Value)
	injected = append([]byte{}, contents[:pos]...)
	injected = append(injected, "\trgx \"regexp\"\n"...)
	injected = append(injected, contents[pos:]...)
	if err = ioutil.WriteFile(inputPath, injected, 0644); err != nil {
		return
	}
	return
}

func getFunctions(name string, fileAST *ast.File) []*ast.FuncDecl {
	funcs := make([]*ast.FuncDecl, 0)

	for _, decl := range fileAST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Recv == nil {
			continue
		}

		if len(funcDecl.Recv.List) != 1 {
			continue
		}

		funcType, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		token, ok := funcType.X.(*ast.Ident)
		if !ok {
			continue
		}

		if name == token.Name {
			funcs = append(funcs, funcDecl)
		}
	}

	return funcs
}

func setupValidateFunc(name string, tags []*tagTextArea) string {
	bodies := []string{fmt.Sprintf("\nfunc (m *%s) Validate() bool {", name), "\tcompiled := &rgx.Regexp{}\n"}
	for _, t := range tags {
		bodies = append(bodies, fmt.Sprintf("\tcompiled = rgx.MustCompile(%s)", t.InjectTag))
		bodies = append(bodies, fmt.Sprintf("\tif !compiled.MatchString(m.%s) {\n\t return false \n\t}\n", t.FieldName))
	}

	bodies = append(bodies, "\treturn true")
	bodies = append(bodies, "}\n\n")
	return strings.Join(bodies, "\n")
}

func setupAst(inputPath string) (f *ast.File, err error) {
	fset := token.NewFileSet()
	f, err = parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	return
}
