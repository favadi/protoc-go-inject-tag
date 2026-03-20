package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var inputFiles, xxxTags string
	var removeTagComment bool
	var defaultTags multiStringFlag
	var defaultTagInjectType string

	flag.StringVar(&inputFiles, "input", "", "pattern to match input file(s)")
	flag.StringVar(&xxxTags, "XXX_skip", "", "tags that should be skipped (applies 'tag:\"-\"') for unknown fields (deprecated since protoc-gen-go v1.4.0)")
	flag.BoolVar(&removeTagComment, "remove_tag_comment", false, "removes tag comments from the generated file(s)")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.Var(&defaultTags, "default_tag", "default tag to inject into every protobuf field (repeatable); supports placeholders: {{field_snake_case}}, {{field_camelCase}}, {{field_PascalCase}}, {{field_UPPER_SNAKE_CASE}}, {{field_kebab_case}}, {{tag_json_name}}")
	flag.StringVar(&defaultTagInjectType, "default_tag_inject_type", "append", "where default tags are placed relative to existing tags: prepend or append")

	flag.Parse()

	// Validate default_tag_inject_type
	if defaultTagInjectType != string(DefaultTagPrepend) && defaultTagInjectType != string(DefaultTagAppend) {
		log.Fatalf("invalid -default_tag_inject_type %q: must be 'prepend' or 'append'", defaultTagInjectType)
	}

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

		if len(defaultTags) > 0 {
			areas, err = enrichAreasWithDefaultTags(path, nil, areas, DefaultTagOptions{
				DefaultTags: []string(defaultTags),
				InjectType:  DefaultTagInjectType(defaultTagInjectType),
			})
			if err != nil {
				log.Fatal(err)
			}
		}

		if err = writeFile(path, areas, removeTagComment); err != nil {
			log.Fatal(err)
		}
	}

	if matched == 0 {
		log.Fatalf("input %q matched no files, see: -help", inputFiles)
	}
}
