package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// enrichAreasWithDefaultTags applies default tag injection to all protobuf business fields.
//
// For each field in the file:
//  1. Resolve placeholders in each default_tag entry using field name & json tag name.
//  2. Merge resolved defaults into the field's effective inject tag (lowest priority).
//  3. If a field already has a @gotags area, update its InjectTag in-place.
//  4. If a field has no @gotags area but defaults produce a non-empty injection, create a new area.
//  5. Skip if the computed injection would produce no change (idempotent).
func enrichAreasWithDefaultTags(
	inputPath string,
	src interface{},
	areas []textArea,
	opts DefaultTagOptions,
) ([]textArea, error) {
	if len(opts.DefaultTags) == 0 {
		return areas, nil
	}

	// Build a lookup: field byte-start → index in areas slice
	areaByStart := map[int]int{}
	for i, a := range areas {
		areaByStart[a.Start] = i
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var extraAreas []textArea

	for _, decl := range f.Decls {
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
		if typeSpec == nil {
			continue
		}

		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range structDecl.Fields.List {
			// Must have a tag
			if field.Tag == nil {
				continue
			}

			rawTag := field.Tag.Value
			currentTag := rawTag[1 : len(rawTag)-1] // strip backticks

			// Only process real protobuf business fields
			if !isProtobufField(currentTag) {
				continue
			}

			// Must have at least one name (anonymous embed fields have none)
			if len(field.Names) == 0 {
				continue
			}
			fieldName := field.Names[0].Name

			// Determine effective json name:
			// If this field has a @gotags area that overrides the json key, use that;
			// otherwise use the json name from currentTag.
			jsonName := extractJSONName(currentTag)
			fieldStart := int(field.Pos())
			if aIdx, hasArea := areaByStart[fieldStart]; hasArea {
				overrideJSON := extractJSONName(areas[aIdx].InjectTag)
				if overrideJSON != "" {
					jsonName = overrideJSON
				}
			}

			// Resolve each default_tag entry for this field
			var resolvedDefaults tagItems
			for _, defaultTagStr := range opts.DefaultTags {
				resolved, ok := resolvePlaceholders(defaultTagStr, fieldName, jsonName)
				if !ok {
					// e.g. {{tag_json_name}} but no json tag → skip this entry
					continue
				}
				items := newTagItems(resolved)
				resolvedDefaults = append(resolvedDefaults, items...)
			}

			if len(resolvedDefaults) == 0 {
				continue
			}

			// Build "existing tag items" = currentTag (what's already in the .go file)
			existingItems := newTagItems(currentTag)

			if aIdx, hasArea := areaByStart[fieldStart]; hasArea {
				// Field has @gotags: merge default into InjectTag (lowest priority)
				// The default must not override keys already in currentTag
				currentItems := newTagItems(areas[aIdx].CurrentTag)
				injectItems := newTagItems(areas[aIdx].InjectTag)

				// Filter out default keys that are present in currentTag
				currentKeys := map[string]bool{}
				for _, ci := range currentItems {
					currentKeys[ci.key] = true
				}
				var filteredDefaults tagItems
				for _, d := range resolvedDefaults {
					if !currentKeys[d.key] {
						filteredDefaults = append(filteredDefaults, d)
					}
				}
				if len(filteredDefaults) == 0 {
					continue
				}

				// Merge filtered defaults into injectItems
				newInjectItems := mergeWithDefault(injectItems, filteredDefaults, opts.InjectType)
				newInjectTag := newInjectItems.format()

				if newInjectTag == areas[aIdx].InjectTag {
					continue // no change
				}
				areas[aIdx].InjectTag = newInjectTag
			} else {
				// No @gotags area: compute the full desired final tag and inject it.
				// Strategy: set CurrentTag="" and InjectTag=full-desired-tag so that
				// cti.override(iti) = iti (empty cti), giving us full control over order.
				finalItems := mergeWithDefault(existingItems, resolvedDefaults, opts.InjectType)
				finalTag := finalItems.format()
				if finalTag == currentTag {
					continue // already has all default tags → idempotent
				}

				newArea := textArea{
					Start: fieldStart,
					End:   int(field.End()),
					// Empty CurrentTag so override() returns InjectTag as-is,
					// preserving our desired ordering (prepend or append).
					CurrentTag: "",
					InjectTag:  finalTag,
				}
				extraAreas = append(extraAreas, newArea)
			}
		}
	}

	return append(areas, extraAreas...), nil
}

// filterDefaultTagKeys returns the subset of items whose key is NOT in excludeKeys.
func filterDefaultTagKeys(items tagItems, excludeKeys map[string]bool) tagItems {
	var out tagItems
	for _, item := range items {
		if !excludeKeys[item.key] {
			out = append(out, item)
		}
	}
	return out
}

// effectiveTagString simulates what injectTag() would produce for a textArea
// (used to detect idempotency for areas without @gotags).
func effectiveTagString(currentTag, injectTag string) string {
	cti := newTagItems(currentTag)
	iti := newTagItems(injectTag)
	return cti.override(iti).format()
}

// resolveDefaultTagsForField resolves all default_tag entries for a single field,
// returning the merged tagItems ready to inject.
func resolveDefaultTagsForField(
	opts DefaultTagOptions,
	fieldName string,
	jsonName string,
) tagItems {
	var resolved tagItems
	for _, raw := range opts.DefaultTags {
		r, ok := resolvePlaceholders(raw, fieldName, jsonName)
		if !ok {
			continue
		}
		resolved = append(resolved, newTagItems(r)...)
	}
	return resolved
}

// uniqueTagItems deduplicates tagItems keeping the first occurrence of each key.
func uniqueTagItems(items tagItems) tagItems {
	seen := map[string]bool{}
	var out tagItems
	for _, item := range items {
		if !seen[item.key] {
			seen[item.key] = true
			out = append(out, item)
		}
	}
	return out
}

// splitTagString parses a raw CLI default_tag value like `json:"{{field_snake_case}}"`.
// It handles both `key:"value"` format (standard) and any placeholder-containing value.
func splitTagString(s string) (key string, value string, ok bool) {
	idx := strings.Index(s, ":")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}
