# otel-cli release image. The top-level Dockerfile builds from source for local
# development; this one only packages binaries that GoReleaser already built.
#
# Built multiarch (amd64 + arm64) by .github/workflows/release.yml, which stages
# each Linux binary under binaries/<docker arch>/ — pure COPY, no RUN, so neither
# platform needs emulation to build.
#
# Alpine rather than scratch/distroless so the image carries the CA trust store
# (ca-certificates-bundle is in the base) and a shell for debugging inside the
# container, as the README has documented since 0.4.2. Pinned by manifest-list
# digest, same discipline as the workflow's action pins; Dependabot bumps it.
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Links the ghcr package to the repo even for a manual build; the workflow's
# metadata-action adds the full OCI label set on top.
LABEL org.opencontainers.image.source="https://github.com/tobert/otel-cli" \
      org.opencontainers.image.description="OpenTelemetry command-line tool for sending events from shell scripts & similar environments" \
      org.opencontainers.image.licenses="Apache-2.0"

ARG TARGETARCH
COPY binaries/${TARGETARCH}/otel-cli /otel-cli

ENTRYPOINT ["/otel-cli"]
