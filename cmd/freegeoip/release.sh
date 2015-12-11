#!/bin/bash

VERSION=$(go run main.go -version | cut -d\  -f2)

function pack() {
	dir=freegeoip-$VERSION-$1
	mkdir $dir
	cp -r freegeoip public $dir
	tar czf ${dir}.tar.gz $dir
	rm -rf $dir
}

for OS in linux darwin freebsd
do
	GOOS=$OS GOARCH=amd64 go build -ldflags '-w'
	pack $OS-amd64
done
