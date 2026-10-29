import json
from pathlib import Path
import tempfile
import unittest

from prepare_grounded_dialogue import prepare


class BundleIntegrityTest(unittest.TestCase):
    bundle = Path(__file__).resolve().parents[1] / "models/grounded-dialogue-v1"

    def test_unpack_and_refuse_existing_state(self):
        with tempfile.TemporaryDirectory() as root:
            state = Path(root) / "state"
            prepare(self.bundle, state)
            checkpoint = (state / "language-brain.json").read_bytes()
            self.assertEqual(json.loads(checkpoint)["sequence"], 126)
            with self.assertRaises(FileExistsError):
                prepare(self.bundle, state)
            self.assertEqual((state / "language-brain.json").read_bytes(), checkpoint)

    def test_corrupted_asset_is_refused_before_creating_state(self):
        with tempfile.TemporaryDirectory() as root:
            root = Path(root)
            bad = root / "bundle"
            bad.mkdir()
            (bad / "manifest.json").write_bytes((self.bundle / "manifest.json").read_bytes())
            manifest = json.loads((bad / "manifest.json").read_text())
            first = manifest["files"][0]["source"]
            data = bytearray((self.bundle / first).read_bytes())
            data[len(data) // 2] ^= 1
            (bad / first).write_bytes(data)
            state = root / "state"
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                prepare(bad, state)
            self.assertFalse(state.exists())


if __name__ == "__main__":
    unittest.main()
