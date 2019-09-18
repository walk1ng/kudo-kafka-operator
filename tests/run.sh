#!/usr/bin/env bash
# This script is intended to be run in a docker container by the ../run-tests.sh script, not manually.
set -eu
export GO111MODULE=on
export PATH=${PATH}:/${VENDOR_DIR}
go mod edit -require github.com/kudobuilder/kudo@${DS_KUDO_VERSION}
go install github.com/onsi/ginkgo/ginkgo
${KUBECTL_PATH} kudo version

ginkgo ./suites/... ${TESTS_FOCUS:+--ginkgo.focus=${TESTS_FOCUS}}
