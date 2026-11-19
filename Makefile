.PHONY: build test lint clean help release release-artifacts

MODULE  := github.com/openshift-online/gcp-hcp-ctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X $(MODULE)/pkg/cli.version=$(VERSION) \
	-X $(MODULE)/pkg/cli.commit=$(COMMIT) \
	-X $(MODULE)/pkg/cli.date=$(DATE)

DIST      := dist
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
# macOS does not ship sha256sum; fall back to shasum there.
SHA256SUM := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo shasum -a 256)
# Deterministic archive metadata when the local tar supports it (GNU tar, used
# by the release workflow). macOS bsdtar does not support these flags, so the
# probe yields empty and plain tar runs there.
TAR_FLAGS := $(shell tar --mtime=@0 --owner=0 --group=0 --numeric-owner -cf /dev/null Makefile >/dev/null 2>&1 && echo '--mtime=@0 --owner=0 --group=0 --numeric-owner')

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Build the gcphcpctl binary
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/gcphcpctl ./cmd/gcphcpctl
	@echo "Built bin/gcphcpctl"

test: ## Run unit tests
	go test -race ./...

lint: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -rf bin/ dist/

release-artifacts: ## Build release tarballs and SHA256SUMS into dist/ (usage: make release-artifacts VERSION=v0.2.0)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building gcphcpctl for $${os}/$${arch}"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" \
			-o $(DIST)/gcphcpctl ./cmd/gcphcpctl; \
		COPYFILE_DISABLE=1 tar $(TAR_FLAGS) -cf - -C $(DIST) gcphcpctl \
			| gzip -n > "$(DIST)/gcphcpctl_$${os}_$${arch}.tar.gz"; \
		rm "$(DIST)/gcphcpctl"; \
	done
	@cd $(DIST) && $(SHA256SUM) *.tar.gz > SHA256SUMS
	@echo "Built release artifacts in $(DIST)/"

release: ## Tag and push a release (usage: make release v=0.2.0)
	@if [ -z "$(v)" ]; then echo "Usage: make release v=0.2.0"; exit 1; fi
	@echo "Tagging v$(v)..."
	git tag -a "v$(v)" -m "Release v$(v)"
	git push origin "v$(v)"
	@echo "Pushed v$(v)"
