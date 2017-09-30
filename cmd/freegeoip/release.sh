#!/bin/bash

VERSION=$(go run main.go -version | cut -d\  -f2)

function pack() {
	dir=freegeoip-$VERSION-$1
	mkdir $dir
	cp -r ${binary} public $dir
	sync
	gtar --owner=0 --group=0 -czf ${dir}.tar.gz $dir
	rm -rf $dir
}

for OS in linux darwin freebsd windows
do
	binary=freegeoip
	[ $OS = "windows" ] && binary=${binary}.exe
	GOOS=$OS GOARCH=amd64 go build -o ${binary} -ldflags '-w -s'
	sleep 1
	pack $OS-amd64
	rm -f ${binary}
done
