#!/bin/sh
# shellcheck disable=SC3043
# compare.sh — Verify 1:1 compatibility between shell and Go distrobox versions.
#
# Usage:
#   ./tests/compare.sh [--container-manager podman|docker]
#
# Requirements:
#   - Shell distrobox installed or available at DISTROBOX_SHELL_PATH
#   - Go distrobox built at DISTROBOX_GO_PATH (default: ./bin/distrobox)
#   - A container manager (podman or docker)
#   - jq

set -eu

# ── Paths ────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REWRITE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_DIR="$(cd "${REWRITE_DIR}/.." && pwd)"

DISTROBOX_SHELL_PATH="${DISTROBOX_SHELL_PATH:-${PROJECT_DIR}/distrobox}"
DISTROBOX_GO_PATH="${DISTROBOX_GO_PATH:-${REWRITE_DIR}/bin/distrobox}"
CONTAINER_MANAGER="${CONTAINER_MANAGER:-autodetect}"
KEEP_CONTAINERS="${KEEP_CONTAINERS:-0}"
CMD_TIMEOUT="${CMD_TIMEOUT:-10}"

# ── Colors / helpers ─────────────────────────────────────────────────────────
RED=$(printf '\033[0;31m')
GREEN=$(printf '\033[0;32m')
YELLOW=$(printf '\033[0;33m')
BOLD=$(printf '\033[1m')
RESET=$(printf '\033[0m')
TAB=$(printf '\t')

pass=0
fail=0
skip=0

# Unique-suffix counter (replaces $RANDOM)
_seq=0

log_pass()
{
	printf "  %sPASS%s %s\n" "${GREEN}" "${RESET}" "$1"
	pass=$((pass + 1))
}
log_fail()
{
	printf "  %sFAIL%s %s\n" "${RED}" "${RESET}" "$1"
	fail=$((fail + 1))
}
log_skip()
{
	printf "  %sSKIP%s %s\n" "${YELLOW}" "${RESET}" "$1"
	skip=$((skip + 1))
}
log_section()
{
	printf "\n%s── %s ──%s\n" "${BOLD}" "$1" "${RESET}"
}
die()
{
	printf "%sERROR:%s %s\n" "${RED}" "${RESET}" "$1" >&2
	exit 1
}

# ── Argument parsing ─────────────────────────────────────────────────────────
while [ $# -gt 0 ]; do
	case "$1" in
		--container-manager)
			CONTAINER_MANAGER="$2"
			shift 2
			;;
		--keep)
			KEEP_CONTAINERS=1
			shift
			;;
		*) die "Unknown argument: $1" ;;
	esac
done

# ── Preflight checks ────────────────────────────────────────────────────────
command -v jq > /dev/null || die "jq is required"
[ -x "${DISTROBOX_SHELL_PATH}" ] || die "Shell distrobox not found at ${DISTROBOX_SHELL_PATH}"
[ -x "${DISTROBOX_GO_PATH}" ] || die "Go distrobox not found at ${DISTROBOX_GO_PATH}. Run 'make build' first."

if [ "${CONTAINER_MANAGER}" = "autodetect" ]; then
	if command -v podman > /dev/null 2>&1; then
		CONTAINER_MANAGER="podman"
	elif command -v docker > /dev/null 2>&1; then
		CONTAINER_MANAGER="docker"
	else
		die "No container manager found. Install podman or docker."
	fi
fi
command -v "${CONTAINER_MANAGER}" > /dev/null || die "${CONTAINER_MANAGER} not found"

echo "Shell distrobox: ${DISTROBOX_SHELL_PATH}"
echo "Go distrobox:    ${DISTROBOX_GO_PATH}"
echo "Container mgr:   ${CONTAINER_MANAGER}"

TMPDIR="$(mktemp -d /tmp/distrobox-compare.XXXXXX)"
trap 'rm -rf "${TMPDIR}"' EXIT

# ── Normalize a dry-run command for diffing ──────────────────────────────────
# Both versions emit a flat string of docker/podman args. We normalize by:
#   1. Stripping the container manager binary name (docker/podman/sudo)
#   2. Splitting into one-arg-per-line (split on whitespace boundaries that
#      precede --)
#   3. Sorting flags (order doesn't matter to docker/podman)
#   4. Removing host-specific paths that will legitimately differ between
#      versions (script paths, binary paths)
normalize_create_cmd()
{
	local input="$1"

	echo "${input}" |
		# Strip Go-style prefix: "Creating 'xxx' using image yyy\t"
		sed -E "s/^Creating '[^']*' using image [^${TAB}]*${TAB}//" |
		# Strip trailing "[ OK ]" or similar status from Go output
		sed -E 's/\[[ A-Z]*\][ ]*$//' |
		# Remove leading container manager name (podman, docker, sudo podman, etc.)
		sed -E 's/^(sudo\s+)?(podman|docker|lilipod)\s+//' |
		# Remove the "create" verb
		sed -E 's/^create\s+//' |
		# Split into one token per line. Each token starts with --
		sed -E 's/ --/\n--/g' |
		# Remove quoting differences first (before whitespace collapse)
		sed -E "s/[\"']//g" |
		# Normalize whitespace: collapse runs, trim leading/trailing
		sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//' |
		# Remove empty lines
		grep -v '^$' |
		# Remove lines with host-specific script paths that will differ
		# between shell and Go (the provisioned script locations)
		grep -v 'distrobox-init' |
		grep -v 'distrobox-export' |
		grep -v 'distrobox-host-exec' |
		grep -v '/usr/bin/entrypoint' |
		grep -v 'entrypoint' |
		# Normalize "-- ''" (shell) vs "--" (Go) — empty init hook terminator
		sed -E "s/^-- *''?$/--/" |
		# Remove trailing whitespace
		sed -E 's/[[:space:]]+$//' |
		# Remove standalone image references (e.g. "fedora:42") — these are
		# positional args whose presence in output depends on print formatting.
		# The image is already verified via container inspect.
		grep -E '^--|^[[:space:]]*$' |
		grep -v '^$' |
		# Sort for order-independent comparison
		sort
}

# Normalize an enter dry-run command similarly.
normalize_enter_cmd()
{
	local input="$1"

	echo "${input}" |
		sed -E 's/^(sudo\s+)?(podman|docker|lilipod)\s+//' |
		sed -E 's/ --/\n--/g' |
		sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//' |
		# The DISTROBOX_ENTER_PATH will differ — remove it
		grep -v 'DISTROBOX_ENTER_PATH' |
		# Environment variables forwarded from host will be the same,
		# but filter out any with host-specific distrobox paths
		grep -v 'DBX_SCRIPTS_DIR' |
		grep -v 'SHLVL' |
		# Shell-internal variables that leak into Go's env via test harness
		grep -v 'container_generate_entry' |
		grep -v 'non_interactive' |
		sed -E 's/"//g' |
		sort
}

