#!/usr/bin/env bash
# Build the release archives with embedded Dolt bundled (cgo). RFC 0024.
#
#   scripts/build-release.sh <version> [outdir]
#
# Writes issuegraph_<version>_<os>-<arch>.tar.gz (zip on windows) and
# SHA256SUMS to outdir (default: dist/<version>).
#
# Needs a macOS host: darwin targets use the Xcode clang, linux and windows
# targets use `zig cc` purely as a C cross-compiler for the cgo parts.
set -euo pipefail

version="${1:?usage: scripts/build-release.sh <version> [outdir]}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="${2:-$root/dist/$version}"

[ "$(uname -s)" = Darwin ] || { echo "build-release.sh: run on macOS (darwin targets need Xcode clang)" >&2; exit 1; }
for tool in go node zig clang shasum tar zip git; do
  command -v "$tool" >/dev/null || { echo "build-release.sh: missing tool: $tool" >&2; exit 1; }
done

# Same toolchain pin as the Makefile, so compile and link tools match.
toolchain="$(sed -n 's/^toolchain go//p' "$root/apps/cli/go.mod")"
[ -n "$toolchain" ] && export GOTOOLCHAIN="go$toolchain"

build_id="$(git -C "$root" rev-parse --short HEAD)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$out"

echo "Generating embedded skill assets..."
node "$root/scripts/generate-issuegraph-assets.mjs" >/dev/null

# build <os> <arch> <cc> <cxx>
build() {
  local os="$1" arch="$2" cc="$3" cxx="$4" exe="issuegraph" asset
  [ "$os" = windows ] && exe="issuegraph.exe"
  asset="issuegraph_${version}_${os}-${arch}"
  local stage="$work/$os-$arch"
  mkdir -p "$stage"

  echo "Building $os/$arch ..."
  CGO_ENABLED=1 GOOS="$os" GOARCH="$arch" CC="$cc" CXX="$cxx" \
    go -C "$root/apps/cli" build -tags gms_pure_go \
      -ldflags="-X main.Build=$build_id" -o "$stage/$exe" .

  if [ "$os" = windows ]; then
    (cd "$stage" && zip -q "$out/$asset.zip" "$exe")
  else
    tar -C "$stage" -czf "$out/$asset.tar.gz" "$exe"
  fi
}

build darwin  arm64 "clang"                                 "clang++"
build darwin  amd64 "clang -arch x86_64"                    "clang++ -arch x86_64"
build linux   amd64 "zig cc -target x86_64-linux-gnu.2.17"  "zig c++ -target x86_64-linux-gnu.2.17"
build linux   arm64 "zig cc -target aarch64-linux-gnu.2.17" "zig c++ -target aarch64-linux-gnu.2.17"
build windows amd64 "zig cc -target x86_64-windows-gnu"     "zig c++ -target x86_64-windows-gnu"
build windows arm64 "zig cc -target aarch64-windows-gnu"    "zig c++ -target aarch64-windows-gnu"

(cd "$out" && shasum -a 256 issuegraph_"${version}"_* > SHA256SUMS)
echo "Wrote $(ls "$out" | wc -l | tr -d ' ') files to $out"
