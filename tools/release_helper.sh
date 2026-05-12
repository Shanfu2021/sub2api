#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACT_DIR_DEFAULT="${ROOT_DIR}/.tmp/release-artifacts"

usage() {
  cat <<'EOF'
Usage:
  tools/release_helper.sh push <branch>
  tools/release_helper.sh dispatch <branch>
  tools/release_helper.sh latest-run <branch> [head-sha]
  tools/release_helper.sh wait-run <run-id> [timeout-seconds]
  tools/release_helper.sh download <run-id> [output-dir]
  tools/release_helper.sh deploy <binary-path>

Required env:
  GITHUB_TOKEN   GitHub PAT with repo/actions access
  GITHUB_OWNER   e.g. Shanfu2021
  GITHUB_REPO    e.g. sub2api

Optional env:
  SERVICE_NAME   default: sub2api.service
  INSTALL_PATH   default: /opt/sub2api/sub2api
  INSTALL_USER   default: sub2api
  INSTALL_GROUP  default: sub2api
EOF
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing env: ${name}" >&2
    exit 1
  fi
}

gh_api() {
  require_env GITHUB_TOKEN
  require_env GITHUB_OWNER
  require_env GITHUB_REPO
  curl -fsSL \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "$@"
}

workflow_name() {
  printf '%s\n' "Build Self-Hosted Binary"
}

get_branch_sha() {
  local branch="${1:-}"
  gh_api "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/branches/${branch}" \
    | python3 -c 'import json,sys; data=json.load(sys.stdin); print(data["commit"]["sha"])'
}

find_success_run_id() {
  local branch="${1:-}"
  local head_sha="${2:-}"
  local wf_name
  wf_name="$(workflow_name)"
  gh_api "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/runs?branch=${branch}&per_page=30" \
    | python3 - "$head_sha" "$wf_name" <<'PY'
import json
import sys

head_sha = (sys.argv[1] or "").strip()
workflow_name = sys.argv[2]
runs = json.load(sys.stdin).get("workflow_runs", [])
for run in runs:
    if run.get("name") != workflow_name:
        continue
    if run.get("status") != "completed" or run.get("conclusion") != "success":
        continue
    if head_sha and run.get("head_sha") != head_sha:
        continue
    print(run["id"])
    break
PY
}

find_dispatched_run_id() {
  local branch="${1:-}"
  local head_sha="${2:-}"
  local wf_name
  wf_name="$(workflow_name)"
  gh_api "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/runs?branch=${branch}&per_page=30" \
    | python3 - "$head_sha" "$wf_name" <<'PY'
import json
import sys

head_sha = (sys.argv[1] or "").strip()
workflow_name = sys.argv[2]
runs = json.load(sys.stdin).get("workflow_runs", [])
for run in runs:
    if run.get("name") != workflow_name:
        continue
    if run.get("event") != "workflow_dispatch":
        continue
    if head_sha and run.get("head_sha") != head_sha:
        continue
    print(run["id"])
    break
PY
}

cmd_push() {
  local branch="${1:-}"
  if [[ -z "${branch}" ]]; then
    echo "branch required" >&2
    exit 1
  fi
  require_env GITHUB_TOKEN
  require_env GITHUB_OWNER
  require_env GITHUB_REPO
  git -C "${ROOT_DIR}" push "https://${GITHUB_OWNER}:${GITHUB_TOKEN}@github.com/${GITHUB_OWNER}/${GITHUB_REPO}.git" "HEAD:refs/heads/${branch}"
}

cmd_dispatch() {
  local branch="${1:-}"
  if [[ -z "${branch}" ]]; then
    echo "branch required" >&2
    exit 1
  fi
  local head_sha
  head_sha="$(get_branch_sha "${branch}")"
  gh_api \
    -X POST \
    "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/workflows/build-selfhosted-binary.yml/dispatches" \
    -d "{\"ref\":\"${branch}\",\"inputs\":{\"ref\":\"${branch}\"}}"

  local run_id=""
  local i
  for i in {1..30}; do
    run_id="$(find_dispatched_run_id "${branch}" "${head_sha}")"
    if [[ -n "${run_id}" ]]; then
      printf '%s\n' "${run_id}"
      return 0
    fi
    sleep 2
  done

  echo "dispatched workflow but failed to resolve run id for branch ${branch} sha ${head_sha}" >&2
  exit 1
}

cmd_latest_run() {
  local branch="${1:-}"
  local head_sha="${2:-}"
  if [[ -z "${branch}" ]]; then
    echo "branch required" >&2
    exit 1
  fi
  find_success_run_id "${branch}" "${head_sha}"
}

