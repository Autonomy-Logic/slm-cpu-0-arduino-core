#!/usr/bin/env bash
# Cross-compile slm-cpu-0-netflash for every host the index declares, into
# <out>/<arduino host>/. CGO off for static binaries; -trimpath and a zeroed
# build id for reproducible output.
#
#   tools/ci/build-uploader.sh dist/tools 0.1.0
set -euo pipefail
OUT="${1:-dist/tools}"
VERSION="${2:-dev}"
mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"
MODULE="$(cd "$(dirname "$0")/../slm-cpu-0-netflash" && pwd)"
PKG="./cmd/slm-cpu-0-netflash"
NAME="slm-cpu-0-netflash"

build() {
  local host=$1 goos=$2 goarch=$3 ext=${4:-}
  local dir="$OUT/$host"
  mkdir -p "$dir"
  ( cd "$MODULE" && env GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -trimpath -buildvcs=false -ldflags "-s -w -buildid= -X main.version=$VERSION" -o "$dir/$NAME$ext" "$PKG" )
  printf "  %-22s %s\n" "$host" "$(ls -lh "$dir/$NAME$ext" | awk '{print $5}')"
}

echo "==> building $NAME $VERSION"
build x86_64-pc-linux-gnu  linux   amd64
build aarch64-linux-gnu    linux   arm64
build x86_64-apple-darwin  darwin  amd64
build arm64-apple-darwin   darwin  arm64
build i686-mingw32         windows amd64 .exe

# macOS kills an unsigned arm64 binary; the Go linker adds an ad-hoc signature.
python3 - "$OUT/arm64-apple-darwin/$NAME" <<'PY'
import struct, sys
with open(sys.argv[1], "rb") as f:
    data = f.read()
magic, _, _, _, ncmds = struct.unpack_from("<IIIII", data, 0)
if magic != 0xFEEDFACF:
    sys.exit(f"not a 64-bit Mach-O: 0x{magic:08X}")
off = 32
for _ in range(ncmds):
    cmd, size = struct.unpack_from("<II", data, off)
    if cmd == 0x1D:  # LC_CODE_SIGNATURE
        print("    arm64-apple-darwin: ad-hoc signature present")
        sys.exit(0)
    off += size
sys.exit("arm64-apple-darwin binary is unsigned and would be killed on Apple Silicon")
PY
