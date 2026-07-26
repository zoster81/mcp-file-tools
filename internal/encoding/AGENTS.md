# Encoding Subsystem Agent Guide

This guide applies to `internal/encoding/`. Follow the repository root [`AGENTS.md`](../../AGENTS.md) first.

## Invariants

- Detect encodings from byte structure and decoded-content evidence only. Never inspect or bias by path, basename, extension, directory, or domain-specific filename.
- BOM detection is authoritative and must check UTF-32 signatures before UTF-16 prefixes.
- The same byte sequence must produce the same result in every filename and detection mode.
- Prefer an empty or low-confidence result over a confident false classification.
- Validate malformed Unicode explicitly. Do not silently repair isolated surrogates, truncated code units, replacement runes, or invalid round trips.
- Binary data with NUL patterns must not be accepted as UTF-16 from parity evidence alone.
- Keep registered encoding names and aliases centralized in `registry.go`.
- Confidence thresholds are public behavior through tool output; change them only with focused regression evidence and documentation.

## Implementation guidance

Keep candidate scoring deterministic and composable. Separate structural validation, decoding, text-quality metrics, binary rejection, round-trip validation, and final confidence selection so each signal can be tested independently.

Sampling, chunked, and full modes must share the same candidate semantics. Account for chunk boundaries before claiming mode equivalence.

Keep encoding detection, incremental decoding, and consuming operation policy separate. Detection works on `ReaderAt` evidence; streaming consumers use registered decoder/encoder readers without adding filename-based behavior.

## Tests

Add table-driven tests for:

- identical bytes under different filenames;
- UTF-16 LE and BE, with and without BOM;
- valid surrogate pairs and mixed scripts;
- odd byte lengths and isolated/truncated surrogates;
- empty, BOM-only, short, and ambiguous content;
- UTF-8, ASCII, legacy code pages, GBK/GB18030, and malformed inputs;
- executable, archive, image, random-byte, alternating-NUL, and sparse-NUL false positives;
- sample, chunked, and full mode consistency.

Use synthetic bytes or checked-in fixtures with clear provenance. Do not use extension-specific expectations.

## Verification

```bash
go test ./internal/encoding -count=1
go test ./filetoolsserver/handler -count=1
go test ./... -count=1
git diff --check
```

Run fuzz tests and benchmarks when changing parsers, scoring loops, confidence thresholds, or large-input behavior.
