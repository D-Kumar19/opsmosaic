# Project principles

- Small and useful: a short briefing, not an endless feed.
- Local-first: no account, tracking, secrets, or backend required.
- Honest output: distinguish TIP, NEWS, RELEASE and cached content.
- Safety: tips are read-only; external strings cannot control the terminal.
- Reliability: bounded HTTP requests, atomic state, explicit warnings/errors.
- Maintainability: Go standard library first; tests use isolated directories
  and local HTTP servers rather than the user's configuration or live feeds.
- Scope discipline: add a dependency only for a demonstrated need.
- Clarification: before implementation, ask the user about unresolved behavior,
  design, or scope choices; record decisions in the plan/spec rather than
  silently assuming. Backlog tasks do not authorize publication or deployment.
