import http.client
import json
import tempfile
import threading
import unittest

from updown.server import UploadHandler
from updown.storage import UploadStorage
from http.server import ThreadingHTTPServer


class ServerTest(unittest.TestCase):
    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), UploadHandler)
        self.server.storage = UploadStorage(self.temp_dir.name, 0)
        self.server.max_upload_bytes = 1024
        self.server.logger = _Logger()
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown()
        self.thread.join()
        self.server.server_close()
        self.temp_dir.cleanup()

    def test_healthz(self):
        response, body = self._request("GET", "/healthz")
        self.assertEqual(204, response.status)
        self.assertEqual(b"", body)

    def test_upload_and_download(self):
        boundary = "test-boundary"
        payload = (
            "--test-boundary\r\n"
            "Content-Disposition: form-data; name=\"file\"; filename=\"example.txt\"\r\n"
            "Content-Type: text/plain\r\n\r\n"
            "file contents\r\n"
            "--test-boundary--\r\n"
        ).encode("utf-8")
        response, body = self._request("POST", "/upload", payload, {"Content-Type": "multipart/form-data; boundary=" + boundary})
        self.assertEqual(200, response.status)
        result = json.loads(body.decode("utf-8"))
        response, body = self._request("GET", result["url"])
        self.assertEqual(200, response.status)
        self.assertEqual(b"file contents", body)

    def test_upload_over_limit_is_rejected(self):
        payload = b"x" * 2048
        response, _ = self._request("POST", "/upload", payload, {"Content-Type": "multipart/form-data; boundary=test"})
        self.assertEqual(413, response.status)

    def _request(self, method, path, body=None, headers=None):
        connection = http.client.HTTPConnection("127.0.0.1", self.server.server_port)
        connection.request(method, path, body, headers or {})
        response = connection.getresponse()
        result = (response, response.read())
        connection.close()
        return result


class _Logger(object):
    def info(self, *args):
        pass


if __name__ == "__main__":
    unittest.main()
