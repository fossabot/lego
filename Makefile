install:
		brew install dep && dep ensure

dist: build
		dep ensure

update: build
		dep ensure -update

build:
		go build ./...

test:
		go test -v -race ./...

gen-cert:
		openssl req -x509 -nodes -newkey rsa:2048 -keyout test_key.pem -out test_cert.pem -days 3650

proto-build:
		@find . -iname '*.proto' -not -path "./vendor/*" | xargs -I '{}' protoc --go_out=plugins=grpc:$(shell dirname '{}') '{}'