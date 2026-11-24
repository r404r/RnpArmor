#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
stage="all"
VERSION="dev"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --stage)
      stage="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown arg: $1" >&2
      exit 1
      ;;
  esac
done

case "$stage" in
  deps|rnp|package|all) ;;
  *)
    echo "unsupported stage: $stage" >&2
    exit 1
    ;;
esac

src="$root/out/src"
deps="$root/out/deps"
work="$root/out/mac-build"
dist="$root/out/dist"
mkdir -p "$src" "$deps" "$work" "$dist"

CURRENT_CMD=""
LAST_LOG=""

sanitize_gha() {
  tr '\r\n' '  ' <<<"$1" | sed 's/::/: :/g'
}

emit_error() {
  local title="$1"
  local msg="$2"
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    echo "::error title=$(sanitize_gha "$title")::$(sanitize_gha "$msg")"
  fi
}

emit_notice() {
  local title="$1"
  local msg="$2"
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    echo "::notice title=$(sanitize_gha "$title")::$(sanitize_gha "$msg")"
  fi
}

on_err() {
  local ec=$?
  local tail_text=""
  if [[ -n "$LAST_LOG" && -f "$LAST_LOG" ]]; then
    tail_text=$(tail -n 15 "$LAST_LOG" | tr '\n' ' ' | sed 's/  */ /g')
  fi
  emit_error "build-macos" "cmd='$CURRENT_CMD' exit=$ec tail='${tail_text:-none}'"
  exit "$ec"
}
trap on_err ERR

run_cmd() {
  local name="$1"
  shift
  local log
  log=$(mktemp -t rnparmor-macos-cmd.XXXXXX.log)
  LAST_LOG="$log"
  CURRENT_CMD="$name"
  echo "+ $name"
  "$@" 2>&1 | tee "$log"
}

sha256_file() {
  shasum -a 256 "$1" | awk '{print $1}'
}

pins="$root/deps/pins.json"

read_pin() {
  python3 - "$pins" "$1" <<'PY'
import json, sys
pins = json.load(open(sys.argv[1]))
expr = sys.argv[2].split('.')
v = pins
for p in expr:
    v = v[p]
print(v)
PY
}

rnp_tag=$(read_pin rnp.tag)
rnp_commit=$(read_pin rnp.commit)
botan_sha=$(read_pin botan.sha256)
botan_tar_url=$(read_pin botan.tarball)
json_tag=$(read_pin json_c.tag)
json_tar_url=$(read_pin json_c.tarball)
json_sha=$(read_pin json_c.sha256)

if [[ "$rnp_tag" != "v0.18.1" || "$rnp_commit" != "aad1892e4d8423398a0b973bd0a7e4544c359afc" ]]; then
  echo "rnp pin mismatch" >&2
  exit 1
fi
if python3 - "$pins" <<'PY'
import json,sys
pins=json.load(open(sys.argv[1]))
assert 'v0.18.0' in pins['rnp']['do_not_use']
PY
then :; else
  echo "rnp do_not_use mismatch" >&2
  exit 1
fi
if [[ "$botan_sha" != "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24" ]]; then
  echo "botan pin mismatch" >&2
  exit 1
fi

