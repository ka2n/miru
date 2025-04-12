_default:
    @just help

help:
    @just --list


test:
    go test ./...

setup:
    aqua i -l

[group('Release')]
prerelease:
    gocredits --skip-missing -w .
    git add CREDITS
