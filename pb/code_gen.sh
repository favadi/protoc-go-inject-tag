#!/bin/bash

set -eu

protoc --go_out=. test.proto
protoc-go-inject-tag -input=./test.pb.go
