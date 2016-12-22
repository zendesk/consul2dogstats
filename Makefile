GIT_REV=$(shell git rev-parse --short HEAD)
GIT_DESCRIBE=$(shell git describe --tags --always)
VERSION_PKG=github.com/zendesk/consul2dogstats/version

.PHONY: bin
bin: vendor bin/consul2dogstats

.PHONY: vendor
vendor:
	go get -u github.com/kardianos/govendor
	govendor sync

bin/consul2dogstats: main.go version/version.go
	go build -o bin/consul2dogstats \
		-ldflags "-X $(VERSION_PKG).GitRevision=$(GIT_REV) -X $(VERSION_PKG).GitDescribe=$(GIT_DESCRIBE)" \
		main.go

.PHONY: clean
clean:
	-rm -f bin/consul2dogstats


.PHONY: test
test: vendor
	echo "" > coverage.txt
	for d in $$(go list ./... | grep -v vendor); do \
		go test -v -race -coverprofile=profile.out -covermode=atomic $$d; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi; \
	done
