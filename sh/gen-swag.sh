#!/bin/bash

export PATH=$(go env GOPATH)/bin:$PATH

swag init -g ./cmd/megachat/main.go -o ./docs