#!/usr/bin/env bash
set -euo pipefail

cmd="${1:-help}"

usage() {
  cat <<'EOF'
Usage: ./run.sh <command> [args]

Commands:
  build               Build local binary: ./forgecrowdbook
  test                Run all tests
  start               Start app via go run .
  mirror [remote]     Push all branches and tags to remote (default: github)
  help                Show this help
EOF
}

build() {
  go build -o forgecrowdbook .
  echo "Built ./forgecrowdbook"
}

test_all() {
  go test ./...
}

start() {
  go run .
}

mirror() {
  local remote="${1:-github}"
  if ! git remote get-url "$remote" >/dev/null 2>&1; then
    echo "Remote '$remote' not found."
    echo "Add one, for example:"
    echo "  git remote add $remote git@github.com:<user>/<repo>.git"
    exit 1
  fi

  git push "$remote" --all
  git push "$remote" --tags
  echo "Mirrored to remote '$remote'."
}

case "$cmd" in
  build)
    build
    ;;
  test)
    test_all
    ;;
  start)
    start
    ;;
  mirror)
    mirror "${2:-}"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "Unknown command: $cmd"
    usage
    exit 2
    ;;
esac
