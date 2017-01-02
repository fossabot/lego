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
