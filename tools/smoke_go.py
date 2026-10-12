#!/usr/bin/env python3
"""Exercise the Go player in a POSIX PTY, optionally with muted real audio."""
import argparse
import errno
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import signal
import struct
import termios
import time

ROOT = Path(__file__).resolve().parent.parent


def run(binary, name, extra, resize=False, controls=False, start_time=147.66, end_time=150.66, working_directory=ROOT):
    report = ROOT / ".build" / f"smoke-{name}.json"
    report.unlink(missing_ok=True)
    command = [str(binary), "--autoplay", "--start", str(start_time), "--stop-after", str(end_time), "--report", str(report), *extra]
    pid, fd = pty.fork()
    if pid == 0:
        os.chdir(working_directory)
        os.environ["TERM"] = "xterm-256color"
        os.execv(str(binary), command)
    started = time.monotonic()
    terminal_started = None
    output = bytearray()
    exit_status = None
    paused = resumed = resized = sought = False
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 44, 128, 0, 0))
    try:
        while time.monotonic() - started < 30:
            # MP3 predecoding happens before terminal initialization. Start
            # interaction timing after output appears, not at process launch.
            elapsed = 0 if terminal_started is None else time.monotonic() - terminal_started
            if controls and elapsed >= 1 and not paused:
                os.write(fd, b" h")  # pause and open help
                paused = True
            if controls and elapsed >= 1.4 and not resumed:
                os.write(fd, b"h ")  # close help and resume
                resumed = True
            if resize and elapsed >= 1.5 and not resized:
                fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 90, 38, 0, 0))
                resized = True
            if controls and elapsed >= 2 and not sought:
                os.write(fd, b"\x1b[C")  # playing seek; crosses --stop-after
                sought = True
            ready, _, _ = select.select([fd], [], [], .02)
            if ready:
                try:
                    data = os.read(fd, 65536)
                except OSError as error:
                    if error.errno == errno.EIO:
                        break
                    raise
                if not data:
                    break
                if terminal_started is None:
                    terminal_started = time.monotonic()
                output.extend(data)
            waited, status = os.waitpid(pid, os.WNOHANG)
            if waited:
                exit_status = status
                break
        else:
            raise RuntimeError(f"{name}: timed out")
        if exit_status is None:
            _, exit_status = os.waitpid(pid, 0)
        code = os.waitstatus_to_exitcode(exit_status)
        if code != 0 or not report.exists():
            raise RuntimeError(f"{name}: exit {code}: {output[-3000:].decode(errors='replace')}")
        if b"\x1b[?1049l" not in output:
            raise RuntimeError(f"{name}: alternate screen was not restored")
        result = json.loads(report.read_text())
        if result["frames"] < 10 or result["last_time"] < end_time:
            raise RuntimeError(f"{name}: timeline did not advance: {result}")
        if resize and (result["Width"], result["Height"]) != (38, 90):
            raise RuntimeError(f"{name}: resize not applied: {result}")
        print(name, json.dumps(result), flush=True)
    finally:
        os.close(fd)
        if exit_status is None:
            try:
                os.kill(pid, signal.SIGKILL)
                os.waitpid(pid, 0)
            except ProcessLookupError:
                pass


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=ROOT / ".build/world-execute")
    parser.add_argument("--audio", type=Path, help="also test this MP3 on the real audio device, muted")
    args = parser.parse_args()
    binary = args.binary.resolve()
    (ROOT / ".build").mkdir(exist_ok=True)
    run(binary, "demo-60", ["--demo", "--fps", "60"])
    run(binary, "demo-120", ["--demo", "--fps", "120"])
    run(binary, "resize-controls", ["--demo"], resize=True, controls=True)
    if args.audio:
        run(binary, "audio-60", ["--audio", str(args.audio.resolve()), "--volume", "0", "--fps", "60"])
        run(binary, "audio-controls", ["--audio", str(args.audio.resolve()), "--volume", "0"], resize=True, controls=True)


if __name__ == "__main__":
    main()
