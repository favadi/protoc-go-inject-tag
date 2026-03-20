# protoc-go-inject-tag

[![Build Status](https://www.travis-ci.com/favadi/protoc-go-inject-tag.svg?branch=master)](https://www.travis-ci.com/favadi/protoc-go-inject-tag)
[![Go Report Card](https://goreportcard.com/badge/github.com/favadi/protoc-go-inject-tag)](https://goreportcard.com/report/github.com/favadi/protoc-go-inject-tag)
[![Coverage Status](https://coveralls.io/repos/github/favadi/protoc-go-inject-tag/badge.svg)](https://coveralls.io/github/favadi/protoc-go-inject-tag)

## Why?

Golang [protobuf](https://github.com/golang/protobuf) doesn't support
[custom tags to generated structs](https://github.com/golang/protobuf/issues/52).
This tool injects custom tags to generated protobuf files, which is commonly
used for validating fields, omitting fields from JSON data, etc.

## Install

- [protobuf version 3](https://github.com/google/protobuf)

  For OS X:

  ```console
  brew install protobuf
  ```

- go support for protobuf: `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`

- `go install github.com/favadi/protoc-go-inject-tag@latest` or download the
  binaries from the releases page.

## Usage

```console
$ protoc-go-inject-tag -h
Usage of protoc-go-inject-tag:
  -XXX_skip string
        tags that should be skipped (applies 'tag:"-"') for unknown fields (deprecated since protoc-gen-go v1.4.0)
  -default_tag value
        default tag to inject into every protobuf field (repeatable); supports placeholders: {{field_snake_case}}, {{field_camelCase}}, {{field_PascalCase}}, {{field_UPPER_SNAKE_CASE}}, {{field_kebab_case}}, {{tag_json_name}}
  -default_tag_inject_type string
        where default tags are placed relative to existing tags: prepend or append (default "append")
  -input string
        pattern to match input file(s)
  -verbose
        verbose logging
  -remove_tag_comment
        removes tag comments from the generated file(s)
```

Add a comment with the following syntax before fields, and these will be
injected into the resulting `.pb.go` file. This can be specified above the
field, or trailing the field.

```proto
// @gotags: custom_tag:"custom_value"
```

## Example

```proto
// file: test.proto
syntax = "proto3";

package pb;
option go_package = "/pb";

message IP {
  // @gotags: valid:"ip"
  string Address = 1;

  // Or:
  string MAC = 2; // @gotags: validate:"omitempty"
}
```

Generate your `.pb.go` files with the protoc command as normal:

```console
protoc --proto_path=. --go_out=paths=source_relative:. test.proto
```

Then run `protoc-go-inject-tag` against the generated files (e.g `test.pb.go`):

```console
$ protoc-go-inject-tag -input=./test.pb.go
# or
$ protoc-go-inject-tag -input="*.pb.go"
```

The custom tags will be injected to `test.pb.go`:

```go
type IP struct {
 // @gotags: valid:"ip"
 Address string `protobuf:"bytes,1,opt,name=Address,json=address" json:"Address,omitempty" valid:"ip"`
}
```

## Remove gotag comments from generated output

Utilizing the `-remove_tag_comment` flag, you can remove the gotag comment that
is normally annotated to the generated code. This allows more seamless support with
libraries like swag/openapi generators that use code comments to generate openapi
files.

## Default tags

Use `-default_tag` to inject a tag into **every** protobuf business field globally,
without adding per-field `@gotags` comments. The flag is repeatable.

```console
$ protoc-go-inject-tag -input="./pb/*.pb.go" -default_tag='validate:"required"' -default_tag='yaml:"{{field_snake_case}}"'
```

### Placeholders

The tag value supports placeholders derived from the field name and its existing `json` tag:

| Placeholder | Example (field: `UserName`) |
|---|---|
| `{{field_snake_case}}` | `user_name` |
| `{{field_camelCase}}` | `userName` |
| `{{field_PascalCase}}` | `UserName` |
| `{{field_UPPER_SNAKE_CASE}}` | `USER_NAME` |
| `{{field_kebab_case}}` | `user-name` |
| `{{tag_json_name}}` | value from existing `json:"..."` tag |

If `{{tag_json_name}}` is used on a field that has no `json` tag, the entry is silently skipped for that field.

### Inject position

Use `-default_tag_inject_type` to control where default tags are placed relative to existing tags:

```console
# Place default tags before existing tags
$ protoc-go-inject-tag -input="./pb/*.pb.go" -default_tag='validate:"required"' -default_tag_inject_type=prepend
```

Default is `append`.

### Priority

Default tags are **lowest priority**: if a field already has the same tag key (via its struct tag or a `@gotags` comment), the existing value is preserved and the default is not applied.

## Deprecated functionality

### Skip `XXX_*` fields

To skip the tag for the generated `XXX_*` fields (unknown fields), use the
`-XXX_skip=yaml,xml` flag. This is deprecated, as this functionality hasn't
existed in `protoc-gen-go` since v1.4.x.

#### `inject_tag` keyword

Since **v1.3.0**, we recommend using `@gotags:` rather than `@inject_tags:`,
as `@gotags` is more indicative of the language the comment is for. We don't
plan on removing `@inject_tags:` support anytime soon, however we strongly
recommend switching to `@gotags`.
