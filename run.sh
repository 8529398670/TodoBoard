#!/usr/bin/env bash
# Rebuild for this machine and run it, opening the app in the browser once it
# answers. The development loop: edit, ./run.sh, look. Ctrl+C stops it.
#
# A copy of this script's binary still running from an earlier ./run.sh is
# stopped first, so rerunning it always serves the code as it is now. For
# release binaries use ./build.sh; for the container, ./dockerRun.sh.
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"

# Same reasoning as build.sh: config.yaml is read through the app itself, and
# assigned before it is eval'd so a config error stops the script.
BUILD_CONFIG="$( go run . config shell build )"
eval "$BUILD_CONFIG"
SERVER_CONFIG="$( go run . config shell server )"
eval "$SERVER_CONFIG"

BINARY="${build_output_dir}/${build_binary_name}"

# The version is when it was built -- 2026.10.23-1430 -- and nothing from
# git, so it reads the same whatever state the checkout is in.
VERSION="$( date +%Y.%m.%d-%H%M )"
BUILD_DATE="$( date -u +%Y-%m-%dT%H:%M:%SZ )"

mkdir -p "$build_output_dir"
echo "Building ${BINARY} ${VERSION}"
go build -ldflags="-X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}" -o "$BINARY" .

# Matched on the full path, so only this checkout's dev binary is stopped.
BINARY_PATH="$( pwd )/${BINARY}"
if pkill -f "^${BINARY_PATH}\$" 2>/dev/null; then
  echo "Stopped the previous run"
  sleep 1
fi

# A wildcard listen address is not something a browser can open.
case "$server_host" in
  ""|0.0.0.0|::) HOST="127.0.0.1" ;;
  *:*)           HOST="[${server_host}]" ;;
  *)             HOST="$server_host" ;;
esac
URL="http://${HOST}:${server_port}/"

# Anything still answering here is not ours (ours was stopped above), and the
# browser would open it instead of this build.
if curl -fs -o /dev/null --max-time 2 "${URL}api/health"; then
  echo "Something else is already serving ${URL}:" >&2
  lsof -nP -iTCP:"$server_port" -sTCP:LISTEN >&2 || true
  echo "Stop it first, or change server.port in config.yaml." >&2
  exit 1
fi

# Wait in the background for the server to answer, then open it. Gives up
# after 30 seconds rather than polling forever if the server failed to start.
(
  for _ in $( seq 60 ); do
    if curl -fs -o /dev/null "${URL}api/health"; then
      echo "Opening ${URL}"
      if command -v open >/dev/null 2>&1; then
        open "$URL"
      elif command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$URL" >/dev/null 2>&1
      fi
      exit 0
    fi
    sleep 0.5
  done
) &

exec "$BINARY_PATH"
