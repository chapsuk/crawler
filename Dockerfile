FROM alpine:3.4

ADD . /go/src/github.com/chapsuk/crawler/

RUN apk add --update ca-certificates go \
    && export GOPATH=/go \
    && go build -o /app $GOPATH/src/github.com/chapsuk/crawler/cmd/crawler/main.go \
    && rm -rf /go /usr/local/go

ENTRYPOINT ["/app"]
