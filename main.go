package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	var inputFile string

	flag.StringVar(&inputFile, "input", "", "path to input file")

	flag.Parse()

	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}

	areas, beegoOrmTbls, goIfcMap, err := parseFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	if err = writeFile(inputFile, areas, beegoOrmTbls, goIfcMap); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("fff %#v\n", goIfcMap)
}
