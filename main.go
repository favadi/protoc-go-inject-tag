package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var inputFiles, outputDir, xxxTags string
	var removeTagComment bool
	flag.StringVar(&inputFiles, "input", "", "pattern to match input file(s)")
	flag.StringVar(&outputDir, "outputdir", "", "if set, write tagged copies into this directory instead of overwriting input in place")
	flag.StringVar(&xxxTags, "XXX_skip", "", "tags that should be skipped (applies 'tag:\"-\"') for unknown fields (deprecated since protoc-gen-go v1.4.0)")
	flag.BoolVar(&removeTagComment, "remove_tag_comment", false, "removes tag comments from the generated file(s)")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")

	flag.Parse()

	var xxxSkipSlice []string
	if len(xxxTags) > 0 {
		logf("warn: deprecated flag '-XXX_skip' used")
		xxxSkipSlice = strings.Split(xxxTags, ",")
	}

	if inputFiles == "" {
		log.Fatal("input file is mandatory, see: -help")
	}

	// Note: glob doesn't handle ** (treats as just one *). This will return
	// files and folders, so we'll have to filter them out.
	globResults, err := filepath.Glob(inputFiles)
	if err != nil {
		log.Fatal(err)
	}

	var matched int
	for _, path := range globResults {
		finfo, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		if finfo.IsDir() {
			continue
		}

		// It should end with ".go" at a minimum.
		if !strings.HasSuffix(strings.ToLower(finfo.Name()), ".go") {
			continue
		}

		matched++

		areas, err := parseFile(path, nil, xxxSkipSlice)
		if err != nil {
			log.Fatal(err)
		}
		outputPath := path
		if outputDir != "" {
			outputPath = filepath.Join(outputDir, filepath.Base(path))
		}
		if err = writeFileTo(path, outputPath, areas, removeTagComment); err != nil {
			log.Fatal(err)
		}
	}

	if matched == 0 {
		log.Fatalf("input %q matched no files, see: -help", inputFiles)
	}
}
