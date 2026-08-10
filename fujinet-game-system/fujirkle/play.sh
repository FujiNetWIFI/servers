#!/usr/bin/env bash
#
# Starts the game server in the background and drops you straight into the
# client. Stops the server again when you quit.
#
#   ./play.sh                     play as $USER on table dev1 (1 bot)
#   ./play.sh -table dev3         3 bots
#   ./play.sh -player bob         pick a name
#   ./play.sh -auto -games 1      watch it play itself
#
# To use a server that is already running somewhere else, skip this script:
#   cd client && go run . -host http://somehost:8080 -player rich

set -uo pipefail

cd "$(dirname "$0")"

PORT="${PORT:-8080}"
HOST="http://localhost:${PORT}"
BUILD="$(mktemp -d)"

# The server chatters constantly (gin logs every request, and the client polls
# twice a second), so all of it goes to a file rather than over the game
# display. Kept outside $BUILD so it survives a crash.
SERVER_LOG="${TMPDIR:-/tmp}/fujirkle-server.log"

cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" 2>/dev/null
    wait "$SERVER_PID" 2>/dev/null
  fi
  rm -rf "$BUILD"
}
trap cleanup EXIT

echo "Building..."
(cd server && go build -o "$BUILD/server" .) || exit 1
(cd client && go build -o "$BUILD/client" .) || exit 1

echo "Starting server on port $PORT (log: $SERVER_LOG)..."
PORT="$PORT" "$BUILD/server" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "$HOST/tables" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "Server failed to start:"
    cat "$SERVER_LOG"
    exit 1
  fi
  sleep 0.5
done

if ! curl -fsS "$HOST/tables" >/dev/null 2>&1; then
  echo "Server never became ready. Log:"
  cat "$SERVER_LOG"
  exit 1
fi

# Default to a player name and table unless the caller supplied them
args=("$@")
case " $* " in
  *" -player "*) ;;
  *) args+=(-player "${USER:-player}") ;;
esac
case " $* " in
  *" -table "*) ;;
  *) args+=(-table dev1) ;;
esac

"$BUILD/client" -host "$HOST" "${args[@]}"
