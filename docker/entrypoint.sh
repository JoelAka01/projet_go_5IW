#!/bin/sh
set -e

./server &
SERVER_PID=$!

sleep 1

SERVER_ADDR="${API_URL:-http://127.0.0.1:8080}" ./ssh-server &
SSH_PID=$!

terminate() {
	kill -TERM "$SERVER_PID" "$SSH_PID" 2>/dev/null
}
trap terminate TERM INT

while true; do
	if ! kill -0 "$SERVER_PID" 2>/dev/null; then
		wait "$SERVER_PID"
		EXIT_CODE=$?
		break
	fi
	if ! kill -0 "$SSH_PID" 2>/dev/null; then
		wait "$SSH_PID"
		EXIT_CODE=$?
		break
	fi
	sleep 1
done

terminate
wait 2>/dev/null

exit "$EXIT_CODE"
