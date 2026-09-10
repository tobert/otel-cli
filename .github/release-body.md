<!--
Rendered onto every release page by GoReleaser (--release-header-tmpl in
release.yml), above GitHub's auto-generated notes. GoReleaser templates it, so
.Tag is the git tag (v-prefixed) and .Version the same without the v — how the
image is tagged. This is the one surface where every placeholder is concrete —
the commands below carry the exact tag, ready to copy — so keep it that way: no
angle-bracket stand-ins. HTML comments don't render on GitHub.
-->
## Get it

Grab your platform's archive or package below ([install notes](https://github.com/tobert/otel-cli#getting-started)), or:

```sh
brew install tobert/otel-cli/otel-cli
go install github.com/tobert/otel-cli@{{ .Tag }}
docker pull ghcr.io/tobert/otel-cli:{{ .Version }}
```

(Pull version tags like `{{ .Version }}` — the `sha256-*` tags on the package page are
cosign signature/attestation artifacts riding alongside the image, not images.)

## Verify it

Run these from the folder holding your downloads. Every artifact carries SLSA
build provenance — any downloaded file, one command:

```sh
gh attestation verify otel-cli_{{ .Version }}_linux_amd64.tar.gz -R tobert/otel-cli
gh attestation verify oci://ghcr.io/tobert/otel-cli:{{ .Version }} -R tobert/otel-cli
```

Or keyless-verify the signed checksum manifest with cosign ≥ 2.5 (covers every file it lists, works offline):

```sh
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/tobert/otel-cli/.github/workflows/release.yml@refs/tags/{{ .Tag }}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  checksums.txt
sha256sum -c --ignore-missing checksums.txt
```

---