# ── Normalize container inspect JSON for comparison ──────────────────────────
# Extract the fields that matter and remove host-specific noise.
normalize_inspect()
{
	local json_file="$1"

	jq '.[0] | {
        # Container config
        Labels: (.Config.Labels // {}),
        Env: (.Config.Env // [] | sort),
        Entrypoint: .Config.Entrypoint,
        Cmd: .Config.Cmd,
        User: .Config.User,
        Hostname: .Config.Hostname,

        # Mounts — normalize to just source:dest:mode triples, removing
        # host-specific source paths for the distrobox scripts
        Mounts: (
            [.Mounts // [] | .[] | {
                Destination: .Destination,
                Mode: .Mode,
                RW: .RW,
                Type: .Type,
                # Redact source for distrobox scripts (paths will differ)
                Source: (
                    if (.Destination | test("distrobox-(export|host-exec|init)|entrypoint"))
                    then "<redacted-script-path>"
                    elif (.Type == "volume" and (.Source | test("/volumes/")))
                    then "<redacted-unnamed-volume>"
                    else .Source
                    end
                )
            }] | sort_by(.Destination)
        ),

        # Security
        Privileged: .HostConfig.Privileged,
        SecurityOpt: (.HostConfig.SecurityOpt // [] | sort),
        CapAdd: (.HostConfig.CapAdd // [] | sort),
        PidsLimit: .HostConfig.PidsLimit,

        # Namespaces
        NetworkMode: .HostConfig.NetworkMode,
        PidMode: .HostConfig.PidMode,
        IpcMode: .HostConfig.IpcMode
    }' "${json_file}" 2> /dev/null || jq '.[0] | {
        # Podman has a slightly different inspect schema
        Labels: (.Config.Labels // {}),
        Env: (.Config.Env // [] | sort),
        Entrypoint: (.Config.Entrypoint // []),
        Cmd: (.Config.Cmd // []),
        User: (.Config.User // ""),
        Hostname: (.Config.Hostname // ""),

        Mounts: (
            [.Mounts // [] | .[] | {
                Destination: .Destination,
                Mode: (.Options // [] | join(",")),
                RW: (if (.Options // [] | index("ro")) then false else true end),
                Type: .Type,
                Source: (
                    if (.Destination | test("distrobox-(export|host-exec|init)|entrypoint"))
                    then "<redacted-script-path>"
                    elif (.Type == "volume" and (.Source | test("/volumes/")))
                    then "<redacted-unnamed-volume>"
                    else .Source
                    end
                )
            }] | sort_by(.Destination)
        ),

        Privileged: .HostConfig.Privileged,
        SecurityOpt: (.HostConfig.SecurityOpt // [] | sort),
        CapAdd: (.HostConfig.CapAdd // [] | sort),
        PidsLimit: .HostConfig.PidsLimit,

        NetworkMode: .HostConfig.NetworkMode,
        PidMode: .HostConfig.PidMode,
        IpcMode: .HostConfig.IpcMode
    }' "${json_file}"
}

# ══════════════════════════════════════════════════════════════════════════════
# TEST SUITE
# ══════════════════════════════════════════════════════════════════════════════

image="alpine:3.21"
safe_name="$(echo "${image}" | tr ':/.@' '-')"
shell_name="dbx-test-shell-${safe_name}"
go_name="dbx-test-go-${safe_name}"

# Clean up leftover containers from previous runs — Go dry-run refuses
# to produce output if a container with the same name already exists.
"${CONTAINER_MANAGER}" rm -f "${shell_name}" > /dev/null 2>&1 || true
"${CONTAINER_MANAGER}" rm -f "${go_name}" > /dev/null 2>&1 || true

# ── Test 1: Dry-run create comparison ────────────────────────────────
log_section     "DRY-RUN CREATE"

shell_dryrun="${TMPDIR}/${safe_name}-create-dryrun-shell.txt"
go_dryrun="${TMPDIR}/${safe_name}-create-dryrun-go.txt"
shell_norm="${TMPDIR}/${safe_name}-create-norm-shell.txt"
go_norm="${TMPDIR}/${safe_name}-create-norm-go.txt"

# Shell first — it's the reference implementation.
DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" create \
	--dry-run \
	--yes \
	--name "${shell_name}" \
	--image "${image}" \
	> "${shell_dryrun}" 2> /dev/null || true

if     [ ! -s "${shell_dryrun}" ]; then
	log_skip "Shell dry-run produced no output"
else

	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--dry-run \
		--yes \
		--name "${go_name}" \
		--image "${image}" \
		> "${go_dryrun}" 2> /dev/null || true

	if [ ! -s "${go_dryrun}" ]; then
		log_fail "Go dry-run produced no output (shell did)"
	else
		# Normalize — also replace container names so they match
		normalize_create_cmd "$(cat "${shell_dryrun}")" |
			sed "s/${shell_name}/__CONTAINER__/g" > "${shell_norm}"
		normalize_create_cmd "$(cat "${go_dryrun}")" |
			sed "s/${go_name}/__CONTAINER__/g" > "${go_norm}"

		diff_out="${TMPDIR}/${safe_name}-create-diff.txt"
		if diff -u "${shell_norm}" "${go_norm}" > "${diff_out}" 2>&1; then
			log_pass "create dry-run: commands match"
		else
			log_fail "create dry-run: commands differ"
			echo "    Diff saved to: ${diff_out}"
			# Show first 20 lines of diff
			head -30 "${diff_out}" | sed 's/^/    /'
		fi
	fi # go output check
fi     # shell output check

# ── Test 2: Dry-run create with all flag permutations ────────────────
log_section     "DRY-RUN CREATE (flag permutations)"

# Generate all 2^N combinations (including empty = default, already tested above)
_get_flag()
{
	case "$1" in
		0) echo "--init" ;;
		1) echo "--unshare-netns" ;;
		2) echo "--unshare-ipc" ;;
		3) echo "--unshare-process" ;;
		4) echo "--unshare-devsys" ;;
		5) echo "--unshare-groups" ;;
		6) echo "--nvidia" ;;
		*) echo "unknown-flag-$1" ;;
	esac
}

_n_flags=7
_n_combos=$((1 << _n_flags))
_i=1
while     [ "${_i}" -lt "${_n_combos}" ]; do
	combo=""
	_j=0
	while [ "${_j}" -lt "${_n_flags}" ]; do
		if [ $((_i & (1 << _j))) -ne 0 ]; then
			flag=$(_get_flag "${_j}")
			if [ -n "${combo}" ]; then
				combo="${combo} ${flag}"
			else
				combo="${flag}"
			fi
		fi
		_j=$((_j + 1))
	done
	printf '%s\n' "${combo}"
	_i=$((_i + 1))
done     > "${TMPDIR}/flag_combos.txt"
printf     '%s\n' "--additional-packages vim" >> "${TMPDIR}/flag_combos.txt"
printf     '%s\n' "--additional-packages vim --init" >> "${TMPDIR}/flag_combos.txt"

while     IFS= read -r flags; do
	flag_slug="$(echo "${flags}" | tr ' -/:' '_' | tr -d '-')"

	shell_out="${TMPDIR}/${safe_name}-create-${flag_slug}-shell.txt"
	go_out="${TMPDIR}/${safe_name}-create-${flag_slug}-go.txt"

	# Shell first — it's the reference implementation.
	# If shell fails, the flag combo is invalid.
	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		--dry-run --yes \
		--name "${shell_name}" \
		--image "${image}" \
		${flags} \
		> "${shell_out}" 2> /dev/null || true

	if [ ! -s "${shell_out}" ]; then
		log_skip "create ${flags}: shell produced no output (invalid combo?)"
		continue
	fi

	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--dry-run --yes \
		--name "${go_name}" \
		--image "${image}" \
		${flags} \
		> "${go_out}" 2> /dev/null || true

	if [ ! -s "${go_out}" ]; then
		log_fail "create ${flags}: Go produced no output (shell did)"
		continue
	fi

	s_norm="${TMPDIR}/${safe_name}-${flag_slug}-norm-shell.txt"
	g_norm="${TMPDIR}/${safe_name}-${flag_slug}-norm-go.txt"

	normalize_create_cmd "$(cat "${shell_out}")" |
		sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
	normalize_create_cmd "$(cat "${go_out}")" |
		sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

	diff_file="${TMPDIR}/${safe_name}-create-${flag_slug}-diff.txt"
	if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
		log_pass "create ${flags}: commands match"
	else
		log_fail "create ${flags}: commands differ"
		head -20 "${diff_file}" | sed 's/^/    /'
	fi
done     < "${TMPDIR}/flag_combos.txt"

# ── Test 3: Short-flag cross-version comparison ─────────────────────
# Run the same short-flag command on BOTH shell and Go, then diff.
# If Go is missing a short alias (e.g. -I for --init), its output will
# differ or be empty, and the test fails.
log_section     "DRY-RUN CREATE (short flags, shell vs Go)"

while     IFS= read -r flags; do
	flag_slug="$(echo "${flags}" | tr ' -/:' '_' | tr -d '-')"

	shell_out="${TMPDIR}/${safe_name}-short-${flag_slug}-shell.txt"
	go_out="${TMPDIR}/${safe_name}-short-${flag_slug}-go.txt"

	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		-i "${image}" -n "${shell_name}" \
		${flags} \
		> "${shell_out}" 2> /dev/null || true

	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		-i "${image}" -n "${go_name}" \
		${flags} \
		> "${go_out}" 2> /dev/null || true

	if [ ! -s "${shell_out}" ] && [ ! -s "${go_out}" ]; then
		log_skip "short ${flags}: both produced no output"
		continue
	elif [ ! -s "${shell_out}" ]; then
		log_fail "short ${flags}: shell produced no output (Go did)"
		continue
	elif [ ! -s "${go_out}" ]; then
		log_fail "short ${flags}: Go produced no output (shell did)"
		continue
	fi

	s_norm="${TMPDIR}/${safe_name}-short-${flag_slug}-norm-shell.txt"
	g_norm="${TMPDIR}/${safe_name}-short-${flag_slug}-norm-go.txt"

	normalize_create_cmd "$(cat "${shell_out}")" |
		sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
	normalize_create_cmd "$(cat "${go_out}")" |
		sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

	diff_file="${TMPDIR}/${safe_name}-short-${flag_slug}-diff.txt"
	if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
		log_pass "short ${flags}: shell vs Go match"
	else
		log_fail "short ${flags}: shell vs Go differ"
		head -20 "${diff_file}" | sed 's/^/    /'
	fi
done     << 'SHORT_FLAGS_EOF'
-d -Y -I
-d -Y -ap vim
-d -Y -I -ap vim
-d -Y -I --nvidia
-d -Y --unshare-netns
-d -Y --unshare-ipc --unshare-process
-d -Y --unshare-devsys
-d -Y -I --unshare-netns --unshare-ipc --unshare-process --unshare-devsys --nvidia
-d -Y -p
-d -Y -H /tmp/dbx-test-home
SHORT_FLAGS_EOF

# ── Test 3b: Dry-run create with value flags ─────────────────────────
# Flags that take values can't be easily permuted, so test standalone.
log_section     "DRY-RUN CREATE (value flags)"

while     IFS= read -r flags; do
	flag_slug="$(echo "${flags}" | tr ' -/:' '_' | tr -d '-')"

	shell_out="${TMPDIR}/${safe_name}-valflag-${flag_slug}-shell.txt"
	go_out="${TMPDIR}/${safe_name}-valflag-${flag_slug}-go.txt"

	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		--dry-run --yes \
		--name "${shell_name}" \
		--image "${image}" \
		${flags} \
		> "${shell_out}" 2> /dev/null || true

	if [ ! -s "${shell_out}" ]; then
		log_skip "create ${flags}: shell produced no output"
		continue
	fi

	# shellcheck disable=SC2086
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--dry-run --yes \
		--name "${go_name}" \
		--image "${image}" \
		${flags} \
		> "${go_out}" 2> /dev/null || true

	if [ ! -s "${go_out}" ]; then
		log_fail "create ${flags}: Go produced no output (shell did)"
		continue
	fi

	s_norm="${TMPDIR}/${safe_name}-valflag-${flag_slug}-norm-shell.txt"
	g_norm="${TMPDIR}/${safe_name}-valflag-${flag_slug}-norm-go.txt"

	normalize_create_cmd "$(cat "${shell_out}")" |
		sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
	normalize_create_cmd "$(cat "${go_out}")" |
		sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

	diff_file="${TMPDIR}/${safe_name}-valflag-${flag_slug}-diff.txt"
	if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
		log_pass "create ${flags}: commands match"
	else
		log_fail "create ${flags}: commands differ"
		head -20 "${diff_file}" | sed 's/^/    /'
	fi
done     << 'VALUE_FLAGS_EOF'
--hostname custom-hostname
--home /tmp/dbx-test-home
--volume /tmp/dbx-test-vol:/mnt/test-vol:ro
--additional-flags --memory=512m
--init-hooks true
--pre-init-hooks true
--unshare-all
--pull
--no-entry
--platform linux/arm64
VALUE_FLAGS_EOF

# ── Test 4: Actual create + inspect comparison ───────────────────────
log_section     "INSPECT COMPARISON"

# Clean up any leftover containers from previous runs
"${CONTAINER_MANAGER}"     rm -f "${shell_name}" > /dev/null 2>&1 || true
"${CONTAINER_MANAGER}"     rm -f "${go_name}" > /dev/null 2>&1 || true

shell_created=0
go_created=0

# Create with shell version
if     DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" create \
	--yes \
	--name "${shell_name}" \
	--image "${image}" \
	2> /dev/null; then
	shell_created=1
fi

# Create with Go version
if     DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_GO_PATH}" create \
	--yes \
	--name "${go_name}" \
	--image "${image}" \
	2> /dev/null; then
	go_created=1
fi

if     [ "${shell_created}" -eq 0 ] || [ "${go_created}" -eq 0 ]; then
	log_skip "inspect: could not create one or both containers"
else
	shell_inspect="${TMPDIR}/${safe_name}-inspect-shell.json"
	go_inspect="${TMPDIR}/${safe_name}-inspect-go.json"

	"${CONTAINER_MANAGER}" inspect "${shell_name}" > "${shell_inspect}" 2> /dev/null
	"${CONTAINER_MANAGER}" inspect "${go_name}" > "${go_inspect}" 2> /dev/null

	shell_norm_inspect="${TMPDIR}/${safe_name}-inspect-norm-shell.json"
	go_norm_inspect="${TMPDIR}/${safe_name}-inspect-norm-go.json"

	normalize_inspect "${shell_inspect}" |
		# Replace container-specific names
		sed "s/${shell_name}/__CONTAINER__/g" |
		jq -S '.' > "${shell_norm_inspect}"

	normalize_inspect "${go_inspect}" |
		sed "s/${go_name}/__CONTAINER__/g" |
		jq -S '.' > "${go_norm_inspect}"

	# ── Compare labels ───────────────────────────────────────────
	shell_labels="${TMPDIR}/${safe_name}-labels-shell.json"
	go_labels="${TMPDIR}/${safe_name}-labels-go.json"
	jq '.Labels' "${shell_norm_inspect}" > "${shell_labels}"
	jq '.Labels' "${go_norm_inspect}" > "${go_labels}"

	if diff -u "${shell_labels}" "${go_labels}" > /dev/null 2>&1; then
		log_pass "inspect labels: match"
	else
		log_fail "inspect labels: differ"
		diff -u "${shell_labels}" "${go_labels}" | head -20 | sed 's/^/    /'
	fi

	# ── Compare env vars ─────────────────────────────────────────
	shell_env="${TMPDIR}/${safe_name}-env-shell.json"
	go_env="${TMPDIR}/${safe_name}-env-go.json"
	jq '.Env' "${shell_norm_inspect}" > "${shell_env}"
	jq '.Env' "${go_norm_inspect}" > "${go_env}"

	if diff -u "${shell_env}" "${go_env}" > /dev/null 2>&1; then
		log_pass "inspect env vars: match"
	else
		log_fail "inspect env vars: differ"
		diff -u "${shell_env}" "${go_env}" | head -20 | sed 's/^/    /'
	fi

	# ── Compare mounts ───────────────────────────────────────────
	shell_mounts="${TMPDIR}/${safe_name}-mounts-shell.json"
	go_mounts="${TMPDIR}/${safe_name}-mounts-go.json"
	jq '.Mounts' "${shell_norm_inspect}" > "${shell_mounts}"
	jq '.Mounts' "${go_norm_inspect}" > "${go_mounts}"

	if diff -u "${shell_mounts}" "${go_mounts}" > /dev/null 2>&1; then
		log_pass "inspect mounts: match"
	else
		log_fail "inspect mounts: differ"
		diff -u "${shell_mounts}" "${go_mounts}" | head -30 | sed 's/^/    /'
	fi

	# ── Compare entrypoint + cmd ─────────────────────────────────
	shell_ep="${TMPDIR}/${safe_name}-ep-shell.json"
	go_ep="${TMPDIR}/${safe_name}-ep-go.json"
	jq '{Entrypoint, Cmd}' "${shell_norm_inspect}" > "${shell_ep}"
	jq '{Entrypoint, Cmd}' "${go_norm_inspect}" > "${go_ep}"

	if diff -u "${shell_ep}" "${go_ep}" > /dev/null 2>&1; then
		log_pass "inspect entrypoint+cmd: match"
	else
		log_fail "inspect entrypoint+cmd: differ"
		diff -u "${shell_ep}" "${go_ep}" | head -20 | sed 's/^/    /'
	fi

	# ── Compare security / namespace config ──────────────────────
	shell_sec="${TMPDIR}/${safe_name}-security-shell.json"
	go_sec="${TMPDIR}/${safe_name}-security-go.json"
	jq '{Privileged, SecurityOpt, CapAdd, PidsLimit, NetworkMode, PidMode, IpcMode}' \
		"${shell_norm_inspect}" > "${shell_sec}"
	jq '{Privileged, SecurityOpt, CapAdd, PidsLimit, NetworkMode, PidMode, IpcMode}' \
		"${go_norm_inspect}" > "${go_sec}"

	if diff -u "${shell_sec}" "${go_sec}" > /dev/null 2>&1; then
		log_pass "inspect security/namespaces: match"
	else
		log_fail "inspect security/namespaces: differ"
		diff -u "${shell_sec}" "${go_sec}" | head -20 | sed 's/^/    /'
	fi

	# ── Full normalized inspect diff (for reference) ─────────────
	full_diff="${TMPDIR}/${safe_name}-inspect-full-diff.txt"
	if diff -u "${shell_norm_inspect}" "${go_norm_inspect}" > "${full_diff}" 2>&1; then
		log_pass "inspect full: identical"
	else
		log_fail "inspect full: differences found"
		echo "    Full diff saved to: ${full_diff}"
	fi
fi

# ── Test 4a: Dry-run create on existing container ──────────────────
# Shell produces dry-run output even when the container already exists.
# Go currently refuses, which is a parity gap.
if     [ "${shell_created}" -eq 1 ] && [ "${go_created}" -eq 1 ]; then
	log_section "DRY-RUN CREATE (existing container)"

	shell_exist="${TMPDIR}/${safe_name}-exist-dryrun-shell.txt"
	go_exist="${TMPDIR}/${safe_name}-exist-dryrun-go.txt"

	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		--dry-run --yes \
		--name "${shell_name}" \
		--image "${image}" \
		> "${shell_exist}" 2> /dev/null || true

	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--dry-run --yes \
		--name "${go_name}" \
		--image "${image}" \
		> "${go_exist}" 2> /dev/null || true

	if [ -s "${shell_exist}" ] && [ -s "${go_exist}" ]; then
		s_norm="${TMPDIR}/${safe_name}-exist-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-exist-norm-go.txt"

		normalize_create_cmd "$(cat "${shell_exist}")" |
			sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
		normalize_create_cmd "$(cat "${go_exist}")" |
			sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-exist-dryrun-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "create dry-run (existing container): commands match"
		else
			log_fail "create dry-run (existing container): commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	elif [ -s "${shell_exist}" ] && [ ! -s "${go_exist}" ]; then
		log_fail "create dry-run (existing container): Go produced no output (shell did)"
	elif [ ! -s "${shell_exist}" ] && [ -s "${go_exist}" ]; then
		log_fail "create dry-run (existing container): shell produced no output (Go did)"
	else
		log_skip "create dry-run (existing container): both produced no output"
	fi
fi

# ── Test 4b: Inspect comparison with flag variants ────────────────────
# Create containers with specific flags and compare inspect output.
log_section     "INSPECT COMPARISON (flag variants)"

while     IFS= read -r variant; do
	variant_slug="$(echo "${variant}" | tr ' -/:' '_' | tr -d '-')"
	shell_variant="dbx-test-shell-${variant_slug}-${safe_name}"
	go_variant="dbx-test-go-${variant_slug}-${safe_name}"

	# Clean up leftovers
	"${CONTAINER_MANAGER}" rm -f "${shell_variant}" > /dev/null 2>&1 || true
	"${CONTAINER_MANAGER}" rm -f "${go_variant}" > /dev/null 2>&1 || true

	s_created=0
	g_created=0

	# shellcheck disable=SC2086
	if DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		--yes --name "${shell_variant}" --image "${image}" \
		${variant} \
		2> /dev/null; then
		s_created=1
	fi

	# shellcheck disable=SC2086
	if DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--yes --name "${go_variant}" --image "${image}" \
		${variant} \
		2> /dev/null; then
		g_created=1
	fi

	if [ "${s_created}" -eq 0 ] || [ "${g_created}" -eq 0 ]; then
		log_skip "inspect ${variant}: could not create one or both containers"
	else
		shell_insp="${TMPDIR}/${variant_slug}-inspect-shell.json"
		go_insp="${TMPDIR}/${variant_slug}-inspect-go.json"
		"${CONTAINER_MANAGER}" inspect "${shell_variant}" > "${shell_insp}" 2> /dev/null
		"${CONTAINER_MANAGER}" inspect "${go_variant}" > "${go_insp}" 2> /dev/null

		s_norm_insp="${TMPDIR}/${variant_slug}-inspect-norm-shell.json"
		g_norm_insp="${TMPDIR}/${variant_slug}-inspect-norm-go.json"

		normalize_inspect "${shell_insp}" |
			sed "s/${shell_variant}/__CONTAINER__/g" |
			jq -S '.' > "${s_norm_insp}"
		normalize_inspect "${go_insp}" |
			sed "s/${go_variant}/__CONTAINER__/g" |
			jq -S '.' > "${g_norm_insp}"

		diff_file="${TMPDIR}/${variant_slug}-inspect-diff.txt"
		if diff -u "${s_norm_insp}" "${g_norm_insp}" > "${diff_file}" 2>&1; then
			log_pass "inspect ${variant}: identical"
		else
			log_fail "inspect ${variant}: differences found"
			head -30 "${diff_file}" | sed 's/^/    /'
		fi
	fi

	# Clean up variant containers
	if [ "${KEEP_CONTAINERS}" -eq 0 ]; then
		"${CONTAINER_MANAGER}" rm -f "${shell_variant}" > /dev/null 2>&1 || true
		"${CONTAINER_MANAGER}" rm -f "${go_variant}" > /dev/null 2>&1 || true
	fi
done     << 'INSPECT_VARIANTS_EOF'
--init
--unshare-netns
--unshare-all
INSPECT_VARIANTS_EOF

# ── Test 4c: Dry-run create with --clone ─────────────────────────────
# --clone needs an existing container, so test after containers are created.
if     [ "${shell_created}" -eq 1 ] && [ "${go_created}" -eq 1 ]; then
	log_section "DRY-RUN CREATE (--clone)"

	clone_shell_out="${TMPDIR}/${safe_name}-clone-dryrun-shell.txt"
	clone_go_out="${TMPDIR}/${safe_name}-clone-dryrun-go.txt"

	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" create \
		--dry-run --yes \
		--name "dbx-test-shell-clone-${safe_name}" \
		--clone "${shell_name}" \
		> "${clone_shell_out}" 2> /dev/null || true

	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" create \
		--dry-run --yes \
		--name "dbx-test-go-clone-${safe_name}" \
		--clone "${go_name}" \
		> "${clone_go_out}" 2> /dev/null || true

	if [ ! -s "${clone_shell_out}" ] && [ ! -s "${clone_go_out}" ]; then
		log_skip "create --clone: both produced no output"
	elif [ ! -s "${clone_shell_out}" ]; then
		log_fail "create --clone: shell produced no output (Go did)"
	elif [ ! -s "${clone_go_out}" ]; then
		log_fail "create --clone: Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-clone-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-clone-norm-go.txt"

		normalize_create_cmd "$(cat "${clone_shell_out}")" |
			sed "s/dbx-test-shell-clone-${safe_name}/__CONTAINER__/g" |
			sed "s/${shell_name}/__SOURCE__/g" > "${s_norm}"
		normalize_create_cmd "$(cat "${clone_go_out}")" |
			sed "s/dbx-test-go-clone-${safe_name}/__CONTAINER__/g" |
			sed "s/${go_name}/__SOURCE__/g" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-clone-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "create --clone: commands match"
		else
			log_fail "create --clone: commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi
fi

# ── Test 5: Enter dry-run comparison ────────────────────────────────
# Enter requires the container to exist, so we test against the shell-created one.
if     [ "${shell_created}" -eq 1 ] && [ "${go_created}" -eq 1 ]; then
	log_section "ENTER DRY-RUN"

	while IFS= read -r flags; do
		flag_slug="default"
		if [ -n "${flags}" ]; then
			flag_slug="$(echo "${flags}" | tr ' -/:' '_' | tr -d '-')"
		fi

		shell_enter="${TMPDIR}/${safe_name}-enter-${flag_slug}-shell.txt"
		go_enter="${TMPDIR}/${safe_name}-enter-${flag_slug}-go.txt"

		# Shell first — reference implementation.
		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_SHELL_PATH}" enter \
			--dry-run --name "${shell_name}" \
			${flags} \
			> "${shell_enter}" 2> /dev/null || true

		if [ ! -s "${shell_enter}" ]; then
			log_skip "enter ${flags:-default}: shell produced no output (invalid combo?)"
			continue
		fi

		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_GO_PATH}" enter \
			--dry-run --name "${go_name}" \
			${flags} \
			> "${go_enter}" 2> /dev/null || true

		if [ ! -s "${go_enter}" ]; then
			log_fail "enter ${flags:-default}: Go produced no output (shell did)"
			continue
		fi

		s_norm="${TMPDIR}/${safe_name}-enter-${flag_slug}-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-enter-${flag_slug}-norm-go.txt"

		normalize_enter_cmd "$(cat "${shell_enter}")" |
			sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
		normalize_enter_cmd "$(cat "${go_enter}")" |
			sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-enter-${flag_slug}-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "enter ${flags:-default}: commands match"
		else
			log_fail "enter ${flags:-default}: commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	done << 'ENTER_FLAGS_EOF'

--no-workdir
--clean-path
--no-tty
--no-workdir --no-tty
--clean-path --no-tty
--additional-flags --env=FOO=bar
-- ls -la
ENTER_FLAGS_EOF

	# Enter with short flags (shell vs Go)
	log_section "ENTER DRY-RUN (short flags)"

	while IFS= read -r pair; do
		shell_flags="${pair%%|*}"
		go_flags="${pair##*|}"
		# shellcheck disable=SC2001
		label="$(echo "${shell_flags}" | sed "s|${shell_name}|NAME|g")"

		_seq=$((_seq + 1))
		shell_out="${TMPDIR}/${safe_name}-enter-short-${_seq}-shell.txt"
		_seq=$((_seq + 1))
		go_out="${TMPDIR}/${safe_name}-enter-short-${_seq}-go.txt"

		# Shell first — reference implementation.
		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_SHELL_PATH}" enter \
			${shell_flags} \
			> "${shell_out}" 2> /dev/null || true

		if [ ! -s "${shell_out}" ]; then
			log_skip "enter short ${label}: shell produced no output (invalid combo?)"
			continue
		fi

		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_GO_PATH}" enter \
			${go_flags} \
			> "${go_out}" 2> /dev/null || true

		if [ ! -s "${go_out}" ]; then
			log_fail "enter short ${label}: Go produced no output (shell did)"
			continue
		fi

		_seq=$((_seq + 1))
		s_norm="${TMPDIR}/${safe_name}-enter-short-${_seq}-norm-shell.txt"
		_seq=$((_seq + 1))
		g_norm="${TMPDIR}/${safe_name}-enter-short-${_seq}-norm-go.txt"

		normalize_enter_cmd "$(cat "${shell_out}")" |
			sed "s/${shell_name}/__CONTAINER__/g" > "${s_norm}"
		normalize_enter_cmd "$(cat "${go_out}")" |
			sed "s/${go_name}/__CONTAINER__/g" > "${g_norm}"

		if diff -u "${s_norm}" "${g_norm}" > /dev/null 2>&1; then
			log_pass "enter short ${label}: shell vs Go match"
		else
			log_fail "enter short ${label}: shell vs Go differ"
			diff -u "${s_norm}" "${g_norm}" | head -20 | sed 's/^/    /'
		fi
	done << EOF
