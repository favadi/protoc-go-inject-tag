# protoc-go-inject-tag

[![Build Status](https://www.travis-ci.com/favadi/protoc-go-inject-tag.svg?branch=master)](https://www.travis-ci.com/favadi/protoc-go-inject-tag)
[![Go Report Card](https://goreportcard.com/badge/github.com/favadi/protoc-go-inject-tag)](https://goreportcard.com/report/github.com/favadi/protoc-go-inject-tag)
[![Coverage Status](https://coveralls.io/repos/github/favadi/protoc-go-inject-tag/badge.svg)](https://coveralls.io/github/favadi/protoc-go-inject-tag)

## Why?

Golang [protobuf](https://github.com/golang/protobuf) doesn't support
[custom tags to generated structs](https://github.com/golang/protobuf/issues/52). This
script injects custom tags to generated protobuf files, useful for
things like validation struct tags.

## Install

* [protobuf version 3](https://github.com/google/protobuf)

  For OS X:
  
  ```
  brew install protobuf
  ```
* go support for protobuf: `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`

*  `go get github.com/favadi/protoc-go-inject-tag` or download the
  binaries from releases page.

## Usage

Add a comment with syntax `// @inject_tag: custom_tag:"custom_value"`
before fields to add custom tag to in .proto files.

Example:

```
// file: test.proto
syntax = "proto3";

package pb;

message IP {
  // @inject_tag: valid:"ip"
  string Address = 1;
}
```

Generate with protoc command as normal.

```
protoc --go_out=. test.proto
```

Run `protoc-go-inject-tag` with generated file `test.pb.go`.

```
protoc-go-inject-tag -input=./test.pb.go
```

The custom tags will be injected to `test.pb.go`.

```
type IP struct {
	// @inject_tag: valid:"ip"
	Address string `protobuf:"bytes,1,opt,name=Address,json=address" json:"Address,omitempty" valid:"ip"`
}
```

To skip the tag for the generated XXX_* fields, use
`-XXX_skip=yaml,xml` flag.

To enable verbose logging, use `-verbose`
