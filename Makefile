install:
		go get github.com/tools/godep && \
        ${GOPATH}/bin/godep restore

dist: build
		${GOPATH}/bin/godep save

build:
		go build ./...

test:
		go test ./...