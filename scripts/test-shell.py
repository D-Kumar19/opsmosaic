#!/usr/bin/env python3
"""Exercise installed shell hooks in real isolated bash/zsh pseudo-terminals."""

import errno
import fcntl
import json
import os
import pty
from pathlib import Path
import select
import shutil
import subprocess
import struct
import tempfile
import termios
import time

from demo import capture


root = Path(__file__).resolve().parent.parent


def capture_topic_picker(command, environment, keys, expected_success=True, resize=None):
    master, slave = pty.openpty()
    process = None
    output = bytearray()
    name_sent = False
    keys_sent = False
    deadline = time.monotonic() + 15
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
        process = subprocess.Popen(command, stdin=slave, stdout=slave, stderr=slave, env=environment)
        while time.monotonic() < deadline:
            readable, _, _ = select.select([master], [], [], 0.1)
            if readable:
                try:
                    chunk = os.read(master, 65536)
                except OSError as error:
                    if error.errno != errno.EIO:
                        raise
                    break
                if not chunk:
                    break
                output.extend(chunk)
                if not name_sent and b"What should I call you?" in output:
                    os.write(master, b"PickerTest\n")
                    name_sent = True
                if name_sent and not keys_sent and b"Choose your interests, PickerTest" in output:
                    if resize is None:
                        os.write(master, keys)
                    else:
                        first, rows, remaining = resize
                        os.write(master, first)
                        time.sleep(0.05)
                        fcntl.ioctl(slave, termios.TIOCSWINSZ,
                                    struct.pack("HHHH", rows, 80, 0, 0))
                        os.write(master, remaining)
                    keys_sent = True
            elif process.poll() is not None:
                break
        process.wait(timeout=1)
        if (process.returncode == 0) != expected_success:
            raise RuntimeError(f"Interactive picker returned {process.returncode}:\n{output.decode(errors='replace')}")
        if not name_sent or not keys_sent:
            raise RuntimeError(f"Interactive picker missed its prompt:\n{output.decode(errors='replace')}")
        attributes = termios.tcgetattr(slave)
        if not attributes[3] & termios.ICANON or not attributes[3] & termios.ECHO:
            raise AssertionError("Interactive picker did not restore canonical input and echo")
        return output.decode("utf-8", errors="replace")
    finally:
        if process is not None and process.poll() is None:
            process.kill()
            process.wait()
        os.close(master)
        os.close(slave)


with tempfile.TemporaryDirectory(prefix="opsmosaic-shell-test-") as temporary:
    temporary = Path(temporary)
    binary = temporary / "opsmosaic"
    subprocess.run([os.environ.get("GO", "go"), "build", "-o", str(binary), "./cmd/opsmosaic"],
                   cwd=root, check=True)
    environment = {
        "PATH": str(temporary), "HOME": str(temporary), "TERM": "xterm-256color",
        "OPSMOSAIC_HOME": str(temporary / "state"), "TZ": "UTC",
    }
    for shell in ("bash", "zsh"):
        executable = shutil.which(shell)
        if executable is None:
            raise RuntimeError(f"Install {shell} to run the shell integration tests")
        rc = temporary / f"{shell} config"
        subprocess.run([str(binary), "shell", "install", shell, str(rc)], check=True, env=environment)
        arguments = [executable, "--norc" if shell == "bash" else "-f", "-i", "-c",
                     '. "$HOME/' + rc.name + '"']
        started = time.monotonic()
        output = capture(arguments, environment)
        elapsed = time.monotonic() - started
        if "What should I call you?" in output or "TIP |" in output:
            raise AssertionError("Missing-profile shell hook attempted onboarding or rendering")
        print(f"{shell}: missing-profile interactive startup {elapsed * 1000:.1f} ms (one local sample)")
    capture_topic_picker(
        [str(binary), "config"], environment, b"",
        resize=(b" ", 7, b"\x1b[B\x1b[B\x1b[B\x1b[B\x1b[B\x1b[B\x1b[B\x1b[B \r"),
    )
    profile_path = temporary / "state" / "config.json"
    profile = json.loads(profile_path.read_text())
    if profile["name"] != "PickerTest" or profile["topics"] != ["docker", "ai"]:
        raise AssertionError(f"Arrow/Space/Enter selection produced wrong profile: {profile}")
    previous_profile = profile_path.read_bytes()
    capture_topic_picker([str(binary), "config"], environment, b"q", expected_success=False)
    if profile_path.read_bytes() != previous_profile:
        raise AssertionError("Cancelling the picker changed the existing profile")
    profile_path.unlink()
    Path(str(profile_path) + ".lock").unlink(missing_ok=True)
    # Create example state via the actual CLI, never the user's profile.
    subprocess.run([str(binary), "--tips"], input="Alex\n1\n", text=True,
                   capture_output=True, env=environment, check=True)
    for shell in ("bash", "zsh"):
        executable = shutil.which(shell)
        rc = temporary / f"{shell} config"
        command = '. "$HOME/' + rc.name + '"'
        for interactive in (False, True):
            args = [executable, "--norc" if shell == "bash" else "-f"]
            if interactive:
                args.append("-i")
            output = capture(args + ["-c", command], environment)
            if output.count("TIP |") != (1 if interactive else 0):
                raise AssertionError(f"{shell}: incorrect interactive-only greeting")
        broken = temporary / "broken"
        broken.mkdir(exist_ok=True)
        fake = broken / "opsmosaic"
        fake.write_text("#!/bin/sh\nprintf 'explicit startup failure\\n' >&2\nexit 7\n")
        fake.chmod(0o700)
        try:
            capture([executable, "--norc" if shell == "bash" else "-f", "-i", "-c", command],
                    {**environment, "PATH": str(broken)})
        except RuntimeError as error:
            if "explicit startup failure" not in str(error):
                raise
        else:
            raise AssertionError("Shell startup failure was hidden")
        output = capture([executable, "--norc" if shell == "bash" else "-f", "-i", "-c", command],
                         {**environment, "PATH": "/nonexistent"})
        if "TIP |" in output:
            raise AssertionError("Missing executable was not skipped")
        subprocess.run([str(binary), "shell", "uninstall", shell, str(rc)],
                       check=True, env=environment)
        if rc.read_text():
            raise AssertionError("Uninstall left managed hook behind")
print("Real shell PTY tests passed; temporary profiles and shell files removed.")
