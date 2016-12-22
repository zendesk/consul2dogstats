.DEFAULT: all
.PHONY: all clean

all: consul2dogstats

consul2dogstats: main.go
	go build -o bin/consul2dogstats main.go

clean:
	-rm -f bin/*
