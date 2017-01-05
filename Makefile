.DEFAULT: push
.PHONY: all clean image push bin

DOCKER_IMAGE := docker-registry.zende.sk/consul2dogstats
DOCKER_TAG := latest
GIT_REV=$(shell git rev-parse --short HEAD)
GIT_DESCRIBE=$(shell git describe --tags --always)
VERSION_PKG=github.com/zendesk/consul2dogstats/version

push: image
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

image: bin
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):git-$(GIT_REV) -f Dkrfile .

bin: bin/consul2dogstats

bin/consul2dogstats: main.go version/version.go
	go build -o bin/consul2dogstats \
		-ldflags "-X $(VERSION_PKG).GitRevision=$(GIT_REV) -X $(VERSION_PKG).GitDescribe=$(GIT_DESCRIBE)" \
		main.go

clean:
	-rm -f bin/*
