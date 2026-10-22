#!/usr/bin/env python3
"""Record an isolated, network-free CLI demo and render it with macOS tools."""

import argparse
import errno
import fcntl
import os
from pathlib import Path
import pty
import select
import struct
import subprocess
import sys
import tempfile
import termios
import time


def capture(command, environment, replies=(), timeout=15, line_input=False):
    master, slave = pty.openpty()
    process = None
    output = bytearray()
    pending = list(replies)
    deadline = time.monotonic() + timeout
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 100, 0, 0))
        process = subprocess.Popen(command, stdin=subprocess.PIPE if line_input else slave,
                                   stdout=slave, stderr=slave, env=environment)
        os.close(slave)
        slave = -1
        while True:
            if time.monotonic() >= deadline:
                raise TimeoutError(f"Demo command exceeded {timeout}s: {command}")
            readable, _, _ = select.select([master], [], [], 0.1)
            if readable:
                try:
                    chunk = os.read(master, 65536)
                except OSError as error:
                    if error.errno != errno.EIO:
                        raise
                    break  # A closed PTY reports EIO on Linux.
                if not chunk:
                    break
                output.extend(chunk)
                if pending and pending[0][0].encode() in output:
                    _, answer = pending.pop(0)
                    if line_input:
                        process.stdin.write((answer + "\n").encode())
                        process.stdin.flush()
                    else:
                        os.write(master, (answer + "\n").encode())
            if line_input and not pending and process.stdin is not None and not process.stdin.closed:
                process.stdin.close()
            elif process.poll() is not None:
                break
        process.wait(timeout=1)
        text = output.decode("utf-8").replace("\r\n", "\n")
        if process.returncode:
            raise RuntimeError(f"Demo command failed ({process.returncode}):\n{text}")
        if pending:
            raise RuntimeError(f"Demo never reached prompt: {pending[0][0]}")
        return text.strip()
    finally:
        if process is not None and process.poll() is None:
            process.kill()
            process.wait()
        if process is not None and process.stdin is not None and not process.stdin.closed:
            process.stdin.close()
        os.close(master)
        if slave != -1:
            os.close(slave)


def validate_briefing(text):
    if text.count("TIP |") != 1:
        raise ValueError("Demo must contain exactly one tip per briefing")
    if "Alex" not in text or "UTC" not in text:
        raise ValueError("Demo is missing its example name or timezone")
    if "\x1b" in text:
        raise ValueError("Demo recording unexpectedly contains terminal escapes")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path("docs/demo"))
    args = parser.parse_args()
    if sys.platform != "darwin":
        parser.error("GIF rendering requires macOS AppKit and Swift; the app itself does not")
    root = Path(__file__).resolve().parent.parent
    destination = args.output.resolve()
    destination.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="opsmosaic-demo-") as temporary:
        temporary = Path(temporary)
        binary = temporary / "opsmosaic"
        subprocess.run([os.environ.get("GO", "go"), "build", "-o", str(binary), "./cmd/opsmosaic"],
                       cwd=root, check=True)
        # Do not inherit personal shell settings or use the real profile/cache.
        environment = {
            "PATH": os.defpath, "HOME": str(temporary),
            "OPSMOSAIC_HOME": str(temporary / "state"), "TERM": "xterm-256color",
            "TZ": "UTC", "LANG": "en_US.UTF-8",
        }
        first = capture([str(binary), "--tips", "--color=never"], environment, [
            ("What should I call you?", "Alex"),
            ("Topic numbers (comma-separated, e.g. 1,2,3):", "1,2,3,4"),
        ], line_input=True)
        offline = capture([str(binary), "--offline", "--color=never"], environment)
        for text in (first, offline):
            validate_briefing(text)
        if "What should I call you?" in offline:
            raise ValueError("Offline return unexpectedly repeated onboarding")
        prompt = first.split("Alex", 1)[0]
        scenes = [
            ("First run / example profile", "$ opsmosaic --tips --color=never\n\n" + prompt + "Alex"),
            ("Choose interests / one random tip", "$ opsmosaic --tips --color=never\n\n" + first),
            ("Return offline / no onboarding or network",
             "$ opsmosaic --offline --color=never\n\n" + offline),
        ]
        transcript = [
            "OpsMosaic terminal demo",
            "Actual CLI output, captured with a disposable Alex profile in a 100-column PTY.",
            "UTC is used for the demo; normal runs use your device timezone.",
            "Both commands avoid networking. Offline uses a bundled tip here; no news cache is seeded.",
            "Dates, selected tips and greetings vary between recordings.",
            "",
        ]
        for index, (title, text) in enumerate(scenes):
            scene = temporary / f"scene-{index}.txt"
            scene.write_text(text, encoding="utf-8")
            transcript.extend([f"=== {title} ===", text, ""])
        subprocess.run(["swift", str(root / "scripts/render-demo.swift"),
                        str(temporary), str(destination / "terminal.gif")], check=True)
        (destination / "transcript.txt").write_text("\n".join(transcript), encoding="utf-8")
    print(f"Demo and transcript saved in {destination}")


if __name__ == "__main__":
    main()
