#!/usr/bin/env bash
# This script is intended to be run in a docker container by the ../run-tests.sh script, not manually.
set -eu
export GO111MODULE=on
go install github.com/onsi/ginkgo/ginkgo
ginkgo ./pkgs/...
