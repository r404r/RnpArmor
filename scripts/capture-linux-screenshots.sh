#!/usr/bin/env bash
# Capture en/zh/ja RnpArmor main-window screenshots on a headless X11 desktop.
set -Eeuo pipefail
export LC_ALL=C

BIN=""
LIB=""
OUT=""

usage() {
  echo "usage: $0 --bin <path> --lib <path> --out <dir>" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bin)
      BIN="${2:-}"
      shift 2
      ;;
    --lib)
      LIB="${2:-}"
      shift 2
      ;;
    --out)
      OUT="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown arg: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$BIN" || -z "$LIB" || -z "$OUT" ]]; then
  usage
  exit 1
fi
if [[ ! -x "$BIN" ]]; then
  echo "bin is not executable: $BIN" >&2
  exit 1
fi
if [[ ! -f "$LIB" ]]; then
  echo "lib not found: $LIB" >&2
  exit 1
fi

BIN=$(readlink -f "$BIN")
LIB=$(readlink -f "$LIB")
mkdir -p "$OUT"
OUT=$(readlink -f "$OUT")

work=$(mktemp -d "${TMPDIR:-/tmp}/rnparmor-shot.XXXXXX")
xdg=$(mktemp -d "${TMPDIR:-/tmp}/rnparmor-xdg.XXXXXX")
chmod 700 "$xdg"
export XDG_RUNTIME_DIR="$xdg"
export DISPLAY=:99
export GDK_BACKEND=x11
export LIBGL_ALWAYS_SOFTWARE=1
export RNPARMOR_LIBRNP="$LIB"

CURRENT_CMD="init"
LAST_LOG=""
FAIL_DETAIL=""
XVFB_PID=""
OPENBOX_PID=""
APP_PID=""

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
  echo "$msg"
}

log_tail() {
  local text=""
  if [[ -n "$LAST_LOG" && -f "$LAST_LOG" ]]; then
    text=$(tail -n 15 "$LAST_LOG" | tr '\n' ' ' | sed 's/  */ /g')
  fi
  printf '%s' "${text:-none}"
}

on_err() {
  local ec=$?
  trap - ERR
  set +e
  local tail_text
  tail_text=$(log_tail)
  echo "command '${CURRENT_CMD:-unknown}' failed exit=${ec}" >&2
  emit_error "linux-screenshot" "command '${CURRENT_CMD:-unknown}' failed exit=${ec} detail='${FAIL_DETAIL:-none}' log_tail=${tail_text}"
  exit "$ec"
}

stop_pid() {
  local pid="$1"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    local _
    for _ in $(seq 1 30); do
      if ! kill -0 "$pid" 2>/dev/null; then
        break
      fi
      sleep 0.1
    done
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
  fi
  pkill -P "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
}

stop_app() {
  stop_pid "${APP_PID:-}"
  APP_PID=""
  if pgrep -x rnparmor >/dev/null 2>&1; then
    pkill -x rnparmor >/dev/null 2>&1 || true
    sleep 0.2
    pkill -9 -x rnparmor >/dev/null 2>&1 || true
  fi
}

cleanup() {
  local ec=$?
  trap - ERR
  set +e
  stop_app
  stop_pid "${OPENBOX_PID:-}"
  OPENBOX_PID=""
  stop_pid "${XVFB_PID:-}"
  XVFB_PID=""
  if [[ -n "${work:-}" && -d "${work:-}" ]]; then
    rm -rf "$work"
  fi
  if [[ -n "${xdg:-}" && -d "${xdg:-}" ]]; then
    rm -rf "$xdg"
  fi
  exit "$ec"
}

trap on_err ERR
trap cleanup EXIT

require_cmd() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    CURRENT_CMD="$name"
    FAIL_DETAIL="required command not found: $name"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
}

require_cmd Xvfb
require_cmd openbox
require_cmd scrot
require_cmd xdotool
require_cmd python3
if ! command -v identify >/dev/null 2>&1; then
  echo "identify not found; screenshot metrics will use python3" >&2
fi

first_id() {
  awk 'NF { print; exit }'
}

search_window() {
  local pid="$1"
  local ids=""
  ids=$(xdotool search --onlyvisible --name "RnpArmor" 2>/dev/null || true)
  if [[ -z "${ids//[$' \t\n']/}" ]]; then
    ids=$(xdotool search --onlyvisible --pid "$pid" 2>/dev/null || true)
  fi
  if [[ -z "${ids//[$' \t\n']/}" ]]; then
    ids=$(xdotool search --onlyvisible --name ".*" 2>/dev/null || true)
  fi
  printf '%s\n' "$ids" | first_id || true
}

