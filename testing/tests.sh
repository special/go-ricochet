#!/bin/bash

set -e
pwd
go test -coverprofile=main.cover.out -v .
go test -coverprofile=utils.cover.out -v ./utils
go test -coverprofile=channels.cover.out -v ./channels
go test -coverprofile=connection.cover.out -v ./connection
go test -coverprofile=policies.cover.out -v ./policies
echo "mode: set" > coverage.out && cat *.cover.out | grep -v mode: | sort -r | \
awk '{if($1 != last) {print $0;last=$1}}' >> coverage.out
rm -rf *.cover.out
