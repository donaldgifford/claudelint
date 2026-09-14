#!/usr/bin/env bash
# spec-drift-issue.sh - Report upstream spec drift through one tracking issue
#
# Keeps exactly one open issue carrying the `spec-drift` label. The body
# begins with a hidden marker holding a hash of the material changes, so
# a scheduled run that finds the same drift as last week stays silent
# instead of nagging (DESIGN-0006 §6, OQ3).
#
# Usage:
#   ./scripts/spec-drift-issue.sh <exit-code> <report.md> <diff.json>
#   ./scripts/spec-drift-issue.sh --dry-run 1 report.md diff.json
#
# Behaviour by specdrift exit code:
#   0  an open issue is commented on and closed; no issue is a no-op
#   1  no open issue creates one; a different marker comments and
#      updates the body; the same marker does nothing
#   2  no issue churn at all; the workflow fails the job instead
#
# Options:
#   --dry-run    Print the gh commands instead of running them
#   --help       Show this help message

set -euo pipefail

LABEL="spec-drift"
LABEL_COLOR="1D76DB"
LABEL_DESCRIPTION="Upstream Claude Code spec drift detected by specdrift"
TITLE="Upstream spec drift"
MARKER_PREFIX="<!-- specdrift:diff-sha256="

DRY_RUN=false

log() { echo "==> $*"; }

die() {
  echo "spec-drift-issue: $*" >&2
  exit 1
}

show_help() {
  sed -n '/^# spec-drift-issue.sh/,/^$/p' "$0" | sed 's/^# \?//'
  exit 0
}

# gh_run executes a gh command, or prints it under --dry-run.
gh_run() {
  if [[ "${DRY_RUN}" == true ]]; then
    printf 'would run: gh'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi
  gh "$@"
}

check_dependencies() {
  local missing=()
  command -v gh >/dev/null 2>&1 || missing+=("gh")
  command -v jq >/dev/null 2>&1 || missing+=("jq")
  command -v sha256sum >/dev/null 2>&1 || missing+=("sha256sum")

  if [[ ${#missing[@]} -gt 0 ]]; then
    die "missing required commands: ${missing[*]}"
  fi
}

# marker_of hashes only the material changes. The report also lists the
# sources whose bytes moved, and those move on prose edits that change
# no fact; hashing the whole file would reopen the conversation every
# week for the same drift.
marker_of() {
  local diff_file="$1"
  local hash

  hash=$(jq -S -c '.changes' "${diff_file}" | sha256sum | cut -d' ' -f1)
  printf '%s%s -->' "${MARKER_PREFIX}" "${hash}"
}

ensure_label() {
  if gh label list --limit 200 --json name --jq '.[].name' 2>/dev/null |
    grep -qx "${LABEL}"; then
    return 0
  fi

  log "creating the ${LABEL} label"
  gh_run label create "${LABEL}" \
    --color "${LABEL_COLOR}" \
    --description "${LABEL_DESCRIPTION}"
}

# open_issue_number prints the number of the single open spec-drift
# issue, or nothing.
open_issue_number() {
  gh issue list --state open --label "${LABEL}" --limit 1 \
    --json number --jq '.[0].number // empty'
}

open_issue_body() {
  gh issue view "$1" --json body --jq '.body'
}

# body_with_marker assembles an issue body: the hidden marker, the
# rendered report, and a line saying what to do about it.
body_with_marker() {
  local marker="$1" report="$2"

  printf '%s\n\n' "${marker}"
  cat "${report}"
  cat <<'FOOTER'

---

This issue is maintained by the `Upstream Spec Drift` workflow. It is
updated when the drift changes and closed when it is resolved.

To act on it: run `just spec-sync` to regenerate
`internal/upstream/digest.json` and `internal/upstream/sources.lock.json`,
then update the affected rules or known-data sets in the same pull
request. See [DESIGN-0006](docs/design/0006-upstream-spec-drift-detection.md).
FOOTER
}

create_issue() {
  local marker="$1" report="$2" body

  body=$(body_with_marker "${marker}" "${report}")

  log "opening a new ${LABEL} issue"
  gh_run issue create \
    --title "${TITLE}" \
    --label "${LABEL}" \
    --body "${body}"
}

update_issue() {
  local number="$1" marker="$2" report="$3" body comment

  body=$(body_with_marker "${marker}" "${report}")
  comment=$(printf 'The drift changed. Current report:\n\n%s\n' "$(cat "${report}")")

  log "commenting on and updating issue #${number}"
  gh_run issue comment "${number}" --body "${comment}"
  gh_run issue edit "${number}" --body "${body}"
}

close_issue() {
  local number="$1"

  log "closing issue #${number}"
  gh_run issue comment "${number}" \
    --body "No drift remains: resolved by upstream or by a \`just spec-sync\`. Closing."
  gh_run issue close "${number}"
}

handle_clean() {
  local number="$1"

  if [[ -z "${number}" ]]; then
    log "no drift and no open issue; nothing to do"
    return 0
  fi

  close_issue "${number}"
}

handle_drift() {
  local number="$1" marker="$2" report="$3"

  if [[ -z "${number}" ]]; then
    create_issue "${marker}" "${report}"
    return 0
  fi

  if open_issue_body "${number}" | grep -qF "${marker}"; then
    log "issue #${number} already reports this drift; staying quiet"
    return 0
  fi

  update_issue "${number}" "${marker}" "${report}"
}

main() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      --help | -h)
        show_help
        ;;
      *)
        break
        ;;
    esac
  done

  [[ $# -eq 3 ]] || die "usage: $0 [--dry-run] <exit-code> <report.md> <diff.json>"

  local code="$1" report="$2" diff_file="$3"

  check_dependencies

  # Exit 2 means the tool could not decide. Acting on a report it did
  # not finish writing would be worse than silence; the workflow fails
  # the job instead.
  if [[ "${code}" == "2" ]]; then
    log "specdrift could not decide (exit 2); leaving the issue untouched"
    return 0
  fi

  [[ -f "${report}" ]] || die "report not found: ${report}"
  [[ -f "${diff_file}" ]] || die "diff not found: ${diff_file}"

  ensure_label

  local number
  number=$(open_issue_number || true)

  case "${code}" in
    0) handle_clean "${number}" ;;
    1) handle_drift "${number}" "$(marker_of "${diff_file}")" "${report}" ;;
    *) die "unexpected specdrift exit code: ${code}" ;;
  esac
}

main "$@"
