#!/usr/bin/env bash
set -euo pipefail

APP_PATH=""
INSTALL_DIRECTORY="/Applications"
CLI_PATH=""
EVIDENCE_PATH="native-smoke-evidence.json"
REQUIRE_SIGNATURE=0

usage() {
    cat <<'EOF'
Usage: packaging/macos/native-smoke.sh --app PATH [options]

Options:
  --app PATH                Upit.app to install and smoke
  --install-directory PATH  Installation directory (default: /Applications)
  --cli PATH                Existing upit CLI for continuity regression
  --evidence PATH            Evidence JSON path (default: native-smoke-evidence.json)
  --require-signature       Require codesign verification for the installed bundle
EOF
}

while (($# > 0)); do
    case "$1" in
        --app) APP_PATH="$2"; shift 2 ;;
        --install-directory) INSTALL_DIRECTORY="$2"; shift 2 ;;
        --cli) CLI_PATH="$2"; shift 2 ;;
        --evidence) EVIDENCE_PATH="$2"; shift 2 ;;
        --require-signature) REQUIRE_SIGNATURE=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
    echo "macOS 14+ Apple Silicon is required for the native smoke." >&2
    exit 1
fi
if [[ -z "$APP_PATH" || ! -d "$APP_PATH" ]]; then
    echo "--app must point to an existing Upit.app bundle." >&2
    exit 2
fi
if [[ -z "$CLI_PATH" && -x "$PWD/bin/upit-darwin-arm64" ]]; then
	CLI_PATH="$PWD/bin/upit-darwin-arm64"
fi
if [[ -z "$CLI_PATH" || ! -x "$CLI_PATH" ]]; then
	echo "--cli must point to the unchanged macOS CLI binary." >&2
	exit 2
fi
read_bool() {
    local prompt="$1"
    local answer
    read -r -p "$prompt [y/N] " answer
    [[ "$answer" =~ ^[Yy]([Ee][Ss])?$ ]]
}

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
validate_args=(--app "$APP_PATH")
if [[ "$REQUIRE_SIGNATURE" == 1 ]]; then
    validate_args+=(--require-signature)
fi
bash "$SCRIPT_DIRECTORY/validate.sh" "${validate_args[@]}"

work_directory="$(mktemp -d "${TMPDIR:-/tmp}/upit-macos-smoke.XXXXXX")"
config_directory="$HOME/.config/upit"
config_backup="$work_directory/config-backup"
config_was_present=0
server_pid=""
installed_app="$INSTALL_DIRECTORY/Upit.app"
if [[ -e "$installed_app" ]]; then
    echo "Native smoke requires a clean dedicated installation directory; refusing to replace $installed_app." >&2
    rm -rf "$work_directory"
    exit 1
fi

restore_environment() {
    if [[ -n "$server_pid" ]]; then
        kill "$server_pid" 2>/dev/null || true
        wait "$server_pid" 2>/dev/null || true
    fi
    if [[ "$config_was_present" == 1 ]]; then
        rm -rf "$config_directory"
        mkdir -p "$(dirname "$config_directory")"
        ditto "$config_backup" "$config_directory"
    else
        rm -rf "$config_directory"
    fi
    if [[ -e "$installed_app" ]]; then
        bash "$SCRIPT_DIRECTORY/uninstall.sh" --install-directory "$INSTALL_DIRECTORY" >/dev/null 2>&1 || true
    fi
    rm -rf "$work_directory"
}
trap restore_environment EXIT

if [[ -d "$config_directory" ]]; then
    config_was_present=1
    ditto "$config_directory" "$config_backup"
fi
rm -rf "$config_directory"
mkdir -p "$config_directory"

cat > "$work_directory/server.py" <<'PY'
import http.server
import pathlib
import sys

port_file = pathlib.Path(sys.argv[1])

class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        self.rfile.read(length)
        self.send_response(200)
        self.send_header("Content-Type", "text/plain")
        self.end_headers()
        self.wfile.write(f"http://127.0.0.1:{self.server.server_port}/result".encode())

    def log_message(self, format, *args):
        pass

server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY
python3 "$work_directory/server.py" "$work_directory/port" >/dev/null 2>&1 &
server_pid=$!
for _ in {1..50}; do
    [[ -s "$work_directory/port" ]] && break
    sleep 0.1
done
if [[ ! -s "$work_directory/port" ]]; then
    echo "Local smoke endpoint did not start." >&2
    exit 1
fi
port="$(<"$work_directory/port")"

