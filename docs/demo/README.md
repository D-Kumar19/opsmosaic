# Terminal demo

![Animated OpsMosaic demo: Alex completes onboarding, receives one tip, then returns offline.](terminal.gif)

[Read the complete text transcript](transcript.txt).

The 21-second animation has three scenes: first-run name entry, topic selection
and one random tip, then a returning offline briefing with a rotated comment.
It uses actual CLI output, not invented headlines or hand-written screenshots.
The GIF is a paginated rendering of captured terminal output, not a real-time
screen recording. Application color is explicitly disabled for this capture;
emoji remain visible. Scene timings are chosen for readability.

The example name is Alex. The demo has its own disposable profile/cache, uses UTC
and performs no feed requests. The offline scene uses bundled tips because no
news cache is seeded; its empty-news warning is intentionally retained.
Normal runs use the device timezone. Greetings, dates and selected tips vary
when the demo is regenerated. No displayed command is executed by the demo.

## Regenerate locally

On macOS, install Python 3 and the Apple Command Line Tools (including Swift)
in addition to the usual Go build prerequisites. No Python packages, Swift
packages, screen-recording permissions or external rendering service are used.
The installed OpsMosaic executable does not need Python or Swift.

From the checkout:

```sh
make test-demo
make demo
```

This builds a temporary executable, captures a 100-column pseudo-terminal and
renders the three scenes using macOS AppKit/ImageIO. It replaces only the
generated GIF and transcript in this directory. The renderer verifies frame
count, dimensions and decodability and rejects overflowing text. The capture
checks one tip per briefing, the example name/timezone, absence of terminal
escape sequences, and no repeated onboarding on return. Unit tests exercise
prompt/reply handling, errors, missing prompts, timeouts and output contracts.

The rendering tool is macOS-only; this is not a new platform restriction on the
Go CLI. `make ci` stays independent from optional media-generation tools.
Review the generated assets before committing.
Generation does not push or upload anything.
