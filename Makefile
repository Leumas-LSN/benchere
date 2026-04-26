.PHONY: all build web clean test

BINARY  := benchere
OUTDIR  := .
VERSION ?= dev

GO_LDFLAGS := -s -w -X main.Version=$(VERSION)

all: build

web:
	cd web && npm run build

build: web
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -ldflags="$(GO_LDFLAGS)" -o $(OUTDIR)/$(BINARY) ./cmd/benchere

test:
	cd web && npm test
	go test ./...

clean:
	rm -f $(OUTDIR)/$(BINARY)
	find web/dist -mindepth 1 -not -name '.gitkeep' -delete 2>/dev/null; true

.DEFAULT_GOAL := all
