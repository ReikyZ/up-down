import os
import unittest
from unittest.mock import patch

from updown import client


class ClientTest(unittest.TestCase):
    def test_server_url_is_required_without_configuration(self):
        with patch.dict(os.environ, {}, clear=True):
            with patch("updown.client.load_dotenv"):
                with patch("updown.client.upload") as upload:
                    with patch("builtins.print"):
                        with patch("sys.argv", ["up", "file.txt"]):
                            self.assertEqual(1, client.main())

        upload.assert_not_called()


if __name__ == "__main__":
    unittest.main()
