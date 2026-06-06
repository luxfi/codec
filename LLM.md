# codec — ARCHIVED (Wave 2G-Cascade complete)

## Status — ARCHIVED 2026-06-06

`github.com/luxfi/codec` is **archived** on GitHub. The repository is
read-only; tagged versions (v1.0.0..v1.1.5) remain resolvable via the
Go module proxy. The Wave 2G-Cascade landed the upstream re-tags
(p2p v1.21.1, node v1.29.3, sdk v1.17.6, vm v1.1.11, geth v1.16.99,
evm v0.19.4, precompile v0.5.37, coreth v1.23.3) that unblock every
downstream `go mod tidy` after codec/jsonrpc was dropped in v1.1.5.

Successor locations:
- `codec/zapcodec` → **`github.com/luxfi/zapcodec`** (v1.0.0+)
- `codec/jsonrpc` → **`github.com/luxfi/utils/json`** (utils v1.1.5+)
- `codec/wrappers` (canonical) → **`github.com/luxfi/utils/wrappers`** (utils v1.1.5+)
- `codec.Manager` (polymorphic) → **archived; consumers migrated to ZAP kind-byte dispatch via `proto/zap_codec`, `vms/pcodecs`, etc.**

## Pre-archive context

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

## Module lifecycle — ARCHIVED 2026-06-06

The Wave 2G-Cascade is complete:

| Repo | Cascade tag | Cascade action |
|---|---|---|
| `luxfi/p2p` v1.21.1 | already on remote | drops `codec/jsonrpc` import in `peer/peer.go` |
| `luxfi/vm` v1.1.11 | shipped 2026-06-06 | tidy floor refresh; codec demoted to indirect |
| `luxfi/sdk` v1.17.6 | shipped 2026-06-06 | bumps p2p v1.21.1, vm v1.1.11; codec demoted to indirect |
| `luxfi/node` v1.29.3 | shipped 2026-06-06 | bumps p2p v1.21.1, sdk v1.17.6, vm v1.1.11; codec demoted to indirect |
| `luxfi/geth` v1.16.99 | shipped 2026-06-06 | bumps p2p v1.21.1, vm v1.1.11, precompile v0.5.37 |
| `luxfi/precompile` v0.5.37 | shipped 2026-06-06 | bumps p2p v1.21.1, vm v1.1.11 |
| `luxfi/evm` v0.19.4 | shipped 2026-06-06 | bumps p2p v1.21.1, vm v1.1.11, geth v1.16.99 |
| `luxfi/coreth` v1.23.3 | shipped 2026-06-06 | bumps p2p, node, sdk, vm, geth, precompile |
| Downstream modules (utxo, warp, precompile/e2e, zwing, crypto, evmgpu, chains, evm, runtime, kms, keys, container, oracle, relay, lpm, consensus/examples, genesis, genesis/cmd, genesis/builder) | shipped 2026-06-06 | each bumped pins and tidied; transitive codec/jsonrpc dependency removed |

Skipped (pre-existing breakage unrelated to cascade — listed for future work):
- `luxfi/proto` — uses removed `node/proto/pb/warp` (pre-existing rip target)
- `luxfi/cli` — `luxfi/evm` v0.8.49 needs `EtnaTime` field, `luxfi/sdk/api` v0.0.2 needs `avajson.Uint64`, plus signature drift in `coreth/atomic/export_tx.go`
- `luxfi/state` and `luxfi/benchmarks` — both reference removed `luxfi/geth/crypto` package
- `luxfi/netrunner` — k8s.io transitive v0.36.1 dropped `autoscaling/v2beta1`, `scheduling/v1alpha1` (pre-existing k8s upgrade lag)

Tagged versions (v1.0.0..v1.1.5) remain resolvable via the Go module
proxy. Codec imports in tagged dependencies continue to work; only new
pushes to the repo are blocked.

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
