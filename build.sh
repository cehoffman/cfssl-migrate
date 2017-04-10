#!/bin/bash

set -e

docker build -t migrate-build -f Dockerfile.build .
docker run --rm -v "$PWD":/go/src/github.com/cehoffman/cfssl-migrate migrate-build
upx --brute ./migrate
docker build -t quay.io/cehoffman/cfssl-migrate .
