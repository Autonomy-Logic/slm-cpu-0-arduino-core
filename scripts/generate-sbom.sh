#!/usr/bin/env bash
#
# generate-sbom.sh: CycloneDX 1.6, SPDX 2.3 and a components CSV under ./sbom/.
# cdxgen scans the Go uploader; security/components.json adds the upstream C/C++ sources and toolchains.
# Env: SBOM_VERSION, GO_VERSION, MIN_COMPONENTS (default 8).

set -euo pipefail
cd "$(dirname "$0")/.."

NAME="$(node -p "require('./security/components.json').root.name")"
VERSION="${SBOM_VERSION:-$(git describe --tags --always 2>/dev/null || echo "")}"
VERSION="${VERSION#v}"
if [ -z "${GO_VERSION:-}" ] && command -v go >/dev/null 2>&1; then
  GO_VERSION="$(go env GOVERSION)"
fi
export GO_VERSION="${GO_VERSION:-}"

OUT="sbom"
mkdir -p "$OUT"

# Exact pin, checked by scripts/__tests__. Do not paste it through a web page: email obfuscation mangles name@version.
CDXGEN_VERSION="11.11.0"

echo "==> Generating CycloneDX 1.6 SBOM (cdxgen ${CDXGEN_VERSION}, -t go) for ${NAME}@${VERSION:-unversioned}"
FETCH_LICENSE=true npx --yes "@cyclonedx/cdxgen@${CDXGEN_VERSION}" \
  -t go \
  --spec-version 1.6 \
  -o "$OUT/$NAME.cdx.json" \
  --project-name "$NAME" \
  ${VERSION:+--project-version "$VERSION"} \
  .

echo "==> Merging the upstream inventory (security/components.json, checked against tools/ci/upstream.json)"
node scripts/merge-components.mjs "$OUT/$NAME.cdx.json" security/components.json tools/ci/upstream.json "$VERSION"

# Content floor: nothing downstream can tell an empty BOM from a clean one.
echo "==> Validating the SBOM has real content"
node scripts/assert-sbom.mjs "$OUT/$NAME.cdx.json" "${MIN_COMPONENTS:-8}"

echo "==> Converting to SPDX 2.3"
node scripts/cdx-to-spdx.mjs "$OUT/$NAME.cdx.json" "$OUT/$NAME.spdx.json"

echo "==> Emitting flattened CSV"
node scripts/cdx-to-csv.mjs "$OUT/$NAME.cdx.json" "$OUT/$NAME.components.csv"

echo "==> Done. Artifacts in ./$OUT/"
ls -la "$OUT"
