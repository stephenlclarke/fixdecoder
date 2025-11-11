#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# Unified CI helper for the fixdecoder project
# ---------------------------------------------------------------------------

set -eo pipefail

# ──────────────────────────────────────────────────────────────
#  Constants
# ──────────────────────────────────────────────────────────────
declare application_name="fixdecoder"

declare modules_list=(
	"cmd/${application_name}"
	"decoder"
	"fix"
	"fix/fix40"
	"fix/fix41"
	"fix/fix42"
	"fix/fix43"
	"fix/fix44"
	"fix/fix50"
	"fix/fix50SP1"
	"fix/fix50SP2"
	"fix/fixT11"
)

declare common_preparation=false
declare unit_tests=false
declare integration_tests=false
declare build_release=false
declare code_scan=false
declare compile_binary=false
declare generate_fix=false

function log_message {
  echo -e "\n\033[1;32m$1\033[0m"
}

function log_warning {
  echo -e "\n\033[38;5;214m$1\033[0m"
}

function log_error {
  echo -e "\n\033[1;31m$1\033[0m"
}

# Helper that runs the common pre-build steps in order
function common_preparation() {
  if [[ ${common_preparation} == true ]]; then
    return
  fi

  log_message ">> Setting up Go environment"

  export GOPATH=$(go env GOPATH)
  export PATH="$(go env GOPATH)/bin:${PATH}"

  go env -w GOPRIVATE=github.com/stephenlclarke/${application_name}

  log_message ">> Installing test dependencies"
  go install gotest.tools/gotestsum@latest

  log_message ">> Running go mod tidy"
  go mod tidy

  log_message ">> Running go mod download"
  go mod download

  common_preparation=true
}

function generate_fix() {
    if [[ ${generate_fix} == true ]]; then
    return
  fi

  log_message ">> Auto-Generating FIX dictionary"
  chmod +x ./resources/generate_fix_go.sh
  ./resources/generate_fix_go.sh

  log_message ">> Auto-Generating FIX sensitive tags"
  go run ./cmd/generateSensitiveTagNames/main.go

  generate_fix=true
}

function unit_tests() {
  if [[ ${unit_tests} == true ]]; then
    return
  fi

  log_message ">> Running unit tests"
  mkdir -p unit-test-reports
  : > unit-test-reports/coverage.unit.out.tmp

  # Ensure gotestsum is available
  if ! command -v gotestsum >/dev/null 2>&1; then
    log_message ">> Installing gotestsum (missing on PATH)"
    go install gotest.tools/gotestsum@latest
  fi

  # Run tests per module once (coverage + JUnit via gotestsum)
  for module in "${modules_list[@]}"; do
    log_message " - Testing ${module}"

    report_name="${module//\//-}"
    cov_path="$(cd unit-test-reports && pwd)/coverage-${report_name}.out"
    junit_path="$(cd unit-test-reports && pwd)/test-report-${report_name}.xml"

    pushd "${module}" >/dev/null

    # Allow extra flags via GO_TEST_FLAGS, e.g. "-race -shuffle=on"
    if ! gotestsum --format=standard-verbose --junitfile "${junit_path}" -- \
         -covermode=atomic -coverprofile="${cov_path}" ${GO_TEST_FLAGS:-} .; then
      popd >/dev/null
      log_message "!! Unit tests failed in ${module}"
      return 1
    fi

    popd >/dev/null
  done

  # Merge coverage safely (only our generated files, skip empty)
  log_message ">> Merging coverage reports"
  echo "mode: atomic" > unit-test-reports/coverage.unit.out
  while IFS= read -r f; do
    [[ -s "$f" ]] || continue
    tail -n +2 "$f" >> unit-test-reports/coverage.unit.out
  done < <(find unit-test-reports -maxdepth 1 -type f -name 'coverage-*.out' | sort)

  # Optional: single aggregated JUnit (re-runs tests) for dashboards that expect one file
  log_message ">> Generating aggregated unit test report (this re-runs tests)"
  if ! gotestsum --format=standard-verbose --junitfile unit-test-reports/unit-test-report.xml -- ${GO_TEST_FLAGS:-} ./...; then
    log_message "!! Aggregated unit test run failed"
    return 1
  fi

  # HTML coverage
  log_message ">> Generating HTML coverage report"
  go tool cover -html=unit-test-reports/coverage.unit.out -o unit-test-reports/coverage.html
  log_message "HTML report available at unit-test-reports/coverage.html"

  unit_tests=true
}

