package main

import "fmt"

func tagFromComment(comment string) (tag string) {
	match := rComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

func beegoOrmTblNameFromComment(comment string) (tblName string) {
	match := rBeegoOrmTable.FindStringSubmatch(comment)
	if len(match) == 2 {
		tblName = match[1]
	}
	return
}

func goInterfaceFromComment(comment string) (IfcName string) {
	match := rGoInterface.FindStringSubmatch(comment)
	if len(match) == 2 {
		IfcName = match[1]
	}

	if IfcName == "" {
		return
	}

	bys := []byte(IfcName)
	for i, c := range bys {
		if c < 65 {
			return ""
		} else if c <= 90 {
			bys[i] = c + 32
		} else if c > 90 && c < 97 {
			return ""
		} else if c <= 122 {
			continue
		} else {
			return ""
		}
	}
	bys[0] -= 32

	return string(bys)
}

func injectTag(contents []byte, area textArea) (injected []byte) {
	expr := make([]byte, area.End-area.Start)
	copy(expr, contents[area.Start-1:area.End-1])
	expr = rInject.ReplaceAll(expr, []byte(fmt.Sprintf(" %s`", area.Tag)))
	// contents[area.Start-1 : area.End-1] = expr
	injected = append(injected, contents[:area.Start-1]...)
	injected = append(injected, expr...)
	injected = append(injected, contents[area.End-1:]...)
	return
}
