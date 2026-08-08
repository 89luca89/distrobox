#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
#
# This file is part of the distrobox project:
#    https://github.com/89luca89/distrobox
#
# Copyright (C) 2021 distrobox contributors
#
# distrobox is free software; you can redistribute it and/or modify it
# under the terms of the GNU General Public License version 3
# as published by the Free Software Foundation.
#
# distrobox is distributed in the hope that it will be useful, but
# WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
# General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with distrobox; if not, see <http://www.gnu.org/licenses/>.
#
# usage: e2e.sh <image> <podman|docker>

set -u

IMAGE="${1:?usage: e2e.sh <image> <podman|docker>}"
MODE="${2:?backend required: podman|docker}"

case "${MODE}" in
	podman) CM=podman ;;
	docker) CM=docker ;;
	*)
		printf 'unknown backend: %s (use podman|docker)\n' "${MODE}" >&2
		exit 2
		;;
esac
export DBX_CONTAINER_MANAGER="${CM}"
DBX="${DBX:-distrobox}"

dbx()
{
	"${DBX}" "$@"
}
engine()
{
	"${CM}" "$@"
}

name="$(basename "${IMAGE}" | sed -E 's/[:.]/-/g')"
fails=0

pass()
{
	printf '  \033[1;32mPASS\033[0m %s\n' "${1}"
}
fail()
{
	printf '  \033[1;31mFAIL\033[0m %s\n' "${1}"
	fails=$((fails + 1))
	if [ -n "${2:-}" ]; then
		# \r -> \n so distrobox progress output (carriage returns) is readable
		# line-by-line instead of one smeared line.
		printf '%s\n' "${2}" | tr '\r' '\n' | sed 's/^/      | /'
	fi
}
eq()
{
	if [ "${2}" = "${3}" ]; then pass "${1}"; else fail "${1}" "got '${2}', want '${3}'"; fi
}
ok()
{
	desc="${1}"
	shift
	if out="$("$@" < /dev/null 2>&1)"; then
		pass "${desc}"
	else
		fail "${desc}" "${out}"
	fi
}
enter_out()
{
	dbx enter --name "${name}" -- "$@" < /dev/null 2> /dev/null | tr -d '\r\000\n'
}

# dump the box's logs (distrobox-init setup output). Only meaningful when the
# container itself won't come up, so it's called only on create/restart failure.
dump_logs()
{
	printf '  --- %s logs for %s (tail) ---\n' "${MODE}" "${name}"
	engine logs "${name}" 2>&1 | tail -40 | sed 's/^/      | /' || true
}

printf '\n== e2e: image=%s backend=%s name=%s ==\n' "${IMAGE}" "${MODE}" "${name}"

case "${name}" in
	*init*) create_flags="--pull --yes --image ${IMAGE} --name ${name} --init --unshare-all" ;;
	*) create_flags="--pull --yes --image ${IMAGE} --name ${name} --additional-packages nano" ;;
esac
# shellcheck disable=SC2086 # create_flags is intentionally word-split
if create_out="$(dbx create ${create_flags} < /dev/null 2>&1)"; then
	pass "create"
else
	fail "create" "${create_out}"
	dump_logs
	printf '== result: %d failed ==\n' "${fails}"
	exit 1
fi

# Warm up: the first enter STARTS the container, and `podman start` echoes the
# box name to stdout (plus first-boot setup streams). Do it once, discarded, so
# the checks below capture only their own command's output.
ok "enter" dbx enter --name "${name}" -- true

# --- systemd boots for --init images ---
case "${name}" in
	*init*)
		sysstate="$(enter_out systemctl is-system-running)"
		case "${sysstate}" in
			running | degraded | starting) pass "systemd is running (--init): ${sysstate}" ;;
			*) fail "systemd is running (--init)" "got '${sysstate}'" ;;
		esac
		;;
	*) ;;
esac

# --- core promises (hard) ---
eq "enter runs as the host user" "$(enter_out whoami)" "$(whoami)"
sudo_out="$(enter_out sudo -n whoami)"
case "${sudo_out}" in
	*root*) pass "passwordless sudo inside the box" ;;
	*) fail "passwordless sudo inside the box" "want 'root' in output, got '${sudo_out}'" ;;
esac

# --- additional-packages: nano (plain images only; init skips it, see create) ---
case "${name}" in
	*init*) ;;
	*) ok "additional-packages installed nano" dbx enter --name "${name}" -- sh -c 'command -v nano' ;;
esac

# --- lifecycle (hard) ---
ok "upgrade" dbx upgrade "${name}"
ok "stop" dbx stop --yes "${name}"
# restart: entering a stopped box must bring it back up. timeout(1) guards a
# stuck setup, but it needs a real executable (dbx is a shell function), so call
# the binary directly.
restart_out="$(timeout 180 "${DBX}" enter --name "${name}" -- true < /dev/null 2>&1)"
rc=$?
if [ "${rc}" -eq 0 ]; then
	pass "enter restarts a stopped box"
else
	fail "enter restarts a stopped box" "${restart_out}"
	dump_logs
fi

ok "rm" dbx rm --force "${name}"
if [ -z "${DBX_E2E_KEEP_IMAGE:-}" ]; then
	engine rmi -f "${IMAGE}" > /dev/null 2>&1 || true
fi

printf '== result: %d failed ==\n' "${fails}"
[ "${fails}" -eq 0 ]