png_metrics() {
  python3 - "$1" <<'PY_PNG'
import math
import struct
import sys
import zlib

path = sys.argv[1]
data = open(path, "rb").read()
if not data.startswith(b"\x89PNG\r\n\x1a\n"):
    raise SystemExit("not a png")
pos = 8
width = height = bit_depth = color_type = interlace = None
idat = bytearray()
plte = None
while pos + 8 <= len(data):
    length = struct.unpack(">I", data[pos:pos + 4])[0]
    typ = data[pos + 4:pos + 8]
    chunk = data[pos + 8:pos + 8 + length]
    pos += 12 + length
    if typ == b"IHDR":
        width, height, bit_depth, color_type, _comp, _filt, interlace = struct.unpack(">IIBBBBB", chunk[:13])
    elif typ == b"PLTE":
        plte = chunk
    elif typ == b"IDAT":
        idat += chunk
    elif typ == b"IEND":
        break
if width is None or not idat:
    raise SystemExit("png missing IHDR or IDAT")
if interlace:
    raise SystemExit("interlaced png is not supported")
if bit_depth != 8:
    raise SystemExit("unsupported bit depth")
channels = {0: 1, 2: 3, 3: 1, 4: 2, 6: 4}.get(color_type)
if channels is None:
    raise SystemExit("unsupported color type")
bpp = channels
raw = zlib.decompress(bytes(idat))
stride = width * bpp
if len(raw) < (stride + 1) * height:
    raise SystemExit("png data short")

def paeth(a, b, c):
    p = a + b - c
    pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
    if pa <= pb and pa <= pc:
        return a
    if pb <= pc:
        return b
    return c

rows = []
prev = bytearray(stride)
i = 0
for _y in range(height):
    filt = raw[i]
    i += 1
    row = bytearray(raw[i:i + stride])
    i += stride
    if filt == 1:
        for x in range(stride):
            left = row[x - bpp] if x >= bpp else 0
            row[x] = (row[x] + left) & 255
    elif filt == 2:
        for x in range(stride):
            row[x] = (row[x] + prev[x]) & 255
    elif filt == 3:
        for x in range(stride):
            left = row[x - bpp] if x >= bpp else 0
            row[x] = (row[x] + ((left + prev[x]) // 2)) & 255
    elif filt == 4:
        for x in range(stride):
            a = row[x - bpp] if x >= bpp else 0
            b = prev[x]
            c = prev[x - bpp] if x >= bpp else 0
            row[x] = (row[x] + paeth(a, b, c)) & 255
    elif filt != 0:
        raise SystemExit("bad png filter")
    rows.append(row)
    prev = row

count = 0
mean = 0.0
m2 = 0.0
for row in rows:
    x = 0
    while x < stride:
        if color_type in (0, 4):
            luma = row[x] / 255.0
        elif color_type == 3:
            if plte is None:
                raise SystemExit("indexed png missing PLTE")
            idx = row[x] * 3
            r, g, b = plte[idx], plte[idx + 1], plte[idx + 2]
            luma = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255.0
        else:
            r, g, b = row[x], row[x + 1], row[x + 2]
            luma = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255.0
        count += 1
        delta = luma - mean
        mean += delta / count
        m2 += delta * (luma - mean)
        x += bpp
if count == 0:
    raise SystemExit("no pixels")
std = math.sqrt(m2 / count)
print(width, height, f"{std:.6f}")
PY_PNG
}

start_desktop() {
  CURRENT_CMD="Xvfb"
  LAST_LOG="$work/xvfb.log"
  Xvfb :99 -screen 0 1280x900x24 -nolisten tcp >"$work/xvfb.log" 2>&1 &
  XVFB_PID=$!
  local _
  local ready=0
  for _ in $(seq 1 100); do
    if ! kill -0 "$XVFB_PID" 2>/dev/null; then
      FAIL_DETAIL="Xvfb exited before the display was ready"
      echo "$FAIL_DETAIL" >&2
      exit 1
    fi
    if xdotool getdisplaygeometry >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "$ready" -ne 1 ]]; then
    FAIL_DETAIL="Xvfb :99 did not become ready"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi

  CURRENT_CMD="openbox"
  LAST_LOG="$work/openbox.log"
  openbox >"$work/openbox.log" 2>&1 &
  OPENBOX_PID=$!
  ready=0
  for _ in $(seq 1 30); do
    if kill -0 "$OPENBOX_PID" 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "$ready" -ne 1 ]]; then
    FAIL_DETAIL="openbox exited before it was ready"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
  sleep 0.5
  if ! kill -0 "$OPENBOX_PID" 2>/dev/null; then
    FAIL_DETAIL="openbox exited during startup"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
}

