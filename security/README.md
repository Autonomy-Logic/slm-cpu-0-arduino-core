# Security — SBOM and vulnerability flows

Two separate flows, run by GitHub Actions:

| Flow | What it is | When it runs | Output |
|---|---|---|---|
| **SBOM** (`security-sbom.yml`) | The bill of materials: every dependency with its version and licence (CycloneDX 1.6, SPDX 2.3, components CSV) | Once per release: a commit landing on `main`. A released version's SBOM never changes, so it is not regenerated on a schedule | Branch `security-archive`, folder `pr-<n>-<date>/` |
| **Vulnerabilities, PR check** (`security-pr-gate.yml`) | osv-scanner on the PR head and base | Every PR into `development` or `main` | Blocks only a **critical advisory the PR introduces**; pre-existing advisories never block. Sticky PR comment, no SBOM, no report |
| **Vulnerabilities, release report** (`security-release-vulns.yml`) | Scan plus the vulnerability report (md, html, pdf) | Once per release, next to the SBOM | Same `security-archive` folder as the SBOM |
| **Vulnerabilities, monthly** (`security-monthly.yml`) | Scan, Claude triage of new advisories, vulnerability report | 1st of every month, 06:00 UTC, and on demand | A PR into `main` with `security/<YYYY-MM>/` |

The vulnerability flows read the manifests directly and honour the VEX baseline
in `../osv-scanner.toml`; they do not need the SBOM. The report's *raw* scan is
run with `--config=security/osv-raw.toml`, an empty config: without `--config`,
osv-scanner applies the `osv-scanner.toml` of the scanned directory, and the raw
and suppressed counts would be identical. A commit on `main` that only
touches `security/` and `osv-scanner.toml` (such as the merged monthly PR) is not
a release, so the release flows skip it.

```
security-archive (orphan branch)
└── pr-<n>-<date>/
    ├── <name>.cdx.json · <name>.spdx.json · <name>.components.csv · PROVENANCE-sbom.txt   ← SBOM flow
    └── vulnerabilities.csv · <Title>-Vulnerability-Report.md/.html/.pdf · PROVENANCE-vulns.txt ← vulnerability flow

security/ (this folder, on the code branches)
├── <YYYY-MM>/                       ← monthly vulnerability snapshot
├── LATEST                           ← text file naming the current month
├── components.json                  ← hand-maintained inventory of the upstream sources and toolchains
└── report-config.json               ← report DATA (VEX triage + narrative)
```

(`LATEST` is a plain text file rather than a symlink: a committed symlink
materializes as a text file on Windows checkouts anyway, so it may as well be one
everywhere.)

> `on: schedule` only fires from the repository's **default branch**. To test the
> release flows without touching the real evidence, run them with **Run workflow**
> on any branch other than `main`; they write to `archive_branch` (default
> `security-archive-test`). On `main` the input is ignored and the target is always
> `security-archive`, and `scripts/security-archive.sh` refuses to write
> `security-archive` from any other ref.

## What is covered

This repository is an Arduino platform: C/C++ sources downloaded at build time
plus a Go uploader. There is no package manager for the C/C++ parts, so coverage
is split:

| Part | In the SBOM | Scanned for advisories |
|---|---|---|
| `slm-cpu-0-netflash` (Go, `tools/slm-cpu-0-netflash/go.mod`) | Yes, by cdxgen (`-t go`) | Yes, by osv-scanner. It has no third-party modules today, so an empty result is normal |
| Go standard library linked into the uploader | Yes, at the toolchain version that ran the SBOM job (`go env GOVERSION`, same `1.22` line as `release.yml`) | No: osv-scanner does not report the stdlib from a `go 1.22` directive |
| STM32duino core, STM32Ethernet, LwIP (bundled in the platform archive) | Yes, from `components.json`, with version, licence and the SHA-256 pinned in `tools/ci/upstream.json` | No: not in any OSV ecosystem. Checked by the monthly triage step and by hand |
| Arm GCC toolchain, CMSIS, STM32Tools (referenced from the board manager index) | Yes, from `components.json`, marked `autonomy:distribution=referenced` | No, same as above |
| The FACTS slot backplane driver | No: proprietary, not part of this repository or its releases | No |

`scripts/merge-components.mjs` merges `components.json` into the cdxgen output
and **fails the SBOM run** if a version or digest disagrees with
`tools/ci/upstream.json`, or if `upstream.json` lists an archive or tool that
`components.json` does not. When you bump an upstream, update both files in the
same PR. Licences come from each upstream's own licence file (see `NOTICE.md`).