deps_stage() {
  local t0=$SECONDS
  local botan_prefix="$deps/botan"
  local json_prefix="$deps/json-c"
  local botan_mod_file="$root/scripts/windows/botan3-modules.txt"
  local botan_mod_sha
  botan_mod_sha=$(sha256_file "$botan_mod_file")
  local botan_stamp="$botan_prefix/.modules-sha256"
  local botan_lib="$botan_prefix/lib/libbotan-3.a"
  local botan_header="$botan_prefix/include/botan-3/botan/version.h"
  local json_lib="$json_prefix/lib/libjson-c.a"

  if [[ -f "$botan_stamp" && -f "$botan_lib" && -f "$botan_header" && -f "$json_lib" ]] && [[ "$(tr -d '[:space:]' < "$botan_stamp")" == "$botan_mod_sha" ]]; then
    echo "deps hot cache hit"
    emit_notice "macos-deps" "done in $((SECONDS-t0))s"
    return
  fi

  local botan_tar="$src/Botan-3.10.0.tar.xz"
  if [[ ! -f "$botan_tar" ]]; then
    run_cmd "curl botan" curl -fsSL --retry 3 -o "$botan_tar" "$botan_tar_url"
  fi
  echo "$botan_sha  $botan_tar" | shasum -a 256 -c -

  local json_tar="$src/json-c-0.19.tar.gz"
  if [[ ! -f "$json_tar" ]]; then
    run_cmd "curl json-c" curl -fsSL --retry 3 -o "$json_tar" "$json_tar_url"
  fi
  echo "$json_sha  $json_tar" | shasum -a 256 -c -

  rm -rf "$src/Botan-3.10.0" "$src/json-c-0.19"
  run_cmd "extract botan" tar -C "$src" -xJf "$botan_tar"
  run_cmd "extract json-c" tar -C "$src" -xzf "$json_tar"

  local botan_src="$src/Botan-3.10.0"
  local botan_build="$work/botan"
  rm -rf "$botan_build" "$botan_prefix"
  mkdir -p "$botan_prefix"
  local modules_csv
  modules_csv=$(grep -v '^\s*#' "$botan_mod_file" | sed '/^\s*$/d' | paste -sd, -)
  run_cmd "botan configure" python3 "$botan_src/configure.py" \
    --cc=clang --os=darwin --cpu=arm64 \
    --prefix="$botan_prefix" \
    --build-targets=static \
    --disable-shared-library \
    --minimized-build \
    --enable-modules="$modules_csv" \
    --no-install-python-module \
    --without-documentation \
    --extra-cxxflags="-fPIC"
  run_cmd "botan make" make -C "$botan_src" -j"$(sysctl -n hw.ncpu)"
  run_cmd "botan install" make -C "$botan_src" install
  echo "$botan_mod_sha" > "$botan_stamp"

  local json_src="$src/json-c-0.19"
  local json_build="$work/json-c"
  rm -rf "$json_build" "$json_prefix"
  run_cmd "json-c cmake" cmake -S "$json_src" -B "$json_build" \
    -DCMAKE_INSTALL_PREFIX="$json_prefix" \
    -DBUILD_SHARED_LIBS=OFF \
    -DBUILD_TESTING=OFF \
    -DDISABLE_WERROR=ON \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
    -DCMAKE_POLICY_VERSION_MINIMUM=3.5
  run_cmd "json-c build" cmake --build "$json_build" -j"$(sysctl -n hw.ncpu)"
  run_cmd "json-c install" cmake --install "$json_build"

  mkdir -p "$deps/licenses"
  cp -f "$botan_src/license.txt" "$deps/licenses/botan-license.txt"
  cp -f "$json_src/COPYING" "$deps/licenses/json-c-COPYING"
  emit_notice "macos-deps" "done in $((SECONDS-t0))s"
}

rnp_stage() {
  local t0=$SECONDS
  local rnp_src="$src/rnp"
  if [[ ! -d "$rnp_src/.git" ]]; then
    run_cmd "clone rnp" git clone --depth 1 --branch "$rnp_tag" https://github.com/rnpgp/rnp.git "$rnp_src"
  fi
  local got
  got=$(git -C "$rnp_src" rev-parse HEAD)
  [[ "$got" == "$rnp_commit" ]]

  local sexpp="$rnp_src/src/libsexpp"
  if [[ ! -d "$sexpp/.git" ]]; then
    rm -rf "$sexpp"
    run_cmd "clone sexpp" git clone https://github.com/rnpgp/sexpp.git "$sexpp"
  fi
  run_cmd "fetch sexpp" git -C "$sexpp" fetch --depth 1 origin c641a2f36520bab783657a58650d9fda548b9dec
  run_cmd "checkout sexpp" git -C "$sexpp" checkout --detach FETCH_HEAD

  local rnp_build="$work/rnp"
  local botan_prefix="$deps/botan"
  local json_prefix="$deps/json-c"
  rm -rf "$rnp_build"
  export PKG_CONFIG_PATH="$botan_prefix/lib/pkgconfig:$json_prefix/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"

  run_cmd "cmake rnp" cmake -S "$rnp_src" -B "$rnp_build" \
    -DBUILD_SHARED_LIBS=ON \
    -DBUILD_TESTING=OFF \
    -DDOWNLOAD_GTEST=OFF \
    -DENABLE_DOC=OFF \
    -DENABLE_CRYPTO_REFRESH=OFF \
    -DENABLE_PQC=OFF \
    -DWITH_SEXP_CLI=OFF \
    -DCRYPTO_BACKEND=botan \
    -DBOTAN_ROOT_DIR="$botan_prefix" \
    -DCMAKE_PREFIX_PATH="$botan_prefix;$json_prefix" \
    -DJSON-C_LIBRARY="$json_prefix/lib/libjson-c.a" \
    -DJSON-C_INCLUDE_DIR="$json_prefix/include" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_SHARED_LINKER_FLAGS="-framework Security"
  run_cmd "build librnp" cmake --build "$rnp_build" --config Release --target librnp -j"$(sysctl -n hw.ncpu)"

  local dylib
  dylib=$(find "$rnp_build" -type f \( -name 'librnp.dylib' -o -name 'librnp*.dylib' \) | head -n 1)
  [[ -n "$dylib" ]]
  local out_dylib_dir="$work/rnp-dylib"
  mkdir -p "$out_dylib_dir"
  cp -f "$dylib" "$out_dylib_dir/librnp.dylib"
  run_cmd "set install_name" install_name_tool -id @loader_path/librnp.dylib "$out_dylib_dir/librnp.dylib"

  local otool_out="$work/otool-librnp.txt"
  run_cmd "otool" otool -L "$out_dylib_dir/librnp.dylib"
  otool -L "$out_dylib_dir/librnp.dylib" > "$otool_out"
  python3 - "$otool_out" <<'PY'
import sys
bad=[]
for ln in open(sys.argv[1]):
    s=ln.strip()
    if not s or s.endswith(':'):
        continue
    p=s.split(' (')[0]
    if p.startswith(('@loader_path/','@rpath/','/usr/lib/','/System/Library/')):
        continue
    bad.append(p)
if bad:
    raise SystemExit('forbidden dylib refs: ' + ', '.join(bad))
PY
  emit_notice "macos-rnp" "done in $((SECONDS-t0))s"
}

