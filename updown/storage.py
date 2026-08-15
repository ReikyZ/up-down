import os
import secrets
import shutil
import threading
from datetime import datetime
from pathlib import Path


class InsufficientStorageError(Exception):
    pass


class UploadStorage(object):
    def __init__(self, directory, min_free_bytes):
        self.directory = Path(directory)
        self.directory.mkdir(parents=True, exist_ok=True)
        self.min_free_bytes = min_free_bytes
        self._lock = threading.Lock()

    def save(self, source, original_name):
        """Store source while serializing retention decisions with other uploads."""
        with self._lock:
            self._ensure_free_space()
            name = self._new_filename(original_name)
            destination_path = self.directory / name
            try:
                with destination_path.open("xb") as destination:
                    shutil.copyfileobj(source, destination, length=1024 * 1024)
                self._ensure_free_space()
            except Exception:
                try:
                    destination_path.unlink()
                except FileNotFoundError:
                    pass
                raise
            return name

    def _ensure_free_space(self):
        while shutil.disk_usage(str(self.directory)).free < self.min_free_bytes:
            oldest = self._oldest_file()
            if oldest is None:
                raise InsufficientStorageError("no uploaded files can be deleted")
            oldest.unlink()

    def _oldest_file(self):
        files = [path for path in self.directory.iterdir() if path.is_file()]
        if not files:
            return None
        return min(files, key=lambda path: path.stat().st_mtime)

    @staticmethod
    def _new_filename(original_name):
        extension = Path(original_name).suffix
        if len(extension) > 32 or "/" in extension or "\\" in extension:
            extension = ""
        timestamp = datetime.utcnow().strftime("%Y%m%dT%H%M%S.%fZ")
        return "{0}-{1}{2}".format(timestamp, secrets.token_hex(12), extension)
