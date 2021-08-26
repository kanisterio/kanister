#!/usr/bin/env sh

readonly IMAGE_TAG="ghcr.io/kanisterio/license-extractor:$(git rev-parse --short=7 HEAD:docker/license_extractor)"

docker build -t "${IMAGE_TAG}" .
docker push "${IMAGE_TAG}"
