BUILD_CHANNEL?=local
OS=$(shell uname)
GIT_REVISION = $(shell git rev-parse HEAD | tr -d '\n')
TAG_VERSION?=$(shell git tag --points-at | sort -Vr | head -n1)
GO_BUILD_LDFLAGS = -ldflags "-X 'main.Version=${TAG_VERSION}' -X 'main.GitRevision=${GIT_REVISION}'"
GOPATH = $(HOME)/go/bin
export PATH := ${GOPATH}:$(PATH) 
SHELL := /usr/bin/env bash 

default: build

build:
	mkdir -p bin &&  go build $(GO_BUILD_LDFLAGS) -o bin/viam-docker-manager module.go

test: 
	go test -v -coverprofile=coverage.txt -covermode=atomic ./... -race

clean: 
	rm -rf bin

