package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	var inputFile string
	var xxxTags string
	flag.StringVar(&inputFile, "input", "", "path to input file")
	flag.StringVar(&xxxTags, "xxx", "", "What tags to inject to xxx fields")

	flag.Parse()

	var xxxSkipSlice []string
	if len(xxxTags) > 0 {
		xxxSkipSlice = strings.Split(xxxTags, ",")
	}

	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}

	areas, err := parseFile(inputFile, xxxSkipSlice)
	if err != nil {
		log.Fatal(err)
	}
	if err = writeFile(inputFile, areas); err != nil {
		log.Fatal(err)
	}
}
