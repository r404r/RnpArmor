#!/usr/bin/env bash
# Build pinned Botan 3.10.0 and librnp 0.18.1 into out/lib.
# Does not download prebuilt binaries. Does not publish a release.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
pins=$root/deps/pins.json
src=$root/out/src
deps=$root/out/deps
libdir=$root/out/lib
mkdir -p "$src" "$deps" "$libdir"

python3 - "$pins" <<'PY'
import json, sys
pins = json.load(open(sys.argv[1]))
assert pins["rnp"]["tag"] == "v0.18.1"
assert pins["rnp"]["commit"] == "aad1892e4d8423398a0b973bd0a7e4544c359afc"
assert "v0.18.0" in pins["rnp"]["do_not_use"]
assert pins["botan"]["sha256"] == "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24"
print("pins ok", pins["rnp"]["tag"], pins["botan"]["version"])
PY

need_botan=1
if compgen -G "$deps/botan/lib/libbotan-3.so*" >/dev/null || compgen -G "$deps/botan/lib64/libbotan-3.so*" >/dev/null; then
  need_botan=0
fi

if [[ ! -f "$src/Botan-3.10.0/license.txt" ]]; then
  tarball=$src/Botan-3.10.0.tar.xz
  if [[ ! -f "$tarball" ]]; then
    curl -fsSL --retry 3 -o "$tarball" "https://botan.randombit.net/releases/Botan-3.10.0.tar.xz"
  fi
  echo "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24  $tarball" | sha256sum -c -
  sums=$src/botan-sha256sums.txt
  curl -fsSL --retry 3 -o "$sums" "https://botan.randombit.net/releases/sha256sums.txt"
  grep -F "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24  Botan-3.10.0.tar.xz" "$sums" >/dev/null
  rm -rf "$src/Botan-3.10.0"
  tar -C "$src" -xJf "$tarball"
fi

if [[ "$need_botan" == 1 ]]; then
  tarball=$src/Botan-3.10.0.tar.xz
  if [[ ! -f "$tarball" ]]; then
    curl -fsSL --retry 3 -o "$tarball" "https://botan.randombit.net/releases/Botan-3.10.0.tar.xz"
  fi
  echo "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24  $tarball" | sha256sum -c -
  sums=$src/botan-sha256sums.txt
  curl -fsSL --retry 3 -o "$sums" "https://botan.randombit.net/releases/sha256sums.txt"
  grep -F "fde194236f6d5434f136ea0a0627f6cc9d26af8b96e9f1e1c7d8c82cd90f4f24  Botan-3.10.0.tar.xz" "$sums" >/dev/null
  rm -rf "$src/Botan-3.10.0"
  tar -C "$src" -xJf "$tarball"
  (
    cd "$src/Botan-3.10.0"
    python3 ./configure.py \
      --prefix="$deps/botan" \
      --disable-static-library \
      --without-documentation \
      --build-targets=shared \
      --build-tool=make
    make -j"$(nproc)"
    make install
  )
fi

rnp_src=$src/rnp
if [[ ! -d "$rnp_src/.git" ]]; then
  git clone --depth 1 --branch v0.18.1 https://github.com/rnpgp/rnp.git "$rnp_src"
fi
got=$(git -C "$rnp_src" rev-parse HEAD)
if [[ "$got" != "aad1892e4d8423398a0b973bd0a7e4544c359afc" ]]; then
  echo "rnp HEAD $got does not match pin" >&2
  exit 1
fi
sexpp=$rnp_src/src/libsexpp
if [[ ! -d "$sexpp/.git" ]]; then
  rm -rf "$sexpp"
  git clone https://github.com/rnpgp/sexpp.git "$sexpp"
fi
git -C "$sexpp" fetch --depth 1 origin c641a2f36520bab783657a58650d9fda548b9dec
git -C "$sexpp" checkout --detach FETCH_HEAD
got_sexpp=$(git -C "$sexpp" rev-parse HEAD)
if [[ "$got_sexpp" != "c641a2f36520bab783657a58650d9fda548b9dec" ]]; then
  echo "sexpp HEAD $got_sexpp does not match pin" >&2
  exit 1
fi

json_c_src=$src/json-c-0.19
if [[ ! -e "$deps/json-c/lib/libjson-c.so" && ! -e "$deps/json-c/lib64/libjson-c.so" ]]; then
  jt=$src/json-c-0.19.tar.gz
  if [[ ! -f "$jt" ]]; then
    curl -fsSL --retry 3 -o "$jt" "https://github.com/json-c/json-c/releases/download/json-c-0.19-20260627/json-c-0.19.tar.gz"
  fi
  echo "37ad0249902e301bd9052bf712e511fcc6acff4ecaad4b5900aad9ce564e26de  $jt" | sha256sum -c -
  rm -rf "$json_c_src"
  tar -C "$src" -xzf "$jt"
  cmake -S "$json_c_src" -B "$src/json-c-build" \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_C_COMPILER=gcc \
    -DCMAKE_INSTALL_PREFIX="$deps/json-c" \
    -DBUILD_SHARED_LIBS=ON \
    -DBUILD_TESTING=OFF \
    -DDISABLE_WERROR=ON
  cmake --build "$src/json-c-build" -j"$(nproc)"
  cmake --install "$src/json-c-build"
