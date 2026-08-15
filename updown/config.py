import os


def load_dotenv(path):
    """Load simple KEY=VALUE entries without overwriting the environment."""
    try:
        with open(path, "r") as dotenv:
            for raw_line in dotenv:
                line = raw_line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                key, value = line.split("=", 1)
                key = key.strip()
                value = value.strip().strip("\"'")
                if key and key not in os.environ:
                    os.environ[key] = value
    except FileNotFoundError:
        pass


def string_env(key, fallback):
    return os.environ.get(key) or fallback


def int_env(key, fallback):
    value = os.environ.get(key)
    if not value:
        return fallback
    try:
        return int(value)
    except ValueError:
        raise ValueError("{0} must be an integer".format(key))
