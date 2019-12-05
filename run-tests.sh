#!/bin/bash
set -exu

DOCKER_IMAGE=golang:1.13.3-stretch

kafka_repo_root="$(realpath "$(dirname "$0")")"
vendor_dir="${kafka_repo_root}/shared/vendor"
operators_repo_root="$(realpath $1)"
zk_operator_dir=${operators_repo_root}/repository/zookeeper/operator
kafka_operator_dir=${operators_repo_root}/repository/kafka/operator

echo "Starting the tests with framework resources:"
ls ${operators_repo_root}
ls ${zk_operator_dir}
ls ${kafka_operator_dir}

# run KUDO Kafka utils unit tests
docker run --rm \
	-w ${kafka_repo_root}/images/kafka-utils \
	-v ${kafka_repo_root}:${kafka_repo_root} \
	${DOCKER_IMAGE} \
	bash -c ${kafka_repo_root}/images/kafka-utils/run-tests.sh

# run KUDO Kafka integration tests
docker run --rm \
	-w ${kafka_repo_root}/tests \
	-e KUBECONFIG=/root/.kube/config \
	-e ZK_FRAMEWORK_DIR=${zk_operator_dir} \
	-e KAFKA_FRAMEWORK_DIR=${kafka_operator_dir} \
	-e DS_KUDO_VERSION=${DS_KUDO_VERSION} \
	-e KUBECTL_PATH=${vendor_dir}/kubectl.sh  \
	-e VENDOR_DIR=${vendor_dir} \
	-e REPO_ROOT=${kafka_repo_root} \
	-v ${KUBECONFIG}:/root/.kube/config:ro \
	-v ${zk_operator_dir}:${zk_operator_dir}:ro \
	-v ${kafka_operator_dir}:${kafka_operator_dir}:ro \
	-v ${vendor_dir}:${vendor_dir} \
	-v ${kafka_repo_root}:${kafka_repo_root} \
	${DOCKER_IMAGE} \
	bash -c ${kafka_repo_root}/tests/run.sh
