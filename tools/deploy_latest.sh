#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRANCH="${1:-shanfu-prod}"
HELPER="${ROOT_DIR}/tools/release_helper.sh"

if [[ ! -x "${HELPER}" ]]; then
  chmod +x "${HELPER}"
fi

RUN_ID="$("${HELPER}" latest-run "${BRANCH}")"
if [[ -z "${RUN_ID}" ]]; then
  echo "no build run found for branch: ${BRANCH}" >&2
  exit 1
fi

DOWNLOAD_DIR="$("${HELPER}" download "${RUN_ID}")"
BINARY_PATH="${DOWNLOAD_DIR}/sub2api"

if [[ ! -f "${BINARY_PATH}" ]]; then
  echo "downloaded artifact missing binary: ${BINARY_PATH}" >&2
  exit 1
fi

"${HELPER}" deploy "${BINARY_PATH}"
echo "run ${RUN_ID} deployed from branch ${BRANCH}"