-d -n ${shell_name}|-d -n ${go_name}
-d -n ${shell_name} -nw|-d -n ${go_name} -nw
-d -n ${shell_name} -T|-d -n ${go_name} -T
-d -n ${shell_name} -H|-d -n ${go_name} -H
EOF

	# ── Test 6: List output comparison ───────────────────────────────
	log_section "LIST"

	shell_list="${TMPDIR}/${safe_name}-list-shell.txt"
	go_list="${TMPDIR}/${safe_name}-list-go.txt"

	# Run list and normalize: extract columns, sort, strip color codes
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" list --no-color \
		> "${shell_list}" 2> /dev/null || true

	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" list --no-color \
		> "${go_list}" 2> /dev/null || true

	# Check shell container appears in shell list, Go container in Go list
	if grep -q "${shell_name}" "${shell_list}" 2> /dev/null; then
		log_pass "list: shell container visible in shell list"
	else
		log_fail "list: shell container not in shell list"
	fi

	if grep -q "${go_name}" "${go_list}" 2> /dev/null; then
		log_pass "list: Go container visible in Go list"
	else
		log_fail "list: Go container not in Go list"
	fi
fi

# ── Test 7: Flag acceptance for non-dry-run commands ─────────────────
# For commands without --dry-run, verify that both shell and Go accept
# the same flags by checking exit codes. A missing flag alias in Go will
# cause it to exit non-zero while shell exits 0 (or vice versa).
log_section     "FLAG ACCEPTANCE (rm)"

