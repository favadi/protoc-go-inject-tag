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
	flag.StringVar(&xxxTags, "XXX_skip", "", "skip tags to inject on XXX fields")

	flag.Parse()

	var xxxSkipSlice []string
	if len(xxxTags) > 0 {
		xxxSkipSlice = strings.Split(xxxTags, ",")
	}

	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}

	injectingTags, validationTags, err := parseFile(inputFile, xxxSkipSlice)
	if err != nil {
		log.Fatal(err)
	}

	if err = modifyTags(inputFile, injectingTags); err != nil {
		log.Fatal(err)
	}

	if err = addValidateFunctions(inputFile, validationTags); err != nil {
		log.Fatal(err)
	}
}
