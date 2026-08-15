import os
import shutil
import tempfile
import threading
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
            name = self._safe_filename(original_name)
            destination_path = self.directory / name
            temporary_path = None
            try:
                with tempfile.NamedTemporaryFile(
                    mode="xb", dir=str(self.directory), prefix=".upload-", delete=False
                ) as destination:
                    temporary_path = Path(destination.name)
                    shutil.copyfileobj(source, destination, length=1024 * 1024)
                self._ensure_free_space(protected_path=temporary_path)
                temporary_path.replace(destination_path)
            except Exception:
                try:
                    if temporary_path is not None:
                        temporary_path.unlink()
                except FileNotFoundError:
                    pass
                raise
            return name

    def _ensure_free_space(self, protected_path=None):
        while shutil.disk_usage(str(self.directory)).free < self.min_free_bytes:
            oldest = self._oldest_file(protected_path)
            if oldest is None:
                raise InsufficientStorageError("no uploaded files can be deleted")
            oldest.unlink()

    def _oldest_file(self, protected_path=None):
        files = [
            path
            for path in self.directory.iterdir()
            if path.is_file() and path != protected_path
        ]
        if not files:
            return None
        return min(files, key=lambda path: path.stat().st_mtime)

    @staticmethod
    def _safe_filename(original_name):
        name = Path(original_name).name
        if name in ("", ".", "..") or name != original_name.replace("\\", "/").rsplit("/", 1)[-1]:
            raise ValueError("invalid upload filename")
        return name
