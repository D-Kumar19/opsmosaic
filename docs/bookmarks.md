# Bookmarks and replay

Normal briefings remember their complete last output as a private local snapshot.
Replay it without fetching or advancing greetings or content history:

```sh
opsmosaic replay
opsmosaic replay --detailed --plain
```

Only content is replayed, not the original greeting. The header says
`Saved content | captured ... | not live news`. Source, link, command and original
publication timestamp remain attached. Capture time is the time of the original
briefing, not the current time or the time you bookmark it. Saved output honors
the usual terminal detection, sanitization, width and color policy.

## Keep an item

```sh
opsmosaic bookmarks add          # Last briefing contained exactly one item
opsmosaic bookmarks add 2        # Second item in the last multi-item briefing
opsmosaic bookmarks list
opsmosaic bookmarks show <ID> --plain
opsmosaic bookmarks remove <ID>
```

Replace `<ID>` with the full identifier from `add` or `list`. For multiple items,
an explicit 1-based number is required, matching display order. The default
briefing still contains exactly one random item.

Bookmarks last until manually removed. At most 100 may be stored: adding a new
one at the limit fails, never evicts another. Saving a duplicate reuses the
existing bookmark without changing its content or original capture time.
News/releases deduplicate by kind and canonical article URL; tips use their
stable tip ID, so distinct tips sharing documentation remain separate.

No snapshot yet means `replay` and `add` fail clearly; an empty `list` succeeds.
Replay, list and show do not require a profile, cache, configured source or
network. They work even if the source has disappeared from the cache.

## Persistence and interruption

Version-1 `saved.json` sits beside the profile, including when using
`OPSMOSAIC_HOME` or paired explicit configuration/cache paths. It contains the
last briefing plus independent bookmarks, not a reference to expiring cache.
Writes use atomic replacement and mode 0600; the file is bounded to 16 MiB.
Do not attach it to bug reports: saved titles, commands and URLs may be private.
Uninstalling the executable leaves this state intact.

Normal displays and bookmark mutations use the existing bounded state lock.
Concurrent adds/removals reload under that lock; busy mutations fail explicitly.
Replay/list/show never acquire it or write state. A busy normal briefing's
read-only fallback **does not replace replay**: replay is the last successfully
persisted normal briefing, not necessarily the last terminal window's output.
Scheduled refresh, doctor and help/version do not replace it.

Invalid/unreadable saved state is an explicit error, not silently reset.
Doctor reports `saved_invalid` without printing saved content or identifiers.
Inspect/repair or back up the file locally; no automatic recovery discards data.

Cache/history and saved snapshots are separate atomic files, not a multi-file
transaction. After successful rendering, cache/history is saved first, then the
snapshot. A snapshot-save failure exits nonzero and explicitly reports that
display/cache succeeded but replay did not update. The previous saved file is
retained. Likewise, an output failure printing confirmation after a bookmark
write does not undo the persisted change; inspect `list` before retrying.
