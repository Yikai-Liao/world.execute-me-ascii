#!/usr/bin/env python3
"""Build native Go players, optionally embedding a locally supplied MP3."""
import argparse
import hashlib
import os
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parent.parent
TARGETS = ("linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--audio", type=Path, help="MP3 to embed in each executable")
    parser.add_argument("--target", action="append", choices=TARGETS, help="default: all six targets")
    args = parser.parse_args()
    bundled = ROOT / "bundled/song.mp3"
    created_audio = False
    if args.audio:
        source = args.audio.expanduser().resolve()
        if not source.is_file():
            parser.error(f"audio file does not exist: {source}")
        if source != bundled.resolve():
            if bundled.exists():
                parser.error("bundled/song.mp3 already exists; pass that file or move it first")
            shutil.copyfile(source, bundled)
            created_audio = True
    try:
        output = ROOT / "dist/go"
        output.mkdir(parents=True, exist_ok=True)
        checksums = []
        for target in args.target or TARGETS:
            system, arch = target.split("/")
            name = f"world-execute-{system}-{arch}" + (".exe" if system == "windows" else "")
            path = output / name
            env = dict(os.environ, GOOS=system, GOARCH=arch, CGO_ENABLED="0")
            subprocess.run(["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(path), "./cmd/world-execute"], cwd=ROOT, env=env, check=True)
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            checksums.append(f"{digest}  {name}\n")
            print(f"Built {path.relative_to(ROOT)} ({path.stat().st_size:,} bytes)", flush=True)
        (output / "SHA256SUMS.txt").write_text("".join(checksums))
    finally:
        if created_audio:
            bundled.unlink()


if __name__ == "__main__":
    main()
