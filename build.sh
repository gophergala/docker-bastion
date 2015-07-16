#!/bin/bash
set -ex -o pipefail
go get github.com/mountkin/go-bindata/...
go get github.com/tools/godep
(
  cd manager/views
  go-bindata -nomemcopy -prefix=assets -o assets.go -pkg=views -tags=publish ./assets
)
godep go build -a -tags=publish
