_default:
    @just help

help:
    @just --list


test:
    go test ./...

setup:
    aqua i -l

lint:
    ghalint -c .ghalint.yml run
    actionlint
    goreleaser check

[group('Nix')]
update-vendor-hash:
    #!/usr/bin/env bash
    set -euo pipefail
    FAKE="sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
    sed -i "s|vendorHash = \"sha256-[^\"]*\";|vendorHash = \"${FAKE}\";|" flake.nix
    HASH=$(nix build .#miru 2>&1 | sed -n 's/.*got: *\(sha256-[A-Za-z0-9+/=]*\).*/\1/p' || true)
    if [ -z "$HASH" ]; then
        echo "Error: failed to get vendorHash" >&2
        sed -i "s|${FAKE}|sha256-TODO|" flake.nix
        exit 1
    fi
    sed -i "s|${FAKE}|${HASH}|" flake.nix
    echo "Updated vendorHash to: ${HASH}"

[group('Release')]
prerelease:
    go mod tidy
    gocredits --skip-missing -w .
    git add CREDITS
    if command -v nix >/dev/null 2>&1; then just update-vendor-hash && git add flake.nix; fi
