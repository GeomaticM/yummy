#!/bin/bash

#sudo apt install libdevmapper-dev
#sudo apt install liblvm2-dev
#go get "github.com/nak3/go-lvm"

pushd cmd/agent
CGO_ENABLED=1 GOPATH=/Users/hubaotao/Code/GoGo/ GOOS=linux GOARCH=amd64 go build agent.go
popd

#scp  -P 10022 agent/agent bottle@e.ieevee.com:/tmp


KUBECONFIG=/home/bottle/.kube/config MY_NODE_NAME=ubuntu-2  ./agent

