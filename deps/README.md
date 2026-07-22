# deps/

Vendored third-party source that is not consumed as a normal module.

## discordgo-self

A fork of [`discordgo-self`](https://github.com/hytams/discordgo-self), the
self-bot (user-token) variant of discordgo. It is checked in here and wired up
through a `replace` directive in [`go.mod`](../go.mod):

```
replace github.com/hytams/discordgo-self => ./deps/discordgo-self
```

### Why it is vendored rather than required

The upstream module is not versioned in a way `go get` can track, and the
project needs small patches to it that have no upstream release. Copying the
source in makes the build reproducible without waiting on a tag.

### What this costs

A `replace` to a local path is invisible to `go get -u`, `go list -m -u`,
Dependabot, and `govulncheck`'s module graph. Nothing will tell you when
upstream fixes a bug or a CVE — that check is manual.

### Keeping it current

1. Diff against upstream to see what moved:
   `git diff --no-index deps/discordgo-self <path-to-fresh-upstream-checkout>`
2. Re-apply local patches on top of the newer source.
3. Run `go build ./... && go test ./...`.

### Local patches

Record every deliberate divergence from upstream here, so step 2 above is a
checklist rather than an archaeology exercise.

| Area | Change | Why |
| ---- | ------ | --- |
| _(none recorded yet)_ | | |

If you find a difference from upstream that is not in this table, it is either
an undocumented patch or upstream drift — resolve it and add a row.
