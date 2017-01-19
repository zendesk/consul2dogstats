FROM alpine:latest

ARG GIT_REV
ARG GIT_DESCRIBE

ENV GOVERSION 1.7.4
ENV GODISTFILE go${GOVERSION}.linux-amd64.tar.gz
ENV GOPATH /tmp/go
ENV SRCDIR ${GOPATH}/src/github.com/zendesk/consul2dogstats
ENV PATH ${PATH}:/usr/local/go/bin:${GOPATH}/bin
ENV VERSION_PKG github.com/zendesk/consul2dogstats/version

ADD etc/ssl/ca-bundle.pem /etc/ssl/

# See http://stackoverflow.com/questions/34729748/installed-go-binary-not-found-in-path-on-alpine-linux-docker
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
    mkdir -p ${SRCDIR}

ADD . ${SRCDIR}
WORKDIR ${SRCDIR}

ADD https://storage.googleapis.com/golang/${GODISTFILE} /tmp

RUN tar -C /usr/local -xzf /tmp/${GODISTFILE} && \
    apk update && apk add git && \
    go get && \
    go build -o bin/consul2dogstats \
             -ldflags "-X ${VERSION_PKG}.GitRevision=${GIT_REV} -X ${VERSION_PKG}.GitDescribe=${GIT_DESCRIBE}" \
             main.go && \
    mv bin/consul2dogstats /bin/ && \
    rm -rf /usr/local/go /tmp/${GODISTFILE} ${GOPATH} && apk del git

WORKDIR /
USER nobody
ENTRYPOINT ["/bin/consul2dogstats"]
