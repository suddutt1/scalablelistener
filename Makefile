.ONESHELL:
.PHONY: all build

MAIN=$(wildcard cmd/*.go)
OUTPUT=build

all: build
qb:  quickBuild

build:
	go mod download
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags "-w -extldflags '-static' " -o $(OUTPUT)/fabric-eventlistener $(MAIN) 

quickBuild:
	go mod download
	go build  -o $(OUTPUT)/fabric-eventlistener $(MAIN) 