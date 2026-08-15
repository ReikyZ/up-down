import argparse
import json
import mimetypes
import secrets
import sys
from http.client import HTTPConnection, HTTPSConnection
from pathlib import Path
from urllib.parse import urlsplit

from updown.config import load_dotenv, string_env


def upload(server_url, file_path):
    target = urlsplit(server_url)
    if target.scheme not in ("http", "https") or not target.netloc:
        raise ValueError("SERVER_URL must be an absolute HTTP URL")
    path = Path(file_path)
    file_size = path.stat().st_size
    boundary = "----up-down-" + secrets.token_hex(16)
    prefix = _multipart_prefix(boundary, path.name)
    suffix = "\r\n--{0}--\r\n".format(boundary).encode("ascii")
    request_path = (target.path.rstrip("/") + "/upload") or "/upload"
    if target.query:
        request_path += "?" + target.query
    connection_class = HTTPSConnection if target.scheme == "https" else HTTPConnection
    connection = connection_class(target.hostname, target.port, timeout=60)
    try:
        connection.putrequest("POST", request_path)
        connection.putheader("Content-Type", "multipart/form-data; boundary=" + boundary)
        connection.putheader("Content-Length", str(len(prefix) + file_size + len(suffix)))
        connection.endheaders()
        connection.send(prefix)
        with path.open("rb") as source:
            while True:
                chunk = source.read(1024 * 1024)
                if not chunk:
                    break
                connection.send(chunk)
        connection.send(suffix)
        response = connection.getresponse()
        body = response.read()
    finally:
        connection.close()
    if response.status != 200:
        raise RuntimeError("upload failed ({0}): {1}".format(response.status, body.decode("utf-8", "replace").strip()))
    try:
        result = json.loads(body.decode("utf-8"))
        upload_path = result["url"]
    except (KeyError, ValueError, TypeError):
        raise RuntimeError("server returned an invalid upload response")
    if not isinstance(upload_path, str) or not upload_path.startswith("/"):
        raise RuntimeError("server returned an invalid upload URL")
    return "{0}://{1}{2}".format(target.scheme, target.netloc, upload_path)


def _multipart_prefix(boundary, filename):
    content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    return (
        "--{0}\r\n"
        "Content-Disposition: form-data; name=\"file\"; filename=\"{1}\"\r\n"
        "Content-Type: {2}\r\n\r\n"
    ).format(boundary, filename.replace('"', ""), content_type).encode("utf-8")


def main():
    parser = argparse.ArgumentParser(prog="up", description="Upload a file to an up-down server")
    parser.add_argument("file")
    arguments = parser.parse_args()
    load_dotenv(".env")
    try:
        print(upload(string_env("SERVER_URL", ""), arguments.file))
    except (OSError, RuntimeError, ValueError) as error:
        print("up: {0}".format(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
