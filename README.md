# up-down

Small Python 3 file-upload service and companion command-line client. It uses only the Python standard library.

## Setup

Copy the example configuration and adjust the server listener or storage path when needed:

```sh
cp .env.example .env
```

`.env.example` defaults the client to `http://138.128.192.42:8080`.

## Run the server

```sh
python3 cmd/up-server.py
```

To manage it in the background:

```sh
chmod +x scripts/up-server.sh
./scripts/up-server.sh start
./scripts/up-server.sh status
./scripts/up-server.sh stop
```

The script writes its PID and logs under `run/`.
It checks whether `LISTEN_ADDR` is available before starting and reports the occupied address without creating a stale PID file.

The server accepts `POST /upload` multipart requests with a `file` field and exposes uploads at `GET /files/<filename>`. Upload bodies are streamed directly to disk, subject to `MAX_UPLOAD_BYTES`. Before and after each upload, it checks available filesystem capacity. When it is below `MIN_FREE_BYTES` (1 GiB by default), it deletes the oldest uploaded files until the threshold is restored. Upload writes and cleanup are serialized so concurrent uploads cannot race the retention process.

The server handles `SIGINT` and `SIGTERM` with a graceful shutdown. Use `GET /healthz` for a liveness probe; it returns `204 No Content`.

## Upload a file

```sh
python3 cmd/up.py ./example.png
```

For an installed command:

```sh
make install
up ./example.png
```

To choose the installed command name:

```sh
make install COMMAND=upload
upload ./example.png
```

The launcher is installed to `~/.local/bin` by default. Override it with `BIN_DIR=/your/bin`; the chosen directory must be in `PATH`.

The command prints the public URL returned by the server.
