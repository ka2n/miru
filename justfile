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

[group('Release')]
prerelease:
    go mod tidy
    gocredits --skip-missing -w .
    git add CREDITS
