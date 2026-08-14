# up-down

Small file-upload service and companion command-line client.

## Setup

Copy the example configuration and adjust the server's listener or storage path when needed:

```sh
cp .env.example .env
```

`.env` defaults the client to `http://138.128.192.42:8080`.

## Run the server

```sh
go run ./cmd/up-server
```

The server accepts `POST /upload` multipart requests with a `file` field and exposes uploads at `GET /files/<filename>`. Before and after each upload, it checks available filesystem capacity. When it is below `MIN_FREE_BYTES` (1 GiB by default), it deletes the oldest uploaded files until the threshold is restored.

## Upload a file

```sh
go run ./cmd/up ./example.png
```

For an installed command:

```sh
go install ./cmd/up
up ./example.png
```

The command prints the public URL returned by the server.
