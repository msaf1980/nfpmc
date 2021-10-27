NAME:=carbontest

VERSION := $(shell git describe --always --tags)

GO ?= go

all: $(NAME)

FORCE:

$(NAME): FORCE
	CGO_ENABLED=0 $(GO) build -ldflags "-X main.version=${VERSION}" ./cmd/nfpmc

debug: FORCE
	CGO_ENABLED=0 $(GO) build -gcflags=all='-N -l' -ldflags "-X main.version=${VERSION}" ./cmd/nfpmc

test: FORCE
	$(GO) test -coverprofile coverage.txt  ./...

clean:
	@rm -f ./${NAME}

lint:
	golangci-lint run
