#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRANCH="${1:-shanfu-prod}"
HEAD_SHA="${2:-}"
HELPER="${ROOT_DIR}/tools/release_helper.sh"

if [[ ! -x "${HELPER}" ]]; then
  chmod +x "${HELPER}"
fi

RUN_ID="$("${HELPER}" latest-run "${BRANCH}" "${HEAD_SHA}")"
if [[ -z "${RUN_ID}" ]]; then
  if [[ -n "${HEAD_SHA}" ]]; then
    echo "no successful build run found for branch ${BRANCH} sha ${HEAD_SHA}" >&2
  else
    echo "no successful build run found for branch: ${BRANCH}" >&2
  fi
  exit 1
fi

DOWNLOAD_DIR="$("${HELPER}" download "${RUN_ID}")"
BINARY_PATH="${DOWNLOAD_DIR}/sub2api"

if [[ ! -f "${BINARY_PATH}" ]]; then
  if [[ -f "${DOWNLOAD_DIR}/dist/sub2api" ]]; then
    BINARY_PATH="${DOWNLOAD_DIR}/dist/sub2api"
  else
    echo "downloaded artifact missing binary: ${BINARY_PATH}" >&2
    exit 1
  fi
fi

"${HELPER}" deploy "${BINARY_PATH}"
echo "run ${RUN_ID} deployed from branch ${BRANCH}${HEAD_SHA:+ sha ${HEAD_SHA}}"
