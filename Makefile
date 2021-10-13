all: build

# Go handles build caching, so Go targets should always be marked phony.
.PHONY: all build tags

GIT_DESCRIBE = $(shell git fetch --tags -q ; git describe --dirty)
GIT_COMMIT = $(shell git rev-parse HEAD)
VERSION = github.com/AccumulateNetwork/accumulated.Version=$(GIT_DESCRIBE)
COMMIT = github.com/AccumulateNetwork/accumulated.Commit=$(GIT_COMMIT)

LDFLAGS = '-X "$(VERSION)" -X "$(COMMIT)"'

build: tags
	go build $(BUILDFLAGS) -ldflags $(LDFLAGS) ./cmd/accumulated

install: tags
	go install -ldflags $(LDFLAGS) ./cmd/accumulated