#!/bin/bash

# Prefer local sonar-scanner if available
if command -v sonar-scanner >/dev/null 2>&1; then
  echo "Using local sonar-scanner"
  sonar-scanner -Dsonar.token="${SONAR_TOKEN}" "$@"
else
  echo "Local sonar-scanner not found, falling back to Docker image"
  docker run --rm \
    --platform=linux/amd64 \
    -e SONAR_TOKEN="${SONAR_TOKEN}" \
    -v "$(pwd):/usr/src" \
    sonarsource/sonar-scanner-cli "$@"
fi
