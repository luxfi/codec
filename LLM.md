# codec — STRUCTURAL FLOOR (Wave 2F)

## Status

`github.com/luxfi/codec` is the legacy polymorphic Marshal/Unmarshal codec
used by classical Lux wire formats: `codec.Manager` dispatches across
registered versions, each version is backed by a `linearcodec.Codec`
which uses `reflectcodec` to walk `serialize:"true"` struct tags.

After Wave 2A (proto/p, proto/x rip — codec deleted from production),
Wave 2D (node/vms — codec consolidated through `node/vms/pcodecs`),
Wave 2E (vm/chains/atomic — codec deleted via hand-rolled binary),
Wave 2F (utility subpackages — codec/wrappers ⇒ luxfi/utils/wrappers;
codec/jsonrpc ⇒ luxfi/utils/json), the codec module has **10 remaining
importers**:

| Importer | Reason — why it can't go yet |
|---|---|
| `bench/modes/zap_vs_codec/fixtures.go` | head-to-head benchmark comparing codec.Manager vs ZAP on identical struct shapes — by definition it imports codec |
| `genesis/builder/builder.go` (`pvmGenesisCodec`, `newXVMParserCodecs`) | proto/p + proto/x carry no codec import; the genesis builder constructs the codec.Manager + linearcodec wiring inline so proto/p/genesis can encode/decode the genesis blob |
| `sdk/wallet/chain/x/constants.go` + `x/builder/constants.go` + `p/pcodecs/pcodecs.go` | wallet tx builders construct the X- and P-chain wire codecs to sign txs that mainnet/testnet validators accept |
| `coreth/plugin/evm/secpfx/secpfx.go` | secp256k1 fee extension — implements the `codec.Registry` interface to plug into the C-chain's fx system |
| `proto/internal/{p,x,pvm}codectest` | test-only bridge: proto/p tests need a `codec.Manager` to round-trip wire formats but proto/p itself has no codec import; these helpers re-export `codec.NewManager`, `linearcodec.NewDefault`, etc. for the test suite |
| `node/vms/pcodecs/{pcodecs,pcodecsmock}.go` | Wave 2D consolidation target — every legacy `node/vms/*` Marshal/Unmarshal call routes through `pcodecs.Codec` (a `codec.Manager` type alias). Until pcodecs itself goes ZAP-native, codec lives. |

All other former importers have been migrated:

- `*/wrappers` → `github.com/luxfi/utils/wrappers` (canonical Packer + Errs)
- `*/jsonrpc` → `github.com/luxfi/utils/json` (new canonical home for
  Uint8/16/32/64, Float32/64, NewCodec — added in utils v1.1.5)
- `codec.Manager` users in node/vms/* → `pcodecs.Codec` (Wave 2D)
- `codec.Manager` users in vm/chains/atomic → hand-rolled big-endian (Wave 2E)
- `codec.Manager` users in proto/p/*, proto/x/* → ZAP wire (Wave 2A, 1A)

## Decomposition of the Module

| Subpackage | Status |
|---|---|
| `codec` (top-level) | `Manager`, `Codec`, `Registry`, `GeneralCodec` — used by structural floor above |
| `codec/linearcodec` | the only codec impl callers still ask for via `NewDefault` |
| `codec/reflectcodec` | `linearcodec`'s backing type-walker; not used directly externally |
| `codec/hierarchycodec` | legacy variant; no external importers — DEAD |
| `codec/zapcodec` | head-to-head adapter used by `bench/modes/zap_vs_codec`; no production import |
| `codec/codecmock` | mock generator — used by tests that mock `codec.Manager` |
| `codec/wrappers` | DEAD — replaced by `luxfi/utils/wrappers` |
| `codec/jsonrpc` | DEAD — replaced by `luxfi/utils/json` |

## Removal Plan (post-Wave 2F)

1. Migrate `node/vms/pcodecs` to a pcodecs-internal codec impl
   (effectively inline linearcodec + a small Manager) → kills the
   largest remaining cluster of importers in one move.
2. Migrate `genesis/builder` and `sdk/wallet/chain/*` to compose the
   same pcodecs (or proto-internal equivalents) → these all build txs
   the validators accept, so the wire format is fixed; only the
   construction shape changes.
3. Migrate `coreth/plugin/evm/secpfx` once the corresponding C-chain
   fx system goes ZAP-native (Wave 2E follow-up).
4. Delete `codec/hierarchycodec`, `codec/wrappers`, `codec/jsonrpc`
   subpackages.
5. Archive the module.

## Why this floor exists

The polymorphic codec.Manager pattern (versioned, dynamic-type registry)
is exactly what Lux moved AWAY from in the ZAP-native design (kind-byte
dispatch, value-typed registries, schema IDs as Option-B map entries —
see LP-208/211/218/220/300). Every removal of a `codec.Manager` import
is a step toward "one wire format, one canonical schema registry."

The remaining ten importers are the wallet/genesis/fx/bench/test glue —
they sit at the boundary between the Lux primary network's classical
wire format (still consumed by mainnet/testnet validators today) and
the new ZAP-native stack (proto/p, proto/x, node/vms/pcodecs). They
will be lifted as the upstream consumers convert.
