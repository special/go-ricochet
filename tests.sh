#!/bin/bash

set -e
pwd
go test -coverprofile=main.cover.out -v .
go test -coverprofile=utils.cover.out -v ./utils
echo "mode: set" > coverage.out && cat *.cover.out | grep -v mode: | sort -r | \
awk '{if($1 != last) {print $0;last=$1}}' >> coverage.out
rm -rf *.cover.out
