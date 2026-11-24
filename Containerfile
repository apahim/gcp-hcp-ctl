FROM registry.access.redhat.com/ubi9/go-toolset:1.27.3-1795320499 AS builder

ARG APP_VERSION=dev
ARG APP_COMMIT=unknown
ARG APP_DATE=

WORKDIR /opt/app-root/src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

RUN set -eux; \
    case "$APP_VERSION" in refs/tags/*) APP_VERSION="${APP_VERSION#refs/tags/}";; esac; \
    [ -n "$APP_DATE" ] || APP_DATE="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"; \
    SAFE_VERSION="${APP_VERSION//\//-}"; \
    MODULE=github.com/openshift-online/gcp-hcp-ctl; \
    LDFLAGS="-X ${MODULE}/pkg/cli.version=${APP_VERSION} -X ${MODULE}/pkg/cli.commit=${APP_COMMIT} -X ${MODULE}/pkg/cli.date=${APP_DATE}"; \
    mkdir -p bin releases; \
    for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
      os=$(echo "$platform" | cut -d/ -f1); \
      arch=$(echo "$platform" | cut -d/ -f2); \
      out="bin/gcphcpctl_${os}_${arch}"; \
      CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -buildvcs=false -ldflags "$LDFLAGS" -o "$out" ./cmd/gcphcpctl; \
      cp "$out" bin/gcphcpctl; \
      tar --mtime=@0 --owner=0 --group=0 --numeric-owner -cf - -C bin gcphcpctl | gzip -n > "releases/gcphcpctl_${os}_${arch}.tar.gz"; \
      rm bin/gcphcpctl; \
    done; \
    ( cd releases && sha256sum *.tar.gz > "gcphcpctl_${SAFE_VERSION}_SHA256SUMS" )

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ARG APP_VERSION=dev
ARG APP_COMMIT=unknown

LABEL org.opencontainers.image.title="gcp-hcp-ctl" \
      org.opencontainers.image.description="CLI for managing GCP Hosted Control Plane (HCP) clusters" \
      org.opencontainers.image.source="https://github.com/openshift-online/gcp-hcp-ctl" \
      org.opencontainers.image.vendor="Red Hat" \
      org.opencontainers.image.revision="${APP_COMMIT}" \
      com.redhat.component="gcp-hcp-ctl" \
      name="gcp-hcp-ctl" \
      version="0.1" \
      release="1" \
      vcs-type="git" \
      vcs-ref="${APP_COMMIT}" \
      url="https://github.com/openshift-online/gcp-hcp-ctl" \
      vendor="Red Hat, Inc." \
      distribution-scope="release" \
      summary="CLI for managing GCP Hosted Control Plane (HCP) clusters" \
      description="CLI for managing GCP Hosted Control Plane (HCP) clusters" \
      io.k8s.description="CLI for managing GCP Hosted Control Plane (HCP) clusters" \
      io.k8s.display-name="gcp-hcp-ctl" \
      cpe="cpe:/a:redhat:gcp_hcp_ctl:1::el9"

COPY --from=builder /opt/app-root/src/releases /releases
USER 1001