cat > "$config_directory/config.json" <<EOF
{
  "version": 2,
  "defaultUploader": "smoke",
  "defaultShortener": "",
  "copyToClipboard": false
}
EOF
cat > "$config_directory/custom-uploader.json" <<EOF
{
  "version": 2,
  "uploaders": {
    "smoke": {
      "request": {"method": "POST", "url": "http://127.0.0.1:$port/upload", "body": "binary"},
      "response": {"url": {"type": "body"}}
    }
  }
}
EOF
chmod 600 "$config_directory"/*.json

fixture_directory="$work_directory/fixture"
mkdir -p "$fixture_directory"
printf 'macOS native smoke fixture\n' > "$fixture_directory/upit-smoke.txt"
mkdir -p "$INSTALL_DIRECTORY"
open "$APP_PATH"
open "$INSTALL_DIRECTORY"
if ! read_bool "Drag Upit.app from the mounted DMG into $INSTALL_DIRECTORY, then confirm the copy is complete"; then
    echo "Consumer drag installation was not completed." >&2
    exit 1
fi
installed_validate_args=(--app "$installed_app")
if [[ "$REQUIRE_SIGNATURE" == 1 ]]; then installed_validate_args+=(--require-signature); fi
bash "$SCRIPT_DIRECTORY/validate.sh" "${installed_validate_args[@]}"
open "$installed_app"
if ! read_bool "Confirm first launch only inspected registration, then explicitly Repair if needed; verify Registered does not claim Finder visibility"; then exit 1; fi
if ! read_bool "Deliberately unregister this test Service, use Desktop Repair to restore it, and verify Keyboard Settings guidance"; then exit 1; fi
if ! read_bool "Confirm only Upit is user-visible, with no separate helper application, then quit Desktop before Finder uploads"; then exit 1; fi
open "$fixture_directory"

if [[ -n "$CLI_PATH" ]]; then
    "$CLI_PATH" config validate >/dev/null
    "$CLI_PATH" upload --no-clipboard "$fixture_directory/upit-smoke.txt" >/dev/null
    cli_regression=true
else
    cli_regression=false
fi


discovery=false
selection_rejection=false
success=false
warning=false
failure=false
cancellation=false
recovery=false
privacy=false
no_window=false
helper_exit=false
uninstall_cleanup=false
always_copy=false
copy_recovery_without_reupload=false
atomic_update=false
manual_continuity=false
removal_prepared=false

if read_bool "Confirm Upload with Upit is visible in Finder Services or Quick Actions"; then discovery=true; fi
if read_bool "Confirm empty, multi-file, directory, and non-regular selections are rejected"; then selection_rejection=true; fi
if read_bool "Confirm one-file success completed against the local endpoint"; then success=true; fi
if [[ "$success" == true && "$(pbpaste)" == "http://127.0.0.1:$port/result" ]]; then always_copy=true; fi
if read_bool "Confirm a Shortener or clipboard warning remains a warning"; then warning=true; fi
if read_bool "Confirm a runtime failure shows sanitized native failure feedback"; then failure=true; fi
if read_bool "Confirm Cancel stops a slow upload without retry"; then cancellation=true; fi
if read_bool "Confirm Copy Final URL, Retry, Open Upit Desktop, and configuration recovery actions"; then recovery=true; fi
if read_bool "Confirm native feedback showed no path, file name, URL, endpoint, response, or credential"; then privacy=true; fi
if read_bool "Confirm no Upit Desktop window or Dock icon appeared during the Finder action"; then no_window=true; fi
if read_bool "Confirm the helper exited after completion or its action window"; then helper_exit=true; fi
if read_bool "Confirm Copy Final URL after automatic copy failure used the existing result without another endpoint request"; then copy_recovery_without_reupload=true; fi
if read_bool "Confirm a signed product update preserved one identity and did not duplicate Finder Services"; then atomic_update=true; fi
if read_bool "Confirm Manual Upload and CLI still work against the same Configuration Set"; then manual_continuity=true; fi

open "$installed_app"
if read_bool "Choose Prepare to Remove Upit in Desktop, explicitly confirm, and verify Desktop closes after unregistering the Service and outer app"; then removal_prepared=true; fi
if read_bool "Move this test Upit.app to Trash and confirm Upload with Upit disappears from Finder Services" && [[ ! -e "$installed_app" ]]; then uninstall_cleanup=true; fi

cat > "$EVIDENCE_PATH" <<EOF
{
  "platform": "macos",
  "minimumSystemVersion": "14.0",
  "architecture": "arm64",
  "signatureRequired": $([[ "$REQUIRE_SIGNATURE" == 1 ]] && echo true || echo false),
  "finderServiceDiscovery": $discovery,
  "selectionValidation": $selection_rejection,
  "success": $success,
  "warning": $warning,
  "failure": $failure,
  "cancellation": $cancellation,
  "recoveryActions": $recovery,
  "privacy": $privacy,
  "noDesktopWindowOrDockIcon": $no_window,
  "helperExit": $helper_exit,
  "cliRegression": $cli_regression,
  "consumerDragInstallAndExplicitRepair": true,
  "oneVisibleProduct": true,
  "alwaysCopyFinalURL": $always_copy,
  "copyRecoveryWithoutReupload": $copy_recovery_without_reupload,
  "atomicUpdate": $atomic_update,
  "manualUploadContinuity": $manual_continuity,
  "removalPrepared": $removal_prepared,
  "uninstallCleanup": $uninstall_cleanup
}
EOF

if [[ "$discovery" != true || "$selection_rejection" != true || "$success" != true || "$warning" != true || "$failure" != true || "$cancellation" != true || "$recovery" != true || "$privacy" != true || "$no_window" != true || "$helper_exit" != true || "$cli_regression" != true || "$uninstall_cleanup" != true || "$always_copy" != true || "$copy_recovery_without_reupload" != true || "$atomic_update" != true || "$manual_continuity" != true || "$removal_prepared" != true ]]; then
	echo "Native smoke is incomplete; see $EVIDENCE_PATH." >&2
	exit 1
fi

echo "macOS native smoke evidence written to $EVIDENCE_PATH"