function compile_binary() {
  # ensure the bin directory exists
  mkdir -p ./bin

  # ensure your tags are fetched
  git fetch --tags

  local git_branch=$(git rev-parse --abbrev-ref HEAD)
  local git_short_sha=$(git rev-parse --short HEAD)
  local git_url=$(git remote get-url origin)
  local git_version=$(get_version)

  local operating_system=${1:-$(go env GOOS)}
  local architecture=${2:-$(go env GOARCH)}

  log_message ">> Building ${application_name} ${git_version} (branch: ${git_branch}, commit: ${git_short_sha}), OS: ${operating_system}, ARCH: ${architecture}"

  local build_tag=""
  [[ -z "$1" || -z "$2" ]] || build_tag="-${git_version#v}.${operating_system}-${architecture}"
  
  mkdir -p ./bin/${application_name}-${git_version#v}

  # build with that version baked in
  time env GOOS=${operating_system} GOARCH=${architecture} go build -ldflags="-X main.Version=${git_version} -X main.Branch=${git_branch} -X main.Sha=${git_short_sha} -X main.Url=${git_url}" -o ./bin/${application_name}-${git_version#v}/${application_name}${build_tag} ./cmd/${application_name}

  compile_binary=true
}

upload_artifacts() {
  # --- constants
  readonly artifact_dir="./bin"
  readonly s3_bucket="ewm-op"
  readonly s3_prefix="release"

  # --- temporarily relax strict mode inside this function only
  # save current shell option state and restore it on function exit
  local __saved_opts
  __saved_opts="$(set +o)"
  # disable errexit and pipefail locally so one warning/skip can't abort the loop
  set +e +o pipefail
  trap 'eval "$__saved_opts"' RETURN

  [[ -d "$artifact_dir" ]] || { log_error "artifact directory not found: $artifact_dir"; return 1; }

  local attempted=0
  local uploaded=0
  local seen_versions=""

  # Consider only filenames that look like artifacts; skip the version folder
  shopt -s nullglob
  for f in "${artifact_dir}"/*.*-*; do
    [[ -e "$f" ]] || continue
    if [[ ! -f "$f" ]]; then
      log_warning "skipping non-regular: ${f##*/}"
      continue
    fi

    local base="${f##*/}"

    # <app>-<semver>.<os>-<arch>  e.g. fixdecoder-2.0.7.linux-amd64
    if [[ "$base" =~ ^([^-]+)-([0-9]+\.[0-9]+\.[0-9]+)\.([^.]+)-([^.]+)$ ]]; then
      local app="${BASH_REMATCH[1]}"
      local semver="${BASH_REMATCH[2]}"
      local ver="v${semver}"
      local final_app="${application_name:-$app}"
      local s3_path="s3://${s3_bucket}/${s3_prefix}/${final_app}/${ver}/"

      ((attempted++))
      log_message "uploading ${base} -> ${s3_path}"
      if aws s3 cp "$f" "$s3_path"; then
        ((uploaded++))
        # de-dupe the (app/version) pairs we touched
        if [[ " $seen_versions " != *" $final_app/$ver "* ]]; then
          seen_versions+="${seen_versions:+ }${final_app}/${ver}"
        fi
      else
        log_error "failed to upload ${base}"
      fi
    else
      log_warning "skipping non-artifact: ${base}"
    fi
  done
  shopt -u nullglob

  if (( attempted == 0 )); then
    log_error "no candidate artifacts found in $artifact_dir"
    return 1
  fi
  if (( uploaded == 0 )); then
    log_error "no artifacts uploaded from $artifact_dir"
    return 1
  fi

  # List once per (app,version)
  for key in $seen_versions; do
    local app="${key%%/*}"
    local ver="${key#*/}"
    log_message "listing artifacts uploaded for ${app} ${ver}"
    aws s3 ls "s3://${s3_bucket}/${s3_prefix}/${app}/${ver}/" --recursive --human-readable --summarize || true
  done

  log_message "upload complete: ${uploaded}/${attempted} artifact(s) uploaded"
}

function integration_tests() {
  if [[ ${integration_tests} == true ]]; then
    return
  fi

  log_message ">> Running integration tests"

  mkdir -p integration-test-reports

  # Ensure gotestsum is available
  if ! command -v gotestsum >/dev/null 2>&1; then
    log_message ">> Installing gotestsum (missing on PATH)"
    go install gotest.tools/gotestsum@latest
  fi

  # Run integration tests across all pkgs; produce JUnit and coverage
  if ! gotestsum --format=standard-verbose --junitfile integration-test-reports/integration-test-report.xml -- \
       -tags=integration -covermode=atomic -coverpkg=./... -coverprofile=integration-test-reports/coverage.integration.out ./...; then
    log_message "!! Integration tests failed"
    return 1
  fi

  integration_tests=true
}

function code_scan() {
  if [[ -n "${BITBUCKET_BUILD_NUMBER:-}" ]]; then
    log_message ">> Skipping SonarQube scan in Bitbucket Pipelines"
    code_scan=true
    return
  fi

  if [[ "${code_scan:-false}" == true ]]; then
    return
  fi

  log_message ">> SonarQube Scan - Replacing autogenerated fix/fix*/*.go files with minimal stubs"
  
  git restore -SW fix/fix*/*.go

  if command -v sonar-scanner >/dev/null 2>&1; then
    log_message ">> Using local sonar-scanner"
    sonar-scanner -Dsonar.token="${SONAR_TOKEN}" "$@"
  else
    log_message ">> Local sonar-scanner not found, falling back to Docker image"
    docker run --rm \
      --platform=linux/amd64 \
      -e SONAR_TOKEN="${SONAR_TOKEN}" \
      -v "$(pwd):/usr/src" \
      sonarsource/sonar-scanner-cli "$@"
  fi

  code_scan=true
}

function clean_repo() {
  log_message ">> Cleaning repository (removing autogenerated files and binaries)"
  git restore -SW fix/fix*/*.go
  rm -rf ./bin/*
}

function copy_binaries() {
  log_message ">> Copying binaries"
  mkdir -p ./bin

  local git_version=$(get_version)

  cp ./bin/fixdecoder-${git_version#v}/fixdecoder* ./bin
}

# Generate a version like: v1.2.3[-dirty]|[-<branch>]
# - On main: append -dirty only if there are changes NOT covered by .gitignore
# - On other branches: always append -<sanitized-branch>
function get_version() {
  set -euo pipefail

  # Resolve branch (prefer Bitbucket vars in CI)
  local git_branch="${BITBUCKET_BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo '')}"
  if [ -n "${BITBUCKET_TAG:-}" ] && [ -z "$git_branch" ]; then
    git_branch="main"
  fi

  # Best-effort: refresh tags if in a git repo (non-fatal)
  if git rev-parse --git-dir >/dev/null 2>&1; then
    git fetch --tags --force --prune >/dev/null 2>&1 || \
      echo "Warning: unable to fetch tags, continuing with local tags" >&2
  fi

  # Base version (latest tag or v0.0.0)
  local git_tag
  git_tag="$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')"
  local git_version="$git_tag"

  # Build pathspec excludes from the repo’s top-level .gitignore
  # We:
  #   - drop comments (#...), blanks, and negations (!pattern)
  #   - trim trailing whitespace
  #   - strip leading '/' (we add :(top) instead)
  local -a EXCLUDES=()
  if git rev-parse --show-toplevel >/dev/null 2>&1; then
    local gitroot
    gitroot="$(git rev-parse --show-toplevel)"
    if [ -f "$gitroot/.gitignore" ]; then
      # shellcheck disable=SC2016
      while IFS= read -r pat; do
        # remove leading slash to work with :(top)
        pat="${pat#/}"
        # skip if empty after strip
        [ -z "$pat" ] && continue
        # create a top-anchored, glob-enabled exclude pathspec
        EXCLUDES+=(":(top,glob,exclude)$pat")
      done < <(sed -e 's/[[:space:]]\+$//' \
                  -e '/^\s*#/d' \
                  -e '/^\s*$/d' \
                  -e '/^\s*!/d' "$gitroot/.gitignore")
    fi
  fi

  if [ "$git_branch" = "main" ]; then
    local repo_dirty=false

    # 1) Unstaged tracked changes (excluding .gitignore patterns)
    if ! git diff --quiet --ignore-submodules HEAD -- . "${EXCLUDES[@]:-}"; then
      repo_dirty=true
    fi

    # 2) Staged changes (excluding .gitignore patterns)
    if ! git diff --quiet --cached --ignore-submodules -- . "${EXCLUDES[@]:-}"; then
      repo_dirty=true
    fi

    # 3) Any untracked files?  (git status already excludes .gitignore)
    if [ "$repo_dirty" = false ]; then
      if git status --porcelain --untracked-files=all | grep -qE '^\?\? '; then
        repo_dirty=true
      fi
    fi

    # 4) Not exactly at a tag?
    if ! git describe --tags --exact-match >/dev/null 2>&1; then
      repo_dirty=true
    fi

    if [ "$repo_dirty" = true ]; then
      git_version="${git_version}-dirty"
    fi
  else
    # Any non-main branch => append sanitized branch name
    local safe_branch="${git_branch//\//-}"
    safe_branch="$(printf '%s' "$safe_branch" \
      | tr -c 'A-Za-z0-9._-' '-' \
      | sed -E 's/-{2,}/-/g; s/^-+//; s/-+$//')"
    [ -z "$safe_branch" ] && safe_branch="branch"
    git_version="${git_version}-${safe_branch}"
  fi

  # Output + export (useful in CI)
  export BUILD_VERSION="${git_version}"
  echo "export BUILD_VERSION=${git_version#v}" > build_version.env
  echo "${git_version}"
}

function current_tag() {
  git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

function next_version() {
  local base="$1"
  local major minor patch
  IFS=. read -r major minor patch <<<"${base#v}"

  # look at commits since last tag
  local commits=$(git log "${base}..HEAD" --pretty=format:%s)

  if grep -qE "^BREAKING CHANGE|!:" <<<"$commits"; then
    echo "v$((major+1)).0.0"
  elif grep -qE "^feat(\(|:)" <<<"$commits"; then
    echo "v${major}.$((minor+1)).0"
  elif grep -qE "^fix(\(|:)" <<<"$commits"; then
    echo "v${major}.${minor}.$((patch+1))"
  else
    # no conventional commits → keep same
    echo "${base}"
  fi
}

function create_and_push_tag() {
  local new="$1"
  log_message "Creating/updating tag ${new}"
  git tag -f -a "$new" -m "Release $new"
  git push origin "refs/tags/$new" --force
}

function write_build_version_env() {
  local tag="$1"
  local numeric="${tag#v}"
  printf 'export BUILD_VERSION=%s\n' "$numeric" > build_version.env
  log_message "Created build_version.env (BUILD_VERSION=${numeric})"
}

function update_git_version_tag() {
  if [[ "${BITBUCKET_BRANCH:-}" != "main" ]]; then
    log_warning "Not on main branch, skipping tag"
    local cur="$(current_tag)"
    write_build_version_env "$cur"
    return 0
  fi

  git fetch --tags

  local current=$(current_tag)
  local next=$(next_version "$current")

  if [[ "$next" != "$current" ]]; then
    log_message "Bumping version: $current → $next"
    create_and_push_tag "$next"
    write_build_version_env "$next"
  else
    log_message "No new version bump (latest: $current)"
    write_build_version_env "$current"
  fi
}

# Argument dispatcher
if [[ $# -eq 0 ]]; then
  log_message "usage: $0 {all|build|unit-test|integration-test|scan} [...]"
  exit 1
fi

for target in "$@"; do
  case "${target}" in
    setup-environment)
      common_preparation
      ;;
    all)
      common_preparation
      compile_binary
      unit_tests
      integration_tests
      code_scan
      ;;
    build)
      common_preparation
      generate_fix
      compile_binary
      copy_binaries
      ;;
    build-release)
      common_preparation
      generate_fix
      compile_binary darwin arm64
      compile_binary linux arm64
      compile_binary linux amd64
      compile_binary windows amd64
      build_release=true
      copy_binaries
      ;;
    upload)
      upload_artifacts
      ;;
    unit-test)
      common_preparation
      unit_tests
      ;;
    integration-test)
      common_preparation
      integration_tests
      ;;
    scan)
      code_scan
      ;;
    tag-version)
      update_git_version_tag
      ;;
    clean)
      clean_repo
      ;;
    *)
      log_message "Unknown target: ${target}"
      log_message "usage: $0 {all|clean|build|unit-test|integration-test|scan} [...]"
      exit 1
      ;;
  esac
done

if [[ ${compile_binary} == true ]]; then
  log_message ">> Artifacts Built:"
  
  tree -D ./bin
fi