write_info_plist() {
  local out="$1"
  local short_ver="$2"
  local bundle_ver="$3"
  cat > "$out" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>RnpArmor</string>
  <key>CFBundleDisplayName</key><string>RnpArmor</string>
  <key>CFBundleIdentifier</key><string>com.r404r.rnparmor</string>
  <key>CFBundleExecutable</key><string>rnparmor</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${short_ver}</string>
  <key>CFBundleVersion</key><string>${bundle_ver}</string>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF
}

package_stage() {
  local t0=$SECONDS
  local out_bin="$work/rnparmor"
  local app_root="$dist/RnpArmor.app"
  local app_contents="$app_root/Contents"
  local app_macos="$app_contents/MacOS"
  local app_res="$app_contents/Resources"
  local app_lic="$app_res/licenses"
  local dylib="$work/rnp-dylib/librnp.dylib"

  run_cmd "go build darwin arm64" env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.appVersion=${VERSION}" -o "$out_bin" ./cmd/rnparmor

  rm -rf "$app_root"
  mkdir -p "$app_macos" "$app_lic"

  local short_ver="$VERSION"
  local bundle_ver="$VERSION"
  if [[ "$VERSION" == "dev" ]]; then
    short_ver="0.0.0"
    bundle_ver="0.0.0"
  fi
  write_info_plist "$app_contents/Info.plist" "$short_ver" "$bundle_ver"

  cp -f "$out_bin" "$app_macos/rnparmor"
  chmod +x "$app_macos/rnparmor"
  cp -f "$dylib" "$app_macos/librnp.dylib"

  cp -f "$root/third_party/licenses/rnp-LICENSE.md" "$app_lic/"
  cp -f "$root/third_party/licenses/sexpp-LICENSE.md" "$app_lic/"
  cp -f "$deps/licenses/botan-license.txt" "$app_lic/"
  cp -f "$deps/licenses/json-c-COPYING" "$app_lic/"
  cp -f "$root/third_party/licenses/mygo-MIT.txt" "$app_lic/"
  cp -f "$root/third_party/licenses/purego-Apache-2.0.txt" "$app_lic/"

  run_cmd "codesign dylib" codesign --force --sign - "$app_macos/librnp.dylib"
  (
    cd "$app_macos"
    shasum -a 256 librnp.dylib > SHA256SUMS
  )
  run_cmd "codesign app" codesign --force --sign - "$app_root"
  run_cmd "codesign verify" codesign --verify --deep --strict "$app_root"

  run_cmd "go test with mac dylib" env RNPARMOR_LIBRNP="$app_macos/librnp.dylib" CGO_ENABLED=0 go test -count=1 -timeout 20m ./internal/loadlib/ ./internal/rnp/ ./cmd/rnparmor/

  local smoke_out
  smoke_out=$("$app_macos/rnparmor" --smoke-test)
  echo "$smoke_out"
  grep -Eq '^status=ok$' <<<"$smoke_out"
  grep -Eq "^app_version=${VERSION}$" <<<"$smoke_out"
  grep -Eq '^i18n=ok$' <<<"$smoke_out"

  local dmg_stage
  dmg_stage=$(mktemp -d -t rnparmor-dmg-stage.XXXXXX)
  cp -R "$app_root" "$dmg_stage/RnpArmor.app"
  ln -s /Applications "$dmg_stage/Applications"
  local dmg="$root/out/rnparmor-macos-arm64.dmg"
  rm -f "$dmg"
  run_cmd "hdiutil create" hdiutil create -volname "RnpArmor" -srcfolder "$dmg_stage" -ov -format UDZO "$dmg"
  [[ -s "$dmg" ]]
  echo "DMG_SHA256 $(sha256_file "$dmg")"
  rm -rf "$dmg_stage"
  emit_notice "macos-package" "done in $((SECONDS-t0))s"
}

run_stage() {
  case "$1" in
    deps) deps_stage ;;
    rnp) rnp_stage ;;
    package) package_stage ;;
    all)
      deps_stage
      rnp_stage
      package_stage
      ;;
  esac
}

run_stage "$stage"