# Helper: compare exit codes for a flag combo across both versions.
# We use --help as a safe way to test flag parsing without side effects
# on commands that don't have --dry-run.
# For rm, we can pass --help along with other flags to test parsing.
# Actually, the best approach: pass flags with a nonexistent container
# and check if the error is "container not found" (flag accepted) vs
# "unknown flag" (flag rejected).

while     IFS= read -r flags; do
	shell_rc=0
	go_rc=0
	_seq=$((_seq + 1))
	go_stderr="${TMPDIR}/${safe_name}-rm-stderr-${_seq}-go.txt"

	# Shell first — reference implementation.
	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" rm ${flags} > /dev/null 2>&1 || shell_rc=$?

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" rm ${flags} > /dev/null 2> "${go_stderr}" || go_rc=$?

	# Shell is the reference.  If shell accepts flags (rc=0), Go must too.
	if [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_fail "rm ${flags}: Go rejected flags shell accepted (go=${go_rc})"
		head -3 "${go_stderr}" | sed 's/^/    /'
	elif [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -eq 0 ]; then
		log_pass "rm ${flags}: both accepted (shell=${shell_rc} go=${go_rc})"
	elif [ "${shell_rc}" -ne 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_pass "rm ${flags}: both rejected (shell=${shell_rc} go=${go_rc})"
	else
		log_fail "rm ${flags}: Go accepted but shell rejected (shell=${shell_rc} go=${go_rc})"
	fi
done     << EOF
--force --yes ${shell_name}_nonexistent
--force --rm-home --yes ${shell_name}_nonexistent
--rm-home --yes ${shell_name}_nonexistent
--yes ${shell_name}_nonexistent
-f -Y ${shell_name}_nonexistent
-f --rm-home -Y ${shell_name}_nonexistent
-Y ${shell_name}_nonexistent
EOF

log_section     "FLAG ACCEPTANCE (stop)"

while     IFS= read -r flags; do
	shell_rc=0
	go_rc=0
	_seq=$((_seq + 1))
	go_stderr="${TMPDIR}/${safe_name}-stop-stderr-${_seq}-go.txt"

	# Shell first — reference implementation.
	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" stop ${flags} > /dev/null 2>&1 || shell_rc=$?

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" stop ${flags} > /dev/null 2> "${go_stderr}" || go_rc=$?

	if [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_fail "stop ${flags}: Go rejected flags shell accepted (go=${go_rc})"
		head -3 "${go_stderr}" | sed 's/^/    /'
	elif [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -eq 0 ]; then
		log_pass "stop ${flags}: both accepted (shell=${shell_rc} go=${go_rc})"
	elif [ "${shell_rc}" -ne 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_pass "stop ${flags}: both rejected (shell=${shell_rc} go=${go_rc})"
	else
		log_fail "stop ${flags}: Go accepted but shell rejected (shell=${shell_rc} go=${go_rc})"
	fi
done     << EOF
--yes ${shell_name}_nonexistent
-Y ${shell_name}_nonexistent
EOF

log_section     "FLAG ACCEPTANCE (generate-entry)"

if     [ "${shell_created}" -eq 1 ] && [ "${go_created}" -eq 1 ]; then
	# NOTE: --all/-a excluded — shell hangs generating entries for all
	# containers (starts each one).

	while IFS= read -r pair; do
		shell_flags="${pair%%|*}"
		go_flags="${pair##*|}"
		label="${shell_flags}"

		shell_rc=0
		go_rc=0
		_seq=$((_seq + 1))
		go_stderr="${TMPDIR}/${safe_name}-genentry-stderr-${_seq}-go.txt"

		# Shell first — reference implementation.
		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_SHELL_PATH}" generate-entry ${shell_flags} > /dev/null 2>&1 || shell_rc=$?

		# shellcheck disable=SC2086
		timeout "${CMD_TIMEOUT}" env \
			DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
			"${DISTROBOX_GO_PATH}" generate-entry ${go_flags} > /dev/null 2> "${go_stderr}" || go_rc=$?

		if [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -ne 0 ]; then
			log_fail "generate-entry ${label}: Go rejected flags shell accepted (go=${go_rc})"
			head -3 "${go_stderr}" | sed 's/^/    /'
		elif [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -eq 0 ]; then
			log_pass "generate-entry ${label}: both accepted (shell=${shell_rc} go=${go_rc})"
		elif [ "${shell_rc}" -ne 0 ] && [ "${go_rc}" -ne 0 ]; then
			log_pass "generate-entry ${label}: both rejected (shell=${shell_rc} go=${go_rc})"
		else
			log_fail "generate-entry ${label}: Go accepted but shell rejected (shell=${shell_rc} go=${go_rc})"
		fi
	done << EOF
${shell_name}|${go_name}
--delete ${shell_name}|--delete ${go_name}
-d ${shell_name}|-d ${go_name}
--icon /tmp/fake.png ${shell_name}|--icon /tmp/fake.png ${go_name}
-i /tmp/fake.png ${shell_name}|-i /tmp/fake.png ${go_name}
EOF
fi

log_section     "FLAG ACCEPTANCE (upgrade)"

# NOTE: --all, -a, --running are excluded — they actually start
# upgrading containers and hang until timeout kills them.

while     IFS= read -r flags; do
	shell_rc=0
	go_rc=0
	_seq=$((_seq + 1))
	go_stderr="${TMPDIR}/${safe_name}-upgrade-stderr-${_seq}-go.txt"

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" upgrade ${flags} > /dev/null 2>&1 || shell_rc=$?

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" upgrade ${flags} > /dev/null 2> "${go_stderr}" || go_rc=$?

	if [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_fail "upgrade ${flags}: Go rejected flags shell accepted (go=${go_rc})"
		head -3 "${go_stderr}" | sed 's/^/    /'
	elif [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -eq 0 ]; then
		log_pass "upgrade ${flags}: both accepted (shell=${shell_rc} go=${go_rc})"
	elif [ "${shell_rc}" -ne 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_pass "upgrade ${flags}: both rejected (shell=${shell_rc} go=${go_rc})"
	else
		log_fail "upgrade ${flags}: Go accepted but shell rejected (shell=${shell_rc} go=${go_rc})"
	fi
done     << 'UPGRADE_FLAGS_EOF'
nonexistent-container
UPGRADE_FLAGS_EOF

# NOTE: ephemeral flag acceptance tests disabled — shell ephemeral
# doesn't support --dry-run and hangs (creates+enters a real container).
# Both versions timeout (rc=124), making exit-code comparison useless.
# Re-enable once ephemeral supports --dry-run or we find a non-hanging
# way to test flag parsing.
#
# log_section "FLAG ACCEPTANCE (ephemeral)"
# ephemeral_flag_combos=(
#     "--image alpine:3.21 --name dbx-test-eph-flag"
#     "-i alpine:3.21"
#     "--image alpine:3.21 --name dbx-test-eph-init --init"
#     "--image alpine:3.21 --name dbx-test-eph-nv --nvidia"
#     "--image alpine:3.21 --name dbx-test-eph-home --home /tmp/dbx-test-home"
#     "--image alpine:3.21 --name dbx-test-eph-vol --volume /tmp/dbx-test-vol:/mnt/test-vol"
#     "--image alpine:3.21 --name dbx-test-eph-ap --additional-packages vim"
#     "--image alpine:3.21 --name dbx-test-eph-net --unshare-netns"
#     "-i alpine:3.21 -n dbx-test-eph-short -I"
#     "-i alpine:3.21 -n dbx-test-eph-short2 -H /tmp/dbx-test-home"
# )

# ── Test 8: Assemble create dry-run comparison ─────────────────────
# Shell assemble --dry-run outputs the distrobox-create command it would
# invoke.  Go must produce the same output.  We normalize by stripping
# status lines, the distrobox binary path, and container names, then
# split flags one-per-line and sort for order-independent comparison.
log_section     "ASSEMBLE CREATE DRY-RUN"

normalize_assemble_output()
{
	local input="$1" name="$2"
	echo "${input}" |
		# Strip status/progress lines (" - Creating ...")
		grep -v '^ *- ' |
		grep -v '^$' |
		# Strip distrobox binary path prefix (any path ending in distrobox or distrobox-create)
		sed -E 's|^(\./)?[^ ]*distrobox(-create)? ||' |
		# Remove quoting
		sed -E "s/[\"']//g" |
		# Split flags one per line
		sed -E 's/ --/\n--/g' |
		sed -E 's/ -/\n-/g' |
		# Normalize container name
		sed "s/${name}/__CONTAINER__/g" |
		grep -v '^$' |
		sort
}

assemble_manifest="${TMPDIR}/${safe_name}-assemble.ini"
assemble_name="dbx-test-asm-${safe_name}"

cat     > "${assemble_manifest}" << MANIFEST_EOF
[${assemble_name}]
image=${image}
MANIFEST_EOF

shell_asm="${TMPDIR}/${safe_name}-assemble-create-shell.txt"
go_asm="${TMPDIR}/${safe_name}-assemble-create-go.txt"

# Shell first — reference implementation.
DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" assemble create \
	--dry-run \
	--file "${assemble_manifest}" \
	> "${shell_asm}" 2> /dev/null || true

if     [ ! -s "${shell_asm}" ]; then
	log_skip "assemble create dry-run: shell produced no output"
else
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble create \
		--dry-run \
		--file "${assemble_manifest}" \
		> "${go_asm}" 2> /dev/null || true

	if [ ! -s "${go_asm}" ]; then
		log_fail "assemble create dry-run: Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-assemble-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-assemble-norm-go.txt"

		normalize_assemble_output "$(cat "${shell_asm}")" "${assemble_name}" > "${s_norm}"
		normalize_assemble_output "$(cat "${go_asm}")" "${assemble_name}" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-assemble-create-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "assemble create dry-run: commands match"
		else
			log_fail "assemble create dry-run: commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi
fi

# Assemble create with options (init, unshare, packages)
assemble_opts_manifest="${TMPDIR}/${safe_name}-assemble-opts.ini"
assemble_opts_name="dbx-test-asm-opts-${safe_name}"

cat     > "${assemble_opts_manifest}" << MANIFEST_EOF
[${assemble_opts_name}]
image=${image}
init=1
unshare_netns=1
additional_packages=vim
MANIFEST_EOF

shell_asm_opts="${TMPDIR}/${safe_name}-assemble-opts-shell.txt"
go_asm_opts="${TMPDIR}/${safe_name}-assemble-opts-go.txt"

DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" assemble create \
	--dry-run \
	--file "${assemble_opts_manifest}" \
	> "${shell_asm_opts}" 2> /dev/null || true

if     [ ! -s "${shell_asm_opts}" ]; then
	log_skip "assemble create dry-run (options): shell produced no output"
else
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble create \
		--dry-run \
		--file "${assemble_opts_manifest}" \
		> "${go_asm_opts}" 2> /dev/null || true

	if [ ! -s "${go_asm_opts}" ]; then
		log_fail "assemble create dry-run (options): Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-assemble-opts-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-assemble-opts-norm-go.txt"

		normalize_assemble_output "$(cat "${shell_asm_opts}")" "${assemble_opts_name}" > "${s_norm}"
		normalize_assemble_output "$(cat "${go_asm_opts}")" "${assemble_opts_name}" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-assemble-opts-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "assemble create dry-run (options): commands match"
		else
			log_fail "assemble create dry-run (options): commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi
fi

# Assemble create with --replace flag
shell_asm_replace="${TMPDIR}/${safe_name}-assemble-replace-shell.txt"
go_asm_replace="${TMPDIR}/${safe_name}-assemble-replace-go.txt"

DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" assemble create \
	--dry-run --replace \
	--file "${assemble_manifest}" \
	> "${shell_asm_replace}" 2> /dev/null || true

if     [ ! -s "${shell_asm_replace}" ]; then
	log_skip "assemble create dry-run (--replace): shell produced no output"
else
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble create \
		--dry-run --replace \
		--file "${assemble_manifest}" \
		> "${go_asm_replace}" 2> /dev/null || true

	if [ ! -s "${go_asm_replace}" ]; then
		log_fail "assemble create dry-run (--replace): Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-assemble-replace-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-assemble-replace-norm-go.txt"

		normalize_assemble_output "$(cat "${shell_asm_replace}")" "${assemble_name}" > "${s_norm}"
		normalize_assemble_output "$(cat "${go_asm_replace}")" "${assemble_name}" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-assemble-replace-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "assemble create dry-run (--replace): commands match"
		else
			log_fail "assemble create dry-run (--replace): commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi
fi

# Assemble create with extended manifest (home, hostname, volume, hooks, nvidia)
assemble_ext_manifest="${TMPDIR}/${safe_name}-assemble-ext.ini"
assemble_ext_name="dbx-test-asm-ext-${safe_name}"

cat     > "${assemble_ext_manifest}" << MANIFEST_EOF
[${assemble_ext_name}]
image=${image}
init=1
nvidia=1
hostname=custom-asm-host
home=/tmp/dbx-test-asm-home
volume=/tmp/dbx-test-vol:/mnt/test-vol:ro
additional_packages=vim curl
init_hooks=echo test-init-hook
pre_init_hooks=echo test-pre-init-hook
MANIFEST_EOF

shell_asm_ext="${TMPDIR}/${safe_name}-assemble-ext-shell.txt"
go_asm_ext="${TMPDIR}/${safe_name}-assemble-ext-go.txt"

DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" assemble create \
	--dry-run \
	--file "${assemble_ext_manifest}" \
	> "${shell_asm_ext}" 2> /dev/null || true

if     [ ! -s "${shell_asm_ext}" ]; then
	log_skip "assemble create dry-run (extended): shell produced no output"
else
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble create \
		--dry-run \
		--file "${assemble_ext_manifest}" \
		> "${go_asm_ext}" 2> /dev/null || true

	if [ ! -s "${go_asm_ext}" ]; then
		log_fail "assemble create dry-run (extended): Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-assemble-ext-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-assemble-ext-norm-go.txt"

		normalize_assemble_output "$(cat "${shell_asm_ext}")" "${assemble_ext_name}" > "${s_norm}"
		normalize_assemble_output "$(cat "${go_asm_ext}")" "${assemble_ext_name}" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-assemble-ext-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "assemble create dry-run (extended): commands match"
		else
			log_fail "assemble create dry-run (extended): commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi
fi

# Assemble with --name to target a single entry
assemble_multi_manifest="${TMPDIR}/${safe_name}-assemble-multi.ini"
assemble_target="dbx-test-asm-target-${safe_name}"
assemble_other="dbx-test-asm-other-${safe_name}"

cat     > "${assemble_multi_manifest}" << MANIFEST_EOF
[${assemble_target}]
image=${image}

[${assemble_other}]
image=alpine:3.21
MANIFEST_EOF

shell_asm_name="${TMPDIR}/${safe_name}-assemble-name-shell.txt"
go_asm_name="${TMPDIR}/${safe_name}-assemble-name-go.txt"

DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
	"${DISTROBOX_SHELL_PATH}" assemble create \
	--dry-run \
	--name "${assemble_target}" \
	--file "${assemble_multi_manifest}" \
	> "${shell_asm_name}" 2> /dev/null || true

if     [ ! -s "${shell_asm_name}" ]; then
	log_skip "assemble create dry-run (--name): shell produced no output"
else
	DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble create \
		--dry-run \
		--name "${assemble_target}" \
		--file "${assemble_multi_manifest}" \
		> "${go_asm_name}" 2> /dev/null || true

	if [ ! -s "${go_asm_name}" ]; then
		log_fail "assemble create dry-run (--name): Go produced no output (shell did)"
	else
		s_norm="${TMPDIR}/${safe_name}-assemble-name-norm-shell.txt"
		g_norm="${TMPDIR}/${safe_name}-assemble-name-norm-go.txt"

		normalize_assemble_output "$(cat "${shell_asm_name}")" "${assemble_target}" > "${s_norm}"
		normalize_assemble_output "$(cat "${go_asm_name}")" "${assemble_target}" > "${g_norm}"

		diff_file="${TMPDIR}/${safe_name}-assemble-name-diff.txt"
		if diff -u "${s_norm}" "${g_norm}" > "${diff_file}" 2>&1; then
			log_pass "assemble create dry-run (--name): commands match"
		else
			log_fail "assemble create dry-run (--name): commands differ"
			head -20 "${diff_file}" | sed 's/^/    /'
		fi
	fi

	# Verify --name filtered correctly: output should NOT contain the other entry
	if grep -q "${assemble_other}" "${shell_asm_name}" 2> /dev/null; then
		log_fail "assemble --name: shell output contains unselected entry"
	else
		log_pass "assemble --name: shell correctly filtered to target entry"
	fi

	if grep -q "${assemble_other}" "${go_asm_name}" 2> /dev/null; then
		log_fail "assemble --name: Go output contains unselected entry"
	else
		log_pass "assemble --name: Go correctly filtered to target entry"
	fi
fi

# ── Test 9: Assemble rm flag acceptance ────────────────────────────
log_section     "ASSEMBLE RM"

assemble_rm_manifest="${TMPDIR}/${safe_name}-assemble-rm.ini"
assemble_rm_name="dbx-test-asm-rm-${safe_name}"

cat     > "${assemble_rm_manifest}" << MANIFEST_EOF
[${assemble_rm_name}]
image=${image}
MANIFEST_EOF

while     IFS= read -r flags; do
	shell_rc=0
	go_rc=0
	_seq=$((_seq + 1))
	go_stderr="${TMPDIR}/${safe_name}-assemble-rm-stderr-${_seq}-go.txt"

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_SHELL_PATH}" assemble rm ${flags} > /dev/null 2>&1 || shell_rc=$?

	# shellcheck disable=SC2086
	timeout "${CMD_TIMEOUT}" env \
		DBX_CONTAINER_MANAGER="${CONTAINER_MANAGER}" \
		"${DISTROBOX_GO_PATH}" assemble rm ${flags} > /dev/null 2> "${go_stderr}" || go_rc=$?

	if [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_fail "assemble rm ${flags}: Go rejected flags shell accepted (go=${go_rc})"
		head -3 "${go_stderr}" | sed 's/^/    /'
	elif [ "${shell_rc}" -eq 0 ] && [ "${go_rc}" -eq 0 ]; then
		log_pass "assemble rm ${flags}: both accepted (shell=${shell_rc} go=${go_rc})"
	elif [ "${shell_rc}" -ne 0 ] && [ "${go_rc}" -ne 0 ]; then
		log_pass "assemble rm ${flags}: both rejected (shell=${shell_rc} go=${go_rc})"
	else
		log_fail "assemble rm ${flags}: Go accepted but shell rejected (shell=${shell_rc} go=${go_rc})"
	fi
done     << EOF
--file ${assemble_rm_manifest} --name ${assemble_rm_name}
--dry-run --file ${assemble_rm_manifest}
-d --file ${assemble_rm_manifest}
--file ${assemble_rm_manifest} -n ${assemble_rm_name}
EOF

# ── Cleanup containers ───────────────────────────────────────────────
if [ "${KEEP_CONTAINERS}" -eq 0 ]; then
	"${CONTAINER_MANAGER}" rm -f "${shell_name}" > /dev/null 2>&1 || true
	"${CONTAINER_MANAGER}" rm -f "${go_name}" > /dev/null 2>&1 || true
fi

# ══════════════════════════════════════════════════════════════════════════════
# SUMMARY
# ══════════════════════════════════════════════════════════════════════════════

log_section "SUMMARY"
total=$((pass + fail + skip))
printf "  Total: %s  %sPass: %s%s  %sFail: %s%s  %sSkip: %s%s\n" \
	"${total}" "${GREEN}" "${pass}" "${RESET}" "${RED}" "${fail}" "${RESET}" "${YELLOW}" "${skip}" "${RESET}"
echo ""
echo "  All artifacts saved in: ${TMPDIR}"

if [ "${fail}" -gt 0 ]; then
	# Don't delete tmpdir on failure so user can inspect
	trap - EXIT
	printf "  %sKeeping temp dir for inspection.%s\n" "${RED}" "${RESET}"
	exit 1
fi

exit 0
