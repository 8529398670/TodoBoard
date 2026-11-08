#!/usr/bin/env bash
# Print a login link -- by default for user 1, the owner -- for whichever
# deployment of this checkout is actually live. It works once, for
# auth.login_token_ttl_seconds, unless --uses or --expires say otherwise.
#
#   scripts/login.sh               the owner (user 1)
#   scripts/login.sh 3             user 3
#   scripts/login.sh 3 --uses 5    a link that works up to 5 times
#   scripts/login.sh --uses unlimited --expires 7d
#                                  any number of times, for 7 days
#                                  (--expires takes 30m, 12h, 7d or never)
#   scripts/login.sh --docker      the docker container, even if a local
#                                  server is running too
#   scripts/login.sh --local       this checkout's local database
#
# Where the link comes from:
#   - the local server (scripts/run.sh) when it is running: the link goes
#     through its control socket;
#   - otherwise the docker container (docker/run.sh) when it is running:
#     the link goes through `docker exec`;
#   - otherwise the local database directly, offline. The link waits there
#     until the server is started.
# The two keep separate databases, so a link from one does nothing in the
# other. This says which one it used.
#
# The URL in the link is server.public_url when set, otherwise the local
# address (see `scripts/run.sh config shell server`).
set -euo pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )/.."


USER_ID=1
MODE=auto
LINK_FLAGS=()
while [ $# -gt 0 ]; do
  argument="$1"
  shift
  case "$argument" in
    --docker) MODE=docker ;;
    --local)  MODE=local ;;
    --uses|--expires)
      if [ $# -eq 0 ]; then
        echo "ERROR: ${argument} needs a value." >&2
        exit 1
      fi
      LINK_FLAGS+=( "-${argument#--}" "$1" )
      shift ;;
    --uses=*|--expires=*)
      name="${argument%%=*}"
      LINK_FLAGS+=( "-${name#--}" "${argument#*=}" ) ;;
    -h|--help) sed -n '2,28p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *[!0-9]*|"")
      echo "ERROR: expected a numeric user id, --uses, --expires, --docker or --local; got '${argument}'." >&2
      exit 1 ;;
    *) USER_ID="$argument" ;;
  esac
done

source scripts/lib.sh
build_binary
read_server_config
read_docker_config

if [ "$config_exists" != "true" ]; then
  echo "ERROR: no config yet -- ${config_file} does not exist." >&2
  echo "       Run scripts/run.sh first: it writes one, and the first start prints" >&2
  echo "       the owner's login link." >&2
  exit 1
fi

LOCAL_PID="$( our_server_pid )"
DOCKER_UP=false
container_running "$docker_container" && DOCKER_UP=true

if [ "$MODE" = "auto" ]; then
  if [ -n "$LOCAL_PID" ]; then
    MODE=local
    if [ "$DOCKER_UP" = "true" ]; then
      echo "Note: the ${docker_container} container is running too. It has its own database;"
      echo "      this link is for the local server. Use --docker for the container."
      echo ""
    fi
  elif [ "$DOCKER_UP" = "true" ]; then
    MODE=docker
  else
    MODE=local
  fi
fi

if [ "$MODE" = "docker" ]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "ERROR: docker is not on PATH." >&2
    exit 1
  fi
  if [ "$DOCKER_UP" != "true" ]; then
    STATE="$( docker ps -a --filter "name=^${docker_container}\$" --format '{{.Status}}' 2>/dev/null || true )"
    echo "ERROR: the ${docker_container} container is not running${STATE:+ (docker: ${STATE})}." >&2
    echo "       Start it with docker/run.sh${STATE:+, or see why it stopped: docker logs ${docker_container}}." >&2
    exit 1
  fi
  echo "Docker container ${docker_container}, served at ${docker_url}"
  case "$docker_data_dir" in
    /*) DATA_DIR="$docker_data_dir" ;;
    *)  DATA_DIR="$( pwd )/${docker_data_dir#./}" ;;
  esac
  echo "  database ${DATA_DIR}/app.db on the host (docker.data_dir)"
  echo ""
  if ! docker exec "$docker_container" /app/server manage reissue-login -user-id "$USER_ID" ${LINK_FLAGS[@]+"${LINK_FLAGS[@]}"}; then
    echo "" >&2
    echo "Accounts in the container: docker exec ${docker_container} /app/server manage list-users" >&2
    exit 1
  fi
  exit 0
fi

if [ ! -f "$database" ]; then
  echo "ERROR: there is no database yet at ${database}." >&2
  echo "       Start the server with scripts/run.sh -- its first start creates the owner" >&2
  echo "       and prints their login link." >&2
  exit 1
fi

if [ -n "$LOCAL_PID" ]; then
  echo "Local server (pid ${LOCAL_PID}), served at ${url}"
else
  echo "No server is running for this checkout, so the link goes straight into"
  echo "the database. It works once you start it: scripts/run.sh"
fi
echo "  database ${database}"
echo ""
if ! "$BINARY" manage reissue-login -user-id "$USER_ID" ${LINK_FLAGS[@]+"${LINK_FLAGS[@]}"}; then
  echo "" >&2
  echo "Accounts in this database: scripts/run.sh manage list-users" >&2
  exit 1
fi
