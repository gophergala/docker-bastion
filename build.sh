#!/bin/bash
set -ex -o pipefail
go get -u github.com/mountkin/go-bindata/...
(
  cd manager/views
  go-bindata -nomemcopy -prefix=assets -o assets.go -pkg=views -tags=publish ./assets
)
go build -tags=publish