Because `components.json` lives under `security/`, a commit on `main` that
changes only that file is not treated as a release; the next release picks it up.

## What stops a bad report from shipping

The chain has to fail loudly rather than produce a clean-looking document from
nothing. Three guards do that:

- **`scripts/assert-sbom.mjs`** — a *content* floor on the generated SBOM (8
  components here: the inventory alone clears it, a broken merge does not). Every
  other stage validates only shape, so an empty or partial CycloneDX file used to
  travel the whole chain and come out as a Confidential report attesting a clean
  posture over zero components. Below the floor, the run stops.
- **Untriaged detection** — each entry in `affected[]` / `mitigated[]` /
  `notAffected[]` **must** list the advisory `ids` it accounts for (any alias
  works; the monthly validation step fails on an entry without `ids`).
  `build-report.mjs` compares those against the live post-suppression scan;
  anything unclaimed gets its own **section 4a — Untriaged in This Scan**, the
  executive summary is marked *Assessment incomplete*, and the stored posture
  statement is labelled as the last full assessment that does not cover them,
  instead of as the headline posture. The document is allowed to be incomplete; it
  is not allowed to be quietly self-contradictory.
- **`scripts/lib/severity.mjs` and `scripts/lib/cdx.mjs`** — one severity verdict
  and one component-collection walk, shared by the gate, the report, the
  vulnerabilities CSV, the SPDX converter and the CSV converter. An advisory is
  one OSV alias group per package (a GHSA and its CVE twins count once); its
  severity is the highest parsable severity of any member or of the group's
  `max_severity`, and **UNKNOWN** when there is none. UNKNOWN is counted and
  listed, and never blocks the gate. `vulnerabilities.csv` is written from the
  same index, so its row count equals the register count in the report. The
  release SBOM flow asserts the three component counts agree before archiving.
- **SPDX supplier** — taken from the CycloneDX `supplier`, `manufacturer`,
  `publisher` or `authors` of each component, and `NOASSERTION` otherwise. Every
  entry of `components.json` carries a `supplier`, which
  `scripts/merge-components.mjs` copies into the CycloneDX component. The SBOM
  does not claim NTIA minimum-elements conformance.

## The suppression baseline (`../osv-scanner.toml`)

Expiry dates are **staggered on purpose** — see the header comment in that file.
The offset from the review date comes from the VEX justification and is then
shifted by severity (CRITICAL is pulled to the first review, LOW/MODERATE pushed
back), so no single month inherits the whole baseline for re-review.

Suppressions written by the automated triage step carry
`AI-TRIAGED [unreviewed]` in their `reason`. **The AI step has authority to write
security suppressions and the only control is human review of the PR diff** —
clearing that marker is the reviewer's acknowledgement that a person checked it.
A step after the triage re-parses both files, refuses a net *loss* of
suppressions, refuses more than **15 new suppressions in one run**, and refuses a
triage entry without `ids`, before anything is rendered. The prompt tells the
model that advisory text is untrusted data, never instructions.

To re-baseline by hand from a triaged register:

```sh
node scripts/gen-osv-ignores.mjs sbom/vulnerabilities.csv 2027-01-01 >> osv-scanner.toml
```

It refuses to run without a per-row verdict column, and it will not suppress an
advisory that any row triages as requiring action.

## One-time setup — run Claude on your subscription (no API key)

Claude runs inside the workflow on **your Claude subscription** (Pro/Max), not a
pay-per-token API key:

1. On your machine: `claude setup-token` → complete the browser login → copy the
   token it prints (`CLAUDE_CODE_OAUTH_TOKEN`).
2. In the repo: **Settings → Secrets and variables → Actions → New repository
   secret** → name `CLAUDE_CODE_OAUTH_TOKEN`, value = the token.
3. Done. The monthly workflow will authenticate as you. (If the secret is absent,
   the workflow still runs and builds the report deterministically — it just
   skips the AI triage step.)

> To move billing off a personal account later (e.g. an org API key), replace the
> secret with `ANTHROPIC_API_KEY` and swap the one input in the workflow — nothing
> else changes.

## Reviewing the monthly PR

- **No new advisory:** confirm the summary, approve, merge (the snapshot is archived).
- **New advisory, not exploitable:** check Claude's justification in the
  `osv-scanner.toml` diff, drop the `AI-TRIAGED [unreviewed]` marker from the
  entries you have verified, then merge.
- **New advisory, exploitable:** bump the dependency (separate PR) and merge the
  report that documents it.
- **Report says *Assessment incomplete*:** section 4a lists advisories nothing
  accounted for. Each one needs either a dependency bump or a documented
  `not_affected` determination with its id recorded in `report-config.json`.
