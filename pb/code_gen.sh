#!/bin/bash

set -eu

protoc --go_out=. test.proto
