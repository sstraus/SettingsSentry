#!/usr/bin/env python3
"""Check real release archives; catch wrong ldflags and leaked command fixtures."""

import json
import platform
from pathlib import Path
import subprocess
import tempfile
import zipfile


def main():
    dist = Path(__file__).resolve().parents[1] / "dist"
    version = json.loads((dist / "metadata.json").read_text())["version"]
    native_arch = "arm64" if platform.machine() == "arm64" else "x86_64"
    for arch in ("arm64", "x86_64"):
        archive = dist / f"SettingsSentry_{arch}.zip"
        with zipfile.ZipFile(archive) as zipped:
            names = zipped.namelist()
            assert "settingssentry" in names, f"Missing binary in {archive}"
            assert any(name.startswith("configs/") for name in names), archive
            assert not any("TestCommand" in name for name in names), archive
            with tempfile.TemporaryDirectory(dir=dist) as temp:
                zipped.extractall(temp)
                binary = Path(temp) / "settingssentry"
                binary.chmod(0o755)
                info = subprocess.check_output(["file", str(binary)], text=True)
                assert arch in info, info
                if arch == native_arch:
                    help_text = subprocess.check_output([str(binary)], text=True)
                    assert f"SettingsSentry v{version}" in help_text, help_text
                    # configsinit reads the embedded filesystem, not archive files.
                    for config in (Path(temp) / "configs").glob("*.cfg"):
                        config.unlink()
                    subprocess.run([str(binary), "configsinit"], check=True, capture_output=True)
                    extracted = list((Path(temp) / "configs").glob("*.cfg"))
                    assert extracted, "No embedded configurations extracted"
                    assert not any("TestCommand" in path.name for path in extracted), extracted
        print(f"PASS: {archive.name}: architecture, configs, no command fixture")
    print(f"PASS: native binary reports artifact version {version}")


if __name__ == "__main__":
    main()
