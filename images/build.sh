#!/usr/bin/env bash

set -eu

function print_help() {
  echo "Usage: ${0} [OPTIONS]"
  echo
  echo "OPTIONS:"
  echo "--push             push the recently build image"
  echo "--image=IMAGE      docker image to build. valid options are '--image=kafka' and '--image=cruise-control'"
  echo
}

source versions.sh

PUSH_IMAGE="false"
IMAGE_NAME="all"
for arg in "$@"
do
    case $arg in
        -p|--push|push)
        PUSH_IMAGE="true"
        shift
        ;;
        -i=*|--image=*)
        IMAGE_NAME="${arg#*=}"
        shift
        ;;
        *)
        print_help
        exit 1
        ;;
    esac
done

case ${IMAGE_NAME} in
all)
  BUILD_KAFKA="true"
  BUILD_CRUISE="true"
  ;;
kafka)
  BUILD_KAFKA="true"
  ;;
cruise-control)
  BUILD_CRUISE="true"
  ;;
*)
  echo "cannot build image $IMAGE_NAME"
  print_help
  exit 1
  ;;
esac

if [[ "${BUILD_KAFKA}" == "true" ]]; then
  docker image build --build-arg KAFKA_VERSION=${KAFKA_VERSION} -t mesosphere/kafka:${KAKFA_TAG_VERSION} ./kafka
fi
if [[ "${BUILD_CRUISE}" == "true" ]]; then
  docker image build --build-arg CRUISE_CONTROL_VERSION=${CRUISE_CONTROL_VERSION} --build-arg CRUISE_CONTROL_UI_VERSION=${CRUISE_CONTROL_UI_VERSION} \
    -t mesosphere/cruise-control:${CRUISE_CONTROL_TAG_VERSION} ./cruise-control
fi

if [[ "${PUSH_IMAGE}" == "true" ]]; then
  if [[ "${BUILD_CRUISE}" == "true" ]]; then
    docker push mesosphere/cruise-control:${CRUISE_CONTROL_TAG_VERSION}
  fi
  if [[ "${BUILD_KAFKA}" == "true" ]]; then
    docker push mesosphere/kafka:${KAFKA_TAG_VERSION}
  fi
else
  echo "Image built successfully."
  echo "To push the image use the '--push' flag or 'push' arg"
fi

exit 0
