.DEFAULT: push
.PHONY: all clean image push bin

DOCKER_IMAGE := docker-registry.zende.sk/consul2dogstats
DOCKER_TAG := latest

push: image
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

image: bin
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f Dkrfile .

bin: bin/consul2dogstats

bin/consul2dogstats: main.go
	go build -o bin/consul2dogstats main.go

clean:
	-rm -f bin/*
