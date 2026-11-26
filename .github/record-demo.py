#!/usr/bin/env python3
"""Record a demo binary in a pseudo terminal as an asciicast v2 file.

Usage: record-demo.py OUTPUT COMMAND [ARGS...]
"""

import codecs
import fcntl
import json
import os
import pty
import shlex
import struct
import sys
import termios
import time

COLUMNS = 90
ROWS = 20
END_PAUSE = 3.0


def main() -> None:
    if len(sys.argv) < 3:
        sys.exit(__doc__.strip())

    output, command = sys.argv[1], sys.argv[2:]
    prompt = "$ " + shlex.join([os.path.basename(command[0]), *command[1:]])
    env = {**os.environ, "TERM": "xterm-256color"}
    env.pop("COLORFGBG", None)
    env.pop("NO_COLOR", None)

    pid, fd = pty.fork()
    if pid == 0:
        try:
            # Set the size before exec, so the demo never sees an unsized terminal.
            fcntl.ioctl(0, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLUMNS, 0, 0))
            os.execvpe(command[0], command, env)
        except OSError as err:
            os.write(2, f"{command[0]}: {err}\n".encode())
            os._exit(127)

    decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")

    start = time.monotonic()
    events = [[0.0, "o", prompt + "\r\n"]]

    while True:
        try:
            data = os.read(fd, 4096)
        except OSError:
            break

        if not data:
            break

        text = decoder.decode(data)
        if text:
            events.append([round(time.monotonic() - start, 6), "o", text])

    text = decoder.decode(b"", final=True)
    if text:
        events.append([round(time.monotonic() - start, 6), "o", text])

    _, status = os.waitpid(pid, 0)
    exit_code = os.waitstatus_to_exitcode(status)
    if exit_code != 0:
        sys.exit(f"{command[0]} failed with exit code {exit_code}")

    # Keep the final frame visible for a moment before the animation loops.
    events.append([round(events[-1][0] + END_PAUSE, 6), "o", ""])

    with open(output, "w", encoding="utf-8") as cast:
        header = {"version": 2, "width": COLUMNS, "height": ROWS, "env": {"TERM": env["TERM"]}}
        cast.write(json.dumps(header) + "\n")

        for event in events:
            cast.write(json.dumps(event) + "\n")


if __name__ == "__main__":
    main()
