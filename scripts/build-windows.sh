#!/usr/bin/env sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
: "${CC:=x86_64-w64-mingw32-gcc}"
: "${WINDRES:=x86_64-w64-mingw32-windres}"
export CC
export GOOS=windows GOARCH=amd64 CGO_ENABLED=1
mkdir -p dist
"$WINDRES" -I scripts -i scripts/windows.rc -o cmd/regexfileextractor/resource_windows_amd64.syso -O coff
go build -tags 'migrated_fynedo,no_emoji' -trimpath -ldflags '-s -w -H=windowsgui -extldflags=-static' -o dist/RegexFileExtractor.exe ./cmd/regexfileextractor
if command -v sha256sum >/dev/null 2>&1; then
  (cd dist && sha256sum RegexFileExtractor.exe > SHA256SUMS.txt)
else
  (cd dist && shasum -a 256 RegexFileExtractor.exe > SHA256SUMS.txt)
fi
echo "Built dist/RegexFileExtractor.exe"
