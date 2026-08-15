import argparse
import json
import signal
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import unquote

from updown.config import int_env, load_dotenv, string_env
from updown.storage import InsufficientStorageError, UploadStorage


class UploadTooLargeError(Exception):
    pass


class InvalidUploadError(Exception):
    pass


class UploadHandler(BaseHTTPRequestHandler):
    server_version = "up-down"

    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(204)
            self.end_headers()
            return
        if self.path.startswith("/files/"):
            self._serve_file(unquote(self.path[len("/files/"):]))
            return
        self.send_error(404)

    def do_POST(self):
        if self.path != "/upload":
            self.send_error(404)
            return
        try:
            part = self._read_file_part()
            name = self.server.storage.save(part, part.filename)
        except UploadTooLargeError:
            self._send_text(413, "file exceeds upload limit")
            return
        except InvalidUploadError as error:
            self._send_text(400, str(error))
            return
        except InsufficientStorageError as error:
            self._send_text(507, "storage is full: {0}".format(error))
            return
        except OSError:
            self._send_text(500, "could not save upload")
            return

        body = json.dumps({"filename": name, "url": "/files/" + name}).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_file_part(self):
        content_type = self.headers.get("Content-Type", "")
        boundary = _multipart_boundary(content_type)
        content_length = self.headers.get("Content-Length")
        if boundary is None or content_length is None:
            raise InvalidUploadError("multipart field 'file' is required")
        try:
            remaining = int(content_length)
        except ValueError:
            raise InvalidUploadError("invalid Content-Length")
        if remaining < 0:
            raise InvalidUploadError("invalid Content-Length")
        if remaining > self.server.max_upload_bytes:
            raise UploadTooLargeError()
        reader = _LimitedReader(self.rfile, remaining)
        first_boundary = reader.readline()
        if first_boundary != b"--" + boundary + b"\r\n":
            raise InvalidUploadError("invalid multipart upload")
        headers = _read_part_headers(reader)
        filename = _file_name(headers.get("content-disposition", ""))
        if filename is None:
            raise InvalidUploadError("multipart field 'file' is required")
        return _MultipartFile(reader, boundary, filename)

    def _serve_file(self, name):
        if not name or "/" in name or "\\" in name or name in (".", ".."):
            self.send_error(404)
            return
        path = self.server.storage.directory / name
        try:
            size = path.stat().st_size
            with path.open("rb") as source:
                self.send_response(200)
                self.send_header("Content-Type", "application/octet-stream")
                self.send_header("Content-Length", str(size))
                self.end_headers()
                while True:
                    chunk = source.read(1024 * 1024)
                    if not chunk:
                        break
                    self.wfile.write(chunk)
        except FileNotFoundError:
            self.send_error(404)

    def _send_text(self, status, message):
        body = (message + "\n").encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format_string, *args):
        self.server.logger.info("%s - %s", self.address_string(), format_string % args)


class _LimitedReader(object):
    def __init__(self, source, remaining):
        self.source = source
        self.remaining = remaining

    def read(self, size=-1):
        if size < 0 or size > self.remaining:
            size = self.remaining
        data = self.source.read(size)
        self.remaining -= len(data)
        return data

    def readline(self):
        if self.remaining == 0:
            return b""
        line = self.source.readline(self.remaining)
        self.remaining -= len(line)
        return line


class _MultipartFile(object):
    def __init__(self, reader, boundary, filename):
        self.reader = reader
        self.filename = filename
        self.marker = b"\r\n--" + boundary
        self.buffer = b""
        self.finished = False

    def read(self, size=-1):
        if self.finished:
            return b""
        if size < 0:
            size = 1024 * 1024
        while len(self.buffer) < size + len(self.marker) and self.reader.remaining:
            self.buffer += self.reader.read(min(1024 * 1024, self.reader.remaining))
            index = self.buffer.find(self.marker)
            if index >= 0:
                data = self.buffer[:index]
                suffix = self.buffer[index + len(self.marker):]
                if not (suffix.startswith(b"--") or suffix.startswith(b"\r\n")):
                    raise InvalidUploadError("invalid multipart upload")
                self.buffer = data
                self.finished = True
                break
        if not self.finished and not self.reader.remaining:
            raise InvalidUploadError("invalid multipart upload")
        result, self.buffer = self.buffer[:size], self.buffer[size:]
        return result


def _multipart_boundary(content_type):
    pieces = [piece.strip() for piece in content_type.split(";")]
    if not pieces or pieces[0].lower() != "multipart/form-data":
        return None
    for piece in pieces[1:]:
        if piece.lower().startswith("boundary="):
            value = piece.split("=", 1)[1].strip().strip('"')
            return value.encode("ascii") if value else None
    return None


def _read_part_headers(reader):
    headers = {}
    while True:
        line = reader.readline()
        if line == b"\r\n":
            return headers
        if not line or b":" not in line:
            raise InvalidUploadError("invalid multipart upload")
        key, value = line.decode("utf-8", "replace").split(":", 1)
        headers[key.lower()] = value.strip()


def _file_name(content_disposition):
    parts = [part.strip() for part in content_disposition.split(";")]
    if not parts or parts[0].lower() != "form-data":
        return None
    values = {}
    for part in parts[1:]:
        if "=" in part:
            key, value = part.split("=", 1)
            values[key.lower()] = value.strip().strip('"')
    if values.get("name") != "file" or not values.get("filename"):
        return None
    return values["filename"]


def main():
    import logging

    load_dotenv(".env")
    parser = argparse.ArgumentParser(description="up-down upload server")
    parser.add_argument("--listen", default=string_env("LISTEN_ADDR", ":8080"))
    arguments = parser.parse_args()
    try:
        min_free = int_env("MIN_FREE_BYTES", 1 << 30)
        max_upload = int_env("MAX_UPLOAD_BYTES", 10 << 30)
        if min_free < 0 or max_upload <= 0:
            raise ValueError("storage limits must be positive")
    except ValueError as error:
        parser.error(str(error))
    host, port = _listen_address(arguments.listen)
    http_server = ThreadingHTTPServer((host, port), UploadHandler)
    http_server.storage = UploadStorage(string_env("UPLOAD_DIR", "./uploads"), min_free)
    http_server.max_upload_bytes = max_upload
    http_server.logger = logging.getLogger("up-down")
    print("upload server listening on {0}".format(arguments.listen))

    def stop_server(signum, frame):
        http_server.shutdown()
    signal.signal(signal.SIGINT, stop_server)
    signal.signal(signal.SIGTERM, stop_server)
    try:
        http_server.serve_forever()
    finally:
        http_server.server_close()


def _listen_address(address):
    if address.startswith(":"):
        return "", int(address[1:])
    host, separator, port = address.rpartition(":")
    if not separator or not host:
        raise ValueError("LISTEN_ADDR must be host:port or :port")
    return host, int(port)


if __name__ == "__main__":
    main()
