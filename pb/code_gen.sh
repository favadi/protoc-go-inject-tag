#!/bin/bash

set -eu

protoc --proto_path=.  --go_opt=default_api_level=API_HYBRID --go_out=paths=source_relative:. test.proto
