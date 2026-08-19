# Contributing

Thanks for helping build show-mapper! This file covers process; architectural
rules live in [AGENTS.md](AGENTS.md) and [docs/architecture.md](docs/architecture.md).

## Setup

Prereqs: **Go 1.24+**, **Node 22+**, git-bash (Windows) or any bash.
For MIDI-enabled builds additionally a C/C++ toolchain -
see [docs/releasing.md](docs/releasing.md#build-prerequisites-per-os).

```bash
git clone <repo> && cd show-mapper
make web && make build-nocgo   # UI + binary w/o MIDI
make check                     # must be green before PRs
```

Dev loop: `make run` (API :8080) + `make web-dev` (UI :5173, hot reload).

## Ground rules

- **Small, focused PRs.** One connector / one profile / one fix per PR.
- `make check` green before review: gofmt, `go vet`, `go test`,
  `npm run check`, `npm run build`.
- Conventional commits: `feat(midi): add apc-40 profile`, `fix(osc): …`.
  Scopes: `config, conductor, midi, osc, streamdeck, server, web, ci, docs, deps`.
- Update docs in the same PR (README status table if applicable). Wire-type
  changes (Go structs ↔ UI) must regenerate types: `make types` and commit
  `web/src/lib/generated/*` (CI enforces freshness).
- New dependency? Justify in the PR description (license + audit note).
- Don't commit `show-mapper.yaml`, built frontend assets, or secrets.

## Adding a built-in MIDI board profile (very welcome!)

1. Verify the note/CC/LED map **physically**: `show-mapper midi monitor <board>`
   while pressing/moving every control; check LED messages with a MIDI monitor
   (e.g. show-mapper's `midi monitor` in send direction is not needed - LEDs are
   sent by the app).
2. Add a profile constructor + LED backend in `internal/sources/midi/profile_<brand>.go`,
   following `profile_apc.go`. Cite the protocol source (manufacturer PDF/wiki)
   in a comment and link it in docs/midi-devices.md.
3. Mark anything unverifiable `TODO(hardware)` - honest data beats wrong data.
4. Unit test the mapping (see `profile_test.go`) - note↔control round trip +
   at least one LED message vector.

Good first issues: APC 20 / APC 40 mk1+mk2 profiles (they're requested by the
project owner!), LED palette verification for `apc-mini` (mk1), custom-board
MIDI-learn→profile wizard, Stream Deck source (HIDs, see AGENTS.md recipes).

## Adding connectors

Source contract: `core.Source`; target contract: `core.Target` - both in
[AGENTS.md](AGENTS.md) recipes + [docs/architecture.md](docs/architecture.md).
Planned next: **Stream Deck source**, **ArtNet/sACN targets**, **timecode**.

## Code of conduct / questions

Be kind, be pragmatic. Open an issue before large refactors. Show-network
gear is unforgiving: prefer boring, well-tested code over clever code.
