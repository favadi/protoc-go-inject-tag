package main

import (
	"flag"
	"log"
)

func main() {
	var inputFile string

	flag.StringVar(&inputFile, "input", "", "path to input file")

	flag.Parse()

	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}

	areas, beegoOrmTbls, err := parseFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	if err = writeFile(inputFile, areas, beegoOrmTbls); err != nil {
		log.Fatal(err)
	}
}
