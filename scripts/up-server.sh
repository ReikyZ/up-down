#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RUN_DIR="$ROOT_DIR/run"
PID_FILE="$RUN_DIR/up-server.pid"
LOG_FILE="$RUN_DIR/up-server.log"

is_running() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

check_port() {
    (
        cd "$ROOT_DIR"
        python3 - ".env" <<'PY'
import socket
import sys

from updown.config import load_dotenv, string_env
from updown.server import _listen_address

load_dotenv(sys.argv[1])
address = string_env("LISTEN_ADDR", ":8080")
try:
    host, port = _listen_address(address)
    probe = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    probe.bind((host, port))
except (OSError, ValueError) as error:
    print("cannot start up-server: {0} is unavailable: {1}".format(address, error), file=sys.stderr)
    sys.exit(1)
finally:
    try:
        probe.close()
    except NameError:
        pass
PY
    )
}

start() {
    if is_running; then
        echo "up-server is already running (pid $(cat "$PID_FILE"))"
        return 0
    fi
    rm -f "$PID_FILE"
    mkdir -p "$RUN_DIR"
    check_port
    (
        cd "$ROOT_DIR"
        nohup python3 cmd/up-server.py >> "$LOG_FILE" 2>&1 &
        echo "$!" > "$PID_FILE"
    )
    sleep 1
    if is_running; then
        echo "up-server started (pid $(cat "$PID_FILE"))"
        return 0
    fi
    echo "up-server failed to start; see $LOG_FILE" >&2
    rm -f "$PID_FILE"
    return 1
}

stop() {
    if ! is_running; then
        echo "up-server is not running"
        rm -f "$PID_FILE"
        return 0
    fi
    pid=$(cat "$PID_FILE")
    kill "$pid"
    echo "up-server stopped (pid $pid)"
    rm -f "$PID_FILE"
}

status() {
    if is_running; then
        echo "up-server is running (pid $(cat "$PID_FILE"))"
        return 0
    fi
    echo "up-server is not running"
    return 1
}

case "${1:-start}" in
    start) start ;;
    stop) stop ;;
    restart) stop; start ;;
    status) status ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}" >&2
        exit 2
        ;;
esac
