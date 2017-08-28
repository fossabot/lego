install:
		curl https://glide.sh/get | sh && \
        glide install

dist: build
		glide install

update: build
		glide up --all-dependencies

build:
		go build $(glide nv)

test:
		glide nv | xargs go test -v -race

gen-cert:
		openssl req -x509 -nodes -newkey rsa:2048 -keyout test_key.pem -out test_cert.pem -days 3650