fi

export PKG_CONFIG_PATH="$deps/json-c/lib/pkgconfig:$deps/json-c/lib64/pkgconfig:$deps/botan/lib/pkgconfig:$deps/botan/lib64/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
rm -rf "$src/rnp-build"
cmake -S "$rnp_src" -B "$src/rnp-build" \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_C_COMPILER=gcc \
  -DCMAKE_CXX_COMPILER=g++ \
  -DCMAKE_INSTALL_PREFIX="$deps/rnp" \
  -DCMAKE_PREFIX_PATH="$deps/json-c;$deps/botan" \
  -DCRYPTO_BACKEND=botan \
  -DBOTAN_ROOT_DIR="$deps/botan" \
  -DBUILD_SHARED_LIBS=ON \
  -DBUILD_TESTING=OFF \
  -DDOWNLOAD_GTEST=OFF \
  -DENABLE_DOC=OFF \
  -DENABLE_CRYPTO_REFRESH=OFF \
  -DENABLE_PQC=OFF \
  -DSYSTEM_LIBSEXPP=OFF \
  -DCMAKE_INSTALL_RPATH='$ORIGIN' \
  -DCMAKE_BUILD_WITH_INSTALL_RPATH=ON
cmake --build "$src/rnp-build" -j"$(nproc)"
cmake --install "$src/rnp-build"

shopt -s nullglob
candidates=("$deps/rnp/lib"/librnp.so "$deps/rnp/lib64"/librnp.so)
src_lib=""
for c in "${candidates[@]}"; do
  if [[ -e "$c" ]]; then
    src_lib=$c
    break
  fi
done
if [[ -z "$src_lib" ]]; then
  echo "librnp.so was not installed" >&2
  find "$deps/rnp" -name 'librnp.so*' -print >&2
  exit 1
fi

rm -rf "$libdir"
mkdir -p "$libdir"
cp -L "$src_lib" "$libdir/librnp.so"
# Bundle the SONAME files we built. ldd would otherwise report botan as
# missing and json-c as the distro copy, because RUNPATH is only $ORIGIN.
botan_soname="$deps/botan/lib/libbotan-3.so.10"
if [[ ! -e "$botan_soname" ]]; then
  botan_soname="$deps/botan/lib64/libbotan-3.so.10"
fi
json_soname="$deps/json-c/lib/libjson-c.so.5"
if [[ ! -e "$json_soname" ]]; then
  json_soname="$deps/json-c/lib64/libjson-c.so.5"
fi
cp -L "$botan_soname" "$libdir/libbotan-3.so.10"
cp -L "$json_soname" "$libdir/libjson-c.so.5"
if command -v patchelf >/dev/null 2>&1; then
  patchelf --set-rpath '$ORIGIN' "$libdir/librnp.so"
fi

env -u LD_LIBRARY_PATH ldd "$libdir/librnp.so" > "$libdir/ldd.txt"
python3 - "$libdir" "$libdir/ldd.txt" <<'PY'
import os, sys
libdir, path = sys.argv[1], sys.argv[2]
text = open(path).read()
need = {"libbotan-3.so.10": False, "libjson-c.so.5": False}
for line in text.splitlines():
    if "=>" not in line:
        continue
    parts = line.strip().split()
    name = parts[0]
    if "not found" in line:
        raise SystemExit("unresolved dependency: " + line.strip())
    if name in need:
        got = parts[2]
        if os.path.dirname(os.path.realpath(got)) != os.path.realpath(libdir):
            raise SystemExit(f"{name} resolved to {got}, not {libdir}")
        need[name] = True
missing = [k for k, ok in need.items() if not ok]
if missing:
    raise SystemExit("ldd did not mention " + ", ".join(missing))
print("bundled", " ".join(sorted(need)))
PY

(
  cd "$libdir"
  files=()
  while IFS= read -r extra; do
    files+=("$extra")
  done < <(find . -maxdepth 1 -type f ! -name 'ldd.txt' ! -name 'SHA256SUMS' -printf '%f\n' | sort)
  sha256sum "${files[@]}" > SHA256SUMS
)

mkdir -p "$root/out/licenses"
cp -f "$src/Botan-3.10.0/license.txt" "$root/out/licenses/botan-license.txt"
cp -f "$rnp_src/LICENSE.md" "$root/out/licenses/rnp-LICENSE.md"
if [[ -f "$json_c_src/COPYING" ]]; then
  cp -f "$json_c_src/COPYING" "$root/out/licenses/json-c-COPYING"
fi
if [[ -f "$sexpp/LICENSE" ]]; then
  cp -f "$sexpp/LICENSE" "$root/out/licenses/sexpp-LICENSE"
elif [[ -f "$sexpp/LICENSE.md" ]]; then
  cp -f "$sexpp/LICENSE.md" "$root/out/licenses/sexpp-LICENSE.md"
fi
cp -f "$libdir/ldd.txt" "$root/out/licenses/ldd.txt"
cp -f "$libdir/SHA256SUMS" "$root/out/licenses/SHA256SUMS"

echo "built $libdir/librnp.so"
echo "set RNPARMOR_LIBRNP=$libdir/librnp.so"