cmd_wait_run() {
  local run_id="${1:-}"
  local timeout_sec="${2:-1800}"
  local started_at
  started_at="$(date +%s)"
  if [[ -z "${run_id}" ]]; then
    echo "run id required" >&2
    exit 1
  fi

  while true; do
    local payload
    payload="$(gh_api "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/runs/${run_id}")"
    local parsed
    parsed="$(printf '%s' "${payload}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("status",""), d.get("conclusion") or "", d.get("html_url",""), d.get("head_sha",""))')"
    local status conclusion html_url head_sha
    read -r status conclusion html_url head_sha <<<"${parsed}"

    echo "run ${run_id}: status=${status} conclusion=${conclusion:-null} sha=${head_sha}" >&2

    if [[ "${status}" == "completed" ]]; then
      if [[ "${conclusion}" == "success" ]]; then
        printf '%s\n' "${run_id}"
        return 0
      fi
      echo "run ${run_id} failed: ${html_url}" >&2
      return 1
    fi

    local now
    now="$(date +%s)"
    if (( now - started_at >= timeout_sec )); then
      echo "timed out waiting for run ${run_id}: ${html_url}" >&2
      return 1
    fi

    sleep 10
  done
}

cmd_download() {
  local run_id="${1:-}"
  local out_dir="${2:-$ARTIFACT_DIR_DEFAULT}"
  if [[ -z "${run_id}" ]]; then
    echo "run id required" >&2
    exit 1
  fi

  mkdir -p "${out_dir}"
  local artifacts_json
  artifacts_json="$(gh_api "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/runs/${run_id}/artifacts")"
  local artifact_id
  artifact_id="$(printf '%s' "${artifacts_json}" | python3 -c 'import json,sys; arts=json.load(sys.stdin).get("artifacts",[]); art=next((a for a in arts if a.get("name")=="sub2api-linux-amd64"), None); print("" if art is None else art["id"])')"
  if [[ -z "${artifact_id}" ]]; then
    echo "artifact sub2api-linux-amd64 not found for run ${run_id}" >&2
    exit 1
  fi

  local zip_path="${out_dir}/artifact-${run_id}.zip"
  rm -f "${zip_path}"
  curl -fsSL \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -o "${zip_path}" \
    "https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/actions/artifacts/${artifact_id}/zip"

  rm -rf "${out_dir}/run-${run_id}"
  mkdir -p "${out_dir}/run-${run_id}"
  python3 - "${zip_path}" "${out_dir}/run-${run_id}" <<'PY'
import sys
import shutil
import zipfile
from pathlib import Path

zip_path = Path(sys.argv[1])
out_dir = Path(sys.argv[2])
out_dir.mkdir(parents=True, exist_ok=True)
with zipfile.ZipFile(zip_path) as zf:
    zf.extractall(out_dir)

root_binary = out_dir / "sub2api"
dist_binary = out_dir / "dist" / "sub2api"
if not root_binary.exists() and dist_binary.exists():
    shutil.copy2(dist_binary, root_binary)
PY
  printf '%s\n' "${out_dir}/run-${run_id}"
}

cmd_deploy() {
  local binary_path="${1:-}"
  local service_name="${SERVICE_NAME:-sub2api.service}"
  local install_path="${INSTALL_PATH:-/opt/sub2api/sub2api}"
  local install_user="${INSTALL_USER:-sub2api}"
  local install_group="${INSTALL_GROUP:-sub2api}"

  if [[ -z "${binary_path}" ]]; then
    echo "binary path required" >&2
    exit 1
  fi
  if [[ -d "${binary_path}" ]]; then
    if [[ -f "${binary_path}/sub2api" ]]; then
      binary_path="${binary_path}/sub2api"
    elif [[ -f "${binary_path}/dist/sub2api" ]]; then
      binary_path="${binary_path}/dist/sub2api"
    else
      echo "binary directory does not contain sub2api: ${binary_path}" >&2
      exit 1
    fi
  fi
  if [[ ! -f "${binary_path}" ]]; then
    echo "binary not found: ${binary_path}" >&2
    exit 1
  fi

  install -m 755 -o "${install_user}" -g "${install_group}" "${binary_path}" "${install_path}"
  systemctl restart "${service_name}"
  systemctl is-active --quiet "${service_name}"
  echo "deployed ${binary_path} -> ${install_path}"
}

main() {
  local cmd="${1:-}"
  shift || true
  case "${cmd}" in
    push) cmd_push "$@" ;;
    dispatch) cmd_dispatch "$@" ;;
    latest-run) cmd_latest_run "$@" ;;
    wait-run) cmd_wait_run "$@" ;;
    download) cmd_download "$@" ;;
    deploy) cmd_deploy "$@" ;;
    ""|-h|--help|help) usage ;;
    *)
      echo "unknown command: ${cmd}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
