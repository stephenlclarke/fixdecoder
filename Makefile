# Makefile — delegates all real work to ./ci.sh
# --------------------------------------------

CI_SCRIPT := ./ci.sh

# The targets your pipeline (and developers) will call
.PHONY: setup-environment build unit-test integration-test scan security-scan clean upload tag-version help

# Straight-through wrappers: “make build” → “./ci.sh build”, etc.
build unit-test integration-test scan build-release upload clean setup-environment tag-version:
	$(CI_SCRIPT) $@

# Alias so `make security-scan` feels natural
security-scan:
	$(CI_SCRIPT) scan

# Simple help text
help:
	@echo "Available targets:"
	@echo "  build              → $(CI_SCRIPT) build"
	@echo "  setup-environment  → $(CI_SCRIPT) setup-environment"
	@echo "  build-release      → $(CI_SCRIPT) build-release"
	@echo "  upload             → $(CI_SCRIPT) upload"
	@echo "  unit-test          → $(CI_SCRIPT) unit-test"
	@echo "  integration-test   → $(CI_SCRIPT) integration-test"
	@echo "  scan               → $(CI_SCRIPT) scan"
	@echo "  clean              → $(CI_SCRIPT) clean"
	@echo "  tag-version        → $(CI_SCRIPT) tag-version"
	@echo "  help               → this help text"
