#!/usr/bin/env bash
set -e
export GO111MODULE=on
export PATH=${PATH}:/${VENDOR_DIR}
go mod edit -require github.com/kudobuilder/kudo@${DS_KUDO_VERSION}
go install github.com/onsi/ginkgo/ginkgo
${KUBECTL_PATH} kudo version

if [[ ! -z $TESTS_FOCUS ]]; then
	ginkgo ./suites/... --ginkgo.focus=${TESTS_FOCUS}
else
	ginkgo ./suites/...
fi
