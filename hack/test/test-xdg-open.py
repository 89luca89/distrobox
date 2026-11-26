#!/usr/bin/env python3
"""Exercise the xdg-open relay without a desktop session or containers."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
ASSETS = ROOT / "internal/inside-distrobox/assets"


class XdgOpenTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.directory = Path(self.tmp.name)
        self.bin = self.directory / "bin"
        self.bin.mkdir()
        for command in ("basename", "id", "sort", "head", "tty", "uname", "grep", "cut"):
            (self.bin / command).symlink_to(shutil.which(command))
        self.helper = self.bin / "distrobox-host-exec"
        self.helper.symlink_to(ASSETS / "distrobox-host-exec")
        (self.bin / "xdg-open").symlink_to(self.helper)
        self.env = dict(os.environ, PATH=str(self.bin), container="1",
                        USER="test", HOME=str(self.directory), SHELL="/bin/sh",
                        PORTAL_LOG=str(self.directory / "portal"),
                        SPAWN_LOG=str(self.directory / "spawn"))
        self.stub("host-spawn", '''
if [ "$1" = --version ]; then
    echo v1.6.0
    exit 0
fi
printf '%s\\n' "$@" > "$SPAWN_LOG"
exit "${SPAWN_STATUS:-0}"
''')

    def stub(self, name, body):
        path = self.bin / name
        path.write_text("#!/bin/sh\n" + body)
        path.chmod(0o755)

    def portal(self, status=0):
        self.stub("gdbus", '''
printf '%s\\n' "$@" > "$PORTAL_LOG"
exit "${PORTAL_STATUS:-0}"
''')
        self.env["PORTAL_STATUS"] = str(status)

    def run_helper(self, *args, symlink=False):
        return subprocess.run([str(self.bin / ("xdg-open" if symlink else
                                               "distrobox-host-exec")), *args],
                              env=self.env, capture_output=True, text=True)

    def test_portal_for_explicit_and_symlink_invocations(self):
        self.portal()
        url = 'https://example.org/a?q="hello world"&next=%27'
        for symlink in (False, True):
            with self.subTest(symlink=symlink):
                result = self.run_helper(*([url] if symlink else ["xdg-open", url]),
                                         symlink=symlink)
                self.assertEqual(result.returncode, 0, result.stderr)
                arguments = (self.directory / "portal").read_text().splitlines()
                self.assertEqual(arguments[-3:], ["", url, "{}"])
                self.assertIn("org.freedesktop.portal.OpenURI.OpenURI", arguments)
                self.assertFalse((self.directory / "spawn").exists())

    def test_unavailable_portal_falls_back(self):
        self.portal(status=1)
        result = self.run_helper("xdg-open", "https://example.org")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.directory / "spawn").read_text().splitlines(),
                         ["--no-pty", "xdg-open", "https://example.org"])

    def test_missing_gdbus_falls_back_and_preserves_exit_status(self):
        self.env["SPAWN_STATUS"] = "7"
        result = self.run_helper("xdg-open", "https://example.org")
        self.assertEqual(result.returncode, 7, result.stderr)
        self.assertTrue((self.directory / "spawn").exists())

    def test_local_files_and_other_commands_bypass_portal(self):
        self.portal()
        for arguments in (("xdg-open", "/tmp/a:b"),
                          ("xdg-open", "./a:b"),
                          ("xdg-open", "file:///tmp/a"),
                          ("xdg-open", "FILE:///tmp/a"),
                          ("xdg-open", "relative file.txt"),
                          ("xdg-open", "--help"),
                          ("printf", "%s", "https://example.org")):
            with self.subTest(arguments=arguments):
                result = self.run_helper(*arguments)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertFalse((self.directory / "portal").exists())
                self.assertEqual((self.directory / "spawn").read_text().splitlines(),
                                 ["--no-pty", *arguments])

    def test_init_relay_preserves_local_overrides(self):
        source = (ASSETS / "distrobox-init").read_text()
        block = source[source.index("# Prefer opening links on the host"):
                       source.index("# If flatpak is not present")]
        local_bin = self.directory / "local"
        block = block.replace("/usr/local/bin", str(local_bin))
        block = block.replace("/usr/bin/distrobox-host-exec", str(self.helper))
        local_bin.mkdir()
        relay = local_bin / "xdg-open"
        for init in (1, 0):
            subprocess.run(["/bin/sh", "-ec", f"init={init}\n" + block], check=True)
            self.assertEqual(relay.is_symlink(), init == 0)
        relay.unlink()
        relay.write_text("custom opener")
        subprocess.run(["/bin/sh", "-ec", "init=0\n" + block], check=True)
        self.assertEqual(relay.read_text(), "custom opener")


if __name__ == "__main__":
    unittest.main()
