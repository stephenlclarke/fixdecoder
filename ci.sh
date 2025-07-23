#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# Unified CI helper for the fixdecoder project
#
#   scripts/ci.sh build             – compile binary (was build.sh → build)
#   scripts/ci.sh unit-test         – unit tests + coverage  (was build.sh → unit)
#   scripts/ci.sh integration-test  – integration tests      (was build.sh → integration)
#   scripts/ci.sh scan              – gitleaks secret scan   (was scan.sh)
#
# Every build-related target runs the common preparation steps that lived
# in build.sh:  setup_environment → install_dependencies → tidy → generate_fix
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

function log_message {
  echo -e "\n\033[1;32m$1\033[0m"
}

function setup_environment() {
  log_message ">> Setting up environment"
  export GOPATH=$(go env GOPATH)
  export PATH="$(go env GOPATH)/bin:${PATH}"
  go env -w GOPRIVATE=bitbucket.org/edgewater/${application_name}
}

function install_dependencies() {
  log_message ">> Installing test dependencies"
  go install github.com/jstemmer/go-junit-report/v2@latest
}

function tidy() {
  log_message ">> Running go mod tidy in all modules"
  go mod tidy
  go mod download
}

function generate_fix() {
  log_message ">> Auto-Generating FIX dictionary"
  chmod +x ./resources/generate_fix_go.sh
  ./resources/generate_fix_go.sh
}

function unit_tests() {
  if [[ ${unit_tests} == true ]]; then
    return
  fi

  log_message ">> Running unit tests"
  mkdir -p reports
  rm -f coverage.out

  for module in ${modules_list[@]}; do
      log_message " - Testing ${module}"
      abs_report_path=$(cd reports && pwd)/coverage-`basename ${module}`.out

      cd ${module}
      go test -v -covermode=atomic -coverprofile=${abs_report_path} .

      cd - >/dev/null
  done

  log_message ">> Merging coverage reports"
  echo "mode: atomic" > reports/coverage.out
  find . -name 'coverage-*.out' -exec tail -n +2 {} \; >> reports/coverage.out

  log_message ">> Generating JUnit test reports per module"
  mkdir -p reports

  local report_dir=$(cd reports && pwd)

  for module in ${modules_list[@]}; do
      report_name=$(echo ${module} | tr / -)
      printf " - Creating report for %s\n" ${report_name}

      local abs_report_path=$(cd reports && pwd)/test-report-$report_name.xml

      cd ${module}
      go test -json . | go-junit-report > ${abs_report_path}

      cd - >/dev/null
  done

  log_message ">> Generating unit test report"
  go test -json ./... | go-junit-report > reports/unit-test-report.xml

  log_message ">> Generating HTML coverage report"
  go tool cover -html=reports/coverage.out -o reports/coverage.html
  log_message "HTML report available at reports/coverage.html"

  unit_tests=true
}

function compile_binary() {
  # ensure the bin directory exists
  mkdir -p ./bin/fixdecoder

  # ensure your tags are fetched
  git fetch --tags

  local git_branch=$(git rev-parse --abbrev-ref HEAD)
  local git_short_sha=$(git rev-parse --short HEAD)
  local git_url=$(git remote get-url origin)
  local git_tag=$(git describe --tags --abbrev=0 2>/dev/null)
  local git_version=${git_tag:="v0.0.0"}

  [ $(git status --porcelain | wc -l) -ne "0" ] && git_version="${git_version}-dirty"

  local operating_system=${1:-$(go env GOOS)}
  local architecture=${2:-$(go env GOARCH)}
  local build_tag=""

  log_message ">> Building ${application_name} ${git_version} (branch: ${git_branch}, commit: ${git_short_sha}), OS: ${operating_system}, ARCH: ${architecture}"

  [[ -z "$1" || -z "$2" ]] || build_tag="-${git_version#v}.${operating_system}-${architecture}"
  
  mkdir -p ./bin/${application_name}/${git_version}

  # build with that version baked in
  env GOOS=${operating_system} GOARCH=${architecture} go build -ldflags="-X main.Version=${git_version} -X main.Branch=${git_branch} -X main.Sha=${git_short_sha} -X main.Url=${git_url}" -o ./bin/${application_name}/${git_version}/${application_name}${build_tag} ./cmd/${application_name}
}

function upload_artifacts() {
  if [[ ${build_release} == false ]]; then
    return
  fi

  local git_tag=$(git describe --tags --abbrev=0 2>/dev/null)
  local git_version=${git_tag:="v0.0.0"}

  [ $(git status --porcelain | wc -l) -ne "0" ] && git_version="${git_version}-dirty"
}

function integration_tests() {
  if [[ ${integration_tests} == true ]]; then
    return
  fi

  log_message ">> Running integration tests"

  # integration tests
  mkdir -p reports
  touch reports/coverage.integration.out
  go test -v -tags=integration -covermode=atomic -coverpkg=./... -coverprofile=reports/coverage.integration.out ./...
  go test -tags=integration -timeout=10m -run '^TestMain' ./...

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

  log_message ">> SonarQube Scan"
  docker run --rm -e SONAR_TOKEN="${SONAR_TOKEN}" -v "$(pwd):/usr/src" sonarsource/sonar-scanner-cli

  code_scan=true
}

# Helper that runs the common pre-build steps in order
function common_preparation() {
  if [[ ${common_preparation} == true ]]; then
    return
  fi

  setup_environment
  install_dependencies
  tidy
  generate_fix

  common_preparation=true
}

# Argument dispatcher
if [[ $# -eq 0 ]]; then
  log_message "usage: $0 {all|build|unit-test|integration-test|scan} [...]"
  exit 1
fi

for target in "$@"; do
  case "${target}" in
    all)
      common_preparation
      compile_binary
      unit_tests
      integration_tests
      code_scan
      ;;
    build)
      common_preparation
      compile_binary
      ;;
    build-release)
      common_preparation
      compile_binary darwin arm64
      compile_binary linux arm64
      compile_binary linux amd64
      compile_binary windows amd64
      build_release=true
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
    *)
      log_message "Unknown target: ${target}"
      log_message "usage: $0 {all|build|unit-test|integration-test|scan} [...]"
      exit 1
      ;;
  esac
done