capture_lang() {
  local lang="$1"
  local png="$OUT/rnparmor-linux-$lang.png"
  local log="$work/rnparmor-$lang.log"
  local wid=""
  local _

  stop_app
  CURRENT_CMD="rnparmor-$lang"
  LAST_LOG="$log"
  RNPARMOR_LIBRNP="$LIB" "$BIN" --lang "$lang" >"$log" 2>&1 &
  APP_PID=$!

  CURRENT_CMD="find-window-$lang"
  for _ in $(seq 1 20); do
    if ! kill -0 "$APP_PID" 2>/dev/null; then
      FAIL_DETAIL="rnparmor exited before the window appeared (lang=$lang)"
      echo "$FAIL_DETAIL" >&2
      exit 1
    fi
    wid=$(search_window "$APP_PID" || true)
    if [[ -n "$wid" ]]; then
      break
    fi
    sleep 0.5
  done
  if [[ -z "$wid" ]]; then
    FAIL_DETAIL="main window not found for lang=$lang"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi

  CURRENT_CMD="place-window-$lang"
  if ! timeout 8 xdotool windowmove --sync "$wid" 40 40; then
    xdotool windowmove "$wid" 40 40
  fi
  if ! timeout 8 xdotool windowactivate --sync "$wid"; then
    xdotool windowactivate "$wid" || true
  fi
  if ! timeout 8 xdotool windowfocus --sync "$wid"; then
    xdotool windowfocus "$wid" || echo "windowfocus returned non-zero for $wid" >&2
  fi
  sleep 2.5

  rm -f "$png"
  CURRENT_CMD="scrot-focused-$lang"
  if ! scrot --focused --border "$png"; then
    CURRENT_CMD="scrot-$lang"
    scrot "$png"
  fi

  CURRENT_CMD="stop-rnparmor-$lang"
  stop_app
}

measure_png() {
  local png="$1"
  local lang="$2"
  local bytes meta w h std
  CURRENT_CMD="validate-$lang"
  LAST_LOG="$work/rnparmor-$lang.log"
  if [[ ! -f "$png" ]]; then
    FAIL_DETAIL="missing screenshot: $png"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
  bytes=$(stat -c '%s' "$png")
  if [[ "$bytes" -le 10000 ]]; then
    FAIL_DETAIL="screenshot too small: $png bytes=$bytes"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
  if command -v identify >/dev/null 2>&1 && meta=$(identify -format '%w %h %[fx:standard_deviation]' "$png" 2>"$work/identify-$lang.err"); then
    :
  else
    meta=$(png_metrics "$png")
  fi
  read -r w h std <<<"${meta:-}" || true
  if [[ -z "${w:-}" || -z "${h:-}" || -z "${std:-}" ]]; then
    meta=$(png_metrics "$png")
    read -r w h std <<<"${meta:-}" || true
  fi
  if [[ -z "${w:-}" || -z "${h:-}" || -z "${std:-}" ]]; then
    FAIL_DETAIL="could not read screenshot metrics: $png"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
  if ! python3 -c 'import sys; raw=sys.argv[1].strip().split()[0].rstrip("%"); v=float(raw); raise SystemExit(0 if v > 0.01 else 1)' "$std"; then
    FAIL_DETAIL="screenshot looks blank: $png ${w}x${h} bytes=$bytes stddev=$std"
    echo "$FAIL_DETAIL" >&2
    exit 1
  fi
  emit_notice "linux-screenshots" "rnparmor-linux-${lang}.png ${w}x${h} bytes=${bytes} stddev=${std}"
}

start_desktop
for lang in en zh ja; do
  capture_lang "$lang"
  measure_png "$OUT/rnparmor-linux-$lang.png" "$lang"
done
emit_notice "linux-screenshots" "captured en zh ja into $OUT"
