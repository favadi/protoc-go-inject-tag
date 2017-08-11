# protoc-go-inject-tag

[![Build Status](https://travis-ci.org/favadi/protoc-go-inject-tag.svg?branch=master)](https://travis-ci.org/favadi/protoc-go-inject-tag)
[![Go Report Card](https://goreportcard.com/badge/github.com/favadi/protoc-go-inject-tag)](https://goreportcard.com/report/github.com/favadi/protoc-go-inject-tag)

## Why?

Golang [protobuf](https://github.com/golang/protobuf) doesn't support
[custom tags to generated structs](https://github.com/golang/protobuf/issues/52). This
script injects custom tags to generated protobuf files, useful for
things like validation struct tags.

This repo is based on favadi/protoc-go-inject-tag, but add @inject_beego_orm_table instruction for beego/orm users.

## Install

* [protobuf version 3](https://github.com/google/protobuf)

  For OS X:

  ```
  brew install --devel protobuf
  ```
* go support for protobuf: `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`

*  `go get github.com/stormgbs/protoc-go-inject-tag` or download the
  binaries from releases page.

## Usage

Add a comment with syntax `// @inject_tag: custom_tag:"custom_value"`before fields to add custom tag,
or a comment with syntax `// @inject_beego_orm_table: "custom_table_name"` before message in .proto files

Example:

```
// file: test.proto
syntax = "proto3";

package pb;

// @inject_beego_orm_table: "my_ip_table"
message IP {
  // @inject_tag: orm:"column(address)"
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
// @inject_go_interface: "eventer" 1
// @inject_beego_orm_table: "event_foo"
message EventFoo {
  // @inject_tag: orm:"pk"
  string id = 1;
}

// @inject_go_interface: "eventer" 2
// @inject_beego_orm_table: "event_bar"
message EventBar {
  // @inject_tag: orm:"pk"
  string id = 1;
  // @inject_tag: orm:"column(bb)"
  string bb = 2;
  // @inject_tag: orm:"column(bc)"
  int32 bc = 3;
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
// @inject_go_interface: "eventer" 1
// @inject_beego_orm_table: "event_foo"
type EventFoo struct {
	// @inject_tag: orm:"pk"
	Id string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty" orm:"pk"`
}

// @inject_go_interface: "eventer" 2
// @inject_beego_orm_table: "event_bar"
type EventBar struct {
	// @inject_tag: orm:"pk"
	Id string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty" orm:"pk"`
	// @inject_tag: orm:"column(bb)"
	Bb string `protobuf:"bytes,2,opt,name=bb" json:"bb,omitempty" orm:"column(bb)"`
	// @inject_tag: orm:"column(bc)"
	Bc int32 `protobuf:"varint,3,opt,name=bc" json:"bc,omitempty" orm:"column(bc)"`
}

... ...

func (_ *EventFoo) TableName() string {
	return "event_foo"
}

func (_ *EventBar) TableName() string {
	return "event_bar"
}

type Eventer interface {
	EventerFunc()
}

func (_ *EventFoo) EventerFunc() {}

func (_ *EventBar) EventerFunc() {}

```
