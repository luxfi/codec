# codec — STRUCTURAL FLOOR (Wave 2G-Archive)

## Status

`github.com/luxfi/codec` is the legacy polymorphic Marshal/Unmarshal codec
used by classical Lux wire formats: `codec.Manager` dispatches across
registered versions, each version is backed by a `linearcodec.Codec`
which uses `reflectcodec` to walk `serialize:"true"` struct tags.

**Wave 2G-Archive (current state)**: every production caller has been
migrated off this module. The remaining importers are:

| Importer | Reason |
|---|---|
| `bench/modes/zap_vs_codec/fixtures.go` | comparative bench — codec/linearcodec IS the head-to-head baseline by construction. Pinned to `v1.1.5`. |

All other former importers migrated:
- `*/wrappers` → `github.com/luxfi/utils/wrappers` (Wave 2F)
- `*/jsonrpc` → `github.com/luxfi/utils/json` (Wave 2F, v1.1.5)
- `codec.Manager` users in node/vms/* → `pcodecs.Codec` (Wave 2D)
- `codec.Manager` users in vm/chains/atomic → hand-rolled big-endian (Wave 2E)
- `codec.Manager` users in proto/p/*, proto/x/* → ZAP wire (Wave 2A, 1A)
- `codec/zapcodec` → **`github.com/luxfi/zapcodec`** (Wave 2G-Archive)
- proto/zap_codec, sdk wallet, node/vms/pcodecs, coreth secpfx, proto test
  helpers → all migrated by Wave 2G-Wallet / 2G-Genesis / 2G-Internal

## Decomposition of the Module (current state)

| Subpackage | Status |
|---|---|
| `codec` (top-level) | `Manager`, `Codec`, `Registry`, `GeneralCodec` — frozen at v1.1.5, no further changes |
| `codec/linearcodec` | the only codec impl callers still ask for via `NewDefault`; kept frozen as the BE baseline in `bench/modes/zap_vs_codec` |
| `codec/reflectcodec` | `linearcodec`'s backing type-walker; internal to this module |
| `codec/codecmock` | mock generator — used by tests that mock `codec.Manager` |
| `codec/wrappers` | retained for `codec`'s own internal use; external canonical home is `luxfi/utils/wrappers` |
| ~~`codec/zapcodec`~~ | **EXTRACTED in Wave 2G-Archive** — moved to `github.com/luxfi/zapcodec` v1.0.0+. Codec module no longer carries the LE codec. |
| ~~`codec/hierarchycodec`~~ | DELETED in Wave 2F — zero external importers |
| ~~`codec/jsonrpc`~~ | DELETED in Wave 2F — canonical home is `luxfi/utils/json` (v1.1.5) |

## Module lifecycle — Pending GitHub archive

The module is **structurally frozen** at v1.1.5. The only remaining task
is `gh repo archive luxfi/codec`. That archive is **blocked** until the
upstream tagging cascade lands:

| Repo | Pinned p2p / sdk version | Action needed |
|---|---|---|
| `luxfi/p2p` v1.19.2 | (head)→v1.21.1+ has codec rip | Need consumers to bump pin past v1.21.0 |
| `luxfi/node` v1.23.36 | pins p2p v1.19.2 | re-tag pinning p2p v1.21.1+ |
| `luxfi/sdk` v1.16.48 | pins p2p v1.19.2 | re-tag pinning p2p v1.21.1+ |

Once those re-tag, every transitive importer of `luxfi/codec` will drop
the `// indirect` line on `go mod tidy`, and the module can be archived
with no surprises for downstream consumers. The v1.1.5 tag stays
resolvable via the Go module proxy even after `gh repo archive` — codec
imports in tagged dependencies continue to work; only new pushes to the
repo are blocked.

## Why this floor exists

The polymorphic codec.Manager pattern (versioned, dynamic-type registry)
is exactly what Lux moved AWAY from in the ZAP-native design (kind-byte
dispatch, value-typed registries, schema IDs as Option-B map entries —
see LP-208/211/218/220/300). The Wave 2G-Archive extraction completes
this transition: the ZAP-native wire format now has its own module
(`luxfi/zapcodec`), the legacy BE codec has its own (this module),
and no production caller depends on the legacy wiring.

## Bench module is the only post-archive consumer

`bench/modes/zap_vs_codec/fixtures.go` deliberately retains the
`luxfi/codec@v1.1.5` import. The bench's whole purpose is to measure
the new ZAP wire format against the old codec wire format — codec IS
the comparison target by construction. After GitHub archive, the v1.1.5
tag stays available via the module proxy; the bench continues to compile
and run. See `bench/modes/zap_vs_codec/README.md` for headline numbers.
