#!/bin/bash

set -xe
pushd cmd/agent
GO111MODULE=off GOPATH=/home/bottle/Code/Go/ GOOS=linux GOARCH=amd64 go build 
popd
