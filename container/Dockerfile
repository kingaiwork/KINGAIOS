# syntax=docker/dockerfile:1.7

ARG GO_IMAGE=golang:1.25.13-bookworm
ARG BASE_IMAGE=ubuntu:26.04
ARG VERSION

FROM --platform=$BUILDPLATFORM ${GO_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION
RUN set -eux; \
    version="${VERSION:-$(tr -d '[:space:]' < VERSION)}"; \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath -tags osusergo -ldflags "-s -w -buildid= -X main.version=${version}" -o /out/kingai ./cmd/kingai; \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" go build -trimpath -tags osusergo -ldflags "-s -w -buildid= -X main.version=${version}" -o /out/kingaid ./cmd/kingaid

FROM ${BASE_IMAGE}
ARG VERSION=dev
ARG REVISION=unknown
ARG KINGAI_UID=10001
ARG KINGAI_GID=10001
LABEL org.opencontainers.image.title="KINGAI OS Container" \
      org.opencontainers.image.description="KINGAI OS D5 runtime for Docker and OCI environments" \
      org.opencontainers.image.source="https://github.com/kingaiwork/KINGAIOS" \
      org.opencontainers.image.vendor="KINGAI" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"

RUN set -eux; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd --gid "${KINGAI_GID}" kingai; \
    useradd --uid "${KINGAI_UID}" --gid "${KINGAI_GID}" --home-dir /var/lib/kingai --no-create-home --shell /usr/sbin/nologin _kingai; \
    install -d -o _kingai -g kingai -m 0750 \
      /var/lib/kingai \
      /var/lib/kingai/approvals \
      /var/lib/kingai/memory \
      /var/lib/kingai/tasks \
      /var/lib/kingai/models \
      /var/log/kingai \
      /run/kingai; \
    install -d -o root -g root -m 0755 /etc/kingai /usr/lib/kingai /usr/share/doc/kingai-os

COPY --from=build --chown=root:root /out/kingai /usr/bin/kingai
COPY --from=build --chown=root:root /out/kingaid /usr/lib/kingai/kingaid
COPY --chown=root:root configs/policy.json /etc/kingai/policy.json
COPY --chown=root:root configs/agents.json /etc/kingai/agents.json
COPY --chown=root:root configs/models.json /etc/kingai/models.json
COPY --chown=root:root configs/system.json /etc/kingai/system.json
COPY --chown=root:root LICENSE NOTICE /usr/share/doc/kingai-os/

RUN chmod 0755 /usr/bin/kingai /usr/lib/kingai/kingaid && \
    chmod 0644 /etc/kingai/*.json /usr/share/doc/kingai-os/*

ENV HOME=/var/lib/kingai \
    TMPDIR=/tmp \
    KINGAI_SOCKET=/run/kingai/kingaid.sock \
    KINGAI_POLICY=/etc/kingai/policy.json \
    KINGAI_AGENTS=/etc/kingai/agents.json \
    KINGAI_MODELS=/etc/kingai/models.json \
    KINGAI_AUDIT=/var/log/kingai/audit.jsonl \
    KINGAI_PUBLIC_STATUS=/run/kingai/public-status.json \
    KINGAI_APPROVAL_ROOT=/var/lib/kingai/approvals \
    KINGAI_MEMORY_ROOT=/var/lib/kingai/memory \
    KINGAI_TASK_ROOT=/var/lib/kingai/tasks

VOLUME ["/var/lib/kingai", "/var/log/kingai"]
STOPSIGNAL SIGTERM
USER _kingai:kingai
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/usr/bin/kingai", "model", "status"]
ENTRYPOINT ["/usr/lib/kingai/kingaid"]
