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
# usage: commands.sh <image> <podman|docker>

set -u

IMAGE="${1:?usage: commands.sh <image> <podman|docker>}"
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
		printf '%s\n' "${2}" | tr '\r' '\n' | sed 's/^/      | /'
	fi
}
ok()
{
	desc="${1}"
	shift
	if out="$("$@" < /dev/null 2>&1)"; then pass "${desc}"; else fail "${desc}" "${out}"; fi
}
count()
{
	engine ps -a -q 2> /dev/null | wc -l | tr -d ' '
}
warmup()
{
	dbx enter --name "${1}" -- true < /dev/null > /dev/null 2>&1 || true
}
eq()
{
	if [ "${2}" = "${3}" ]; then pass "${1}"; else fail "${1}" "got '${2}', want '${3}'"; fi
}
contains()
{
	if printf '%s' "${2}" | grep -qE "${3}"; then pass "${1}"; else fail "${1}" "want /${3}/ in: ${2}"; fi
}
enter_out()
{
	b="${1}"
	shift
	dbx enter --name "${b}" -- "$@" < /dev/null 2> /dev/null | tr -d '\r\000\n'
}

printf '\n== commands: image=%s backend=%s ==\n' "${IMAGE}" "${MODE}"
engine pull "${IMAGE}" > /dev/null 2>&1 || true

# Captured while pristine, to gate the destructive --all tests (which act on EVERY
# distrobox) so a developer's own boxes are never touched. CI runners are clean.
pre_boxes="$(dbx list 2> /dev/null | tail -n +2 | grep -c .)"

# ---- ephemeral: a temporary box, auto-removed on exit ----
before="$(count)"
ok "ephemeral runs a command" dbx ephemeral --yes --image "${IMAGE}" -- true
after="$(count)"
if [ "${before}" = "${after}" ]; then
	pass "ephemeral leaves no leftover container"
else
	fail "ephemeral leaves no leftover container" "before=${before} after=${after}"
fi

# ---- flag checks that spin no container (fast) ----
ok "--version prints" dbx --version
contains "create --compatibility lists images" "$(dbx create --compatibility < /dev/null 2>&1)" 'alpine|ubuntu|fedora|debian'
contains "create --dry-run prints the engine command" "$(dbx create --dry-run --image "${IMAGE}" --name dbx-cmd-dry < /dev/null 2>&1)" "${CM}.*create"
if dbx list 2> /dev/null | grep -q dbx-cmd-dry; then fail "create --dry-run creates nothing" "dbx-cmd-dry present"; else pass "create --dry-run creates nothing"; fi
nc="$(dbx list --no-color < /dev/null 2>&1)"
if printf '%s' "${nc}" | grep -q "$(printf '\033')"; then fail "list --no-color emits no ANSI" "escape found"; else pass "list --no-color emits no ANSI"; fi
ok "ls alias resolves to list" dbx ls

# ---- base box -> generate-entry + clone (guarded on create success) ----
gbox="dbx-cmd-genentry"
cbox="${gbox}-clone"
if cout="$(dbx create --yes --no-entry --image "${IMAGE}" --name "${gbox}" < /dev/null 2>&1)"; then
	pass "create (base box)"
	# First enter triggers setup; sudo writes a marker into the container rootfs
	# (not a host-shared path) so a clone must carry it.
	dbx enter --name "${gbox}" -- sudo -n touch /dbx-clone-marker < /dev/null > /dev/null 2>&1 || true

	desktop="${XDG_DATA_HOME:-${HOME}/.local/share}/applications/${gbox}.desktop"
	geout="$(dbx generate-entry "${gbox}" < /dev/null 2>&1)"
	if [ -f "${desktop}" ]; then pass "generate-entry writes a .desktop"; else fail "generate-entry writes a .desktop" "expected ${desktop}; generate-entry said: ${geout}"; fi
	dbx generate-entry --delete "${gbox}" < /dev/null > /dev/null 2>&1 || true
	if [ ! -f "${desktop}" ]; then pass "generate-entry --delete removes it"; else fail "generate-entry --delete removes it" "${desktop} still present"; fi

	dbx stop --yes "${gbox}" < /dev/null > /dev/null 2>&1 || true
	if clout="$(dbx create --yes --clone "${gbox}" --name "${cbox}" < /dev/null 2>&1)"; then
		pass "clone a stopped box"
		warmup "${cbox}"
		marker="$(dbx enter --name "${cbox}" -- sh -c 'test -f /dbx-clone-marker && printf yes' < /dev/null 2>&1 | tr -d '\r\000\n')"
		if [ "${marker}" = "yes" ]; then pass "clone carries the rootfs marker"; else fail "clone carries the rootfs marker" "marker /dbx-clone-marker missing; enter returned: '${marker}'"; fi
	else
		fail "clone a stopped box" "${clout}"
		fail "clone carries the rootfs marker" "no clone created"
	fi
else
	fail "create (base box)" "${cout}"
fi

# ---- create/enter flags + engine-level mechanics, on one box ----
fbox="dbx-cmd-flags"
vdir="$(mktemp -d)"
: > "${vdir}/vfile"
if fout="$(dbx create --yes --no-entry --image "${IMAGE}" --name "${fbox}" \
	--hostname dbxflagshost \
	--volume "${vdir}:/mnt/vol" \
	--init-hooks 'touch /dbx-hook-ran' \
	--additional-flags '--env CREATEVAR=cval' < /dev/null 2>&1)"; then
	pass "create with a flag bundle"
	warmup "${fbox}"
	eq "--hostname sets the hostname" "$(enter_out "${fbox}" uname -n)" "dbxflagshost"
	ok "--volume mounts a host dir" dbx enter --name "${fbox}" -- test -f /mnt/vol/vfile
	ok "--init-hooks runs at setup" dbx enter --name "${fbox}" -- test -f /dbx-hook-ran
	eq "create --additional-flags forwards --env" "$(enter_out "${fbox}" printenv CREATEVAR)" "cval"
	eq "enter --additional-flags forwards --env" \
		"$(dbx enter --name "${fbox}" --additional-flags '--env ENTERVAR=eval' -- printenv ENTERVAR < /dev/null 2> /dev/null | tr -d '\r\000\n')" "eval"
	hm="${HOME}/.dbx-cmd-home-$$"
	: > "${hm}"
	ok 'host $HOME is shared into the box' dbx enter --name "${fbox}" -- test -f "${hm}"
	rm -f "${hm}"
	eq "host env var forwarded" \
		"$(MY_DBX_VAR=fwd dbx enter --name "${fbox}" -- printenv MY_DBX_VAR < /dev/null 2> /dev/null | tr -d '\r\000\n')" "fwd"
	badv="$(DBX_CMD_BAD='has$dollar' dbx enter --name "${fbox}" -- printenv DBX_CMD_BAD < /dev/null 2> /dev/null | tr -d '\r\000\n')"
	if [ -z "${badv}" ]; then pass "env var with special chars is filtered"; else fail "env var with special chars is filtered" "leaked '${badv}'"; fi
	pn="$(enter_out "${fbox}" printenv PATH)"
	pc="$(dbx enter --name "${fbox}" --clean-path -- printenv PATH < /dev/null 2> /dev/null | tr -d '\r\000\n')"
	if [ -n "${pc}" ] && [ "${pc}" != "${pn}" ] && printf '%s' ":${pc}:" | grep -q ':/usr/bin:'; then pass "enter --clean-path resets PATH to FHS"; else fail "enter --clean-path resets PATH to FHS" "normal='${pn}' clean='${pc}'"; fi
	eq 'enter --no-workdir starts in $HOME' \
		"$(dbx enter --name "${fbox}" --no-workdir -- pwd < /dev/null 2> /dev/null | tr -d '\r\000\n')" "${HOME}"
	contains "enter --dry-run prints the engine command" "$(dbx enter --name "${fbox}" --dry-run -- true < /dev/null 2>&1)" "${CM}.*exec"
else
	fail "create with a flag bundle" "${fout}"
fi
dbx rm --force "${fbox}" < /dev/null > /dev/null 2>&1 || true
rm -rf "${vdir}"

# ---- --home (custom HOME) + rm --rm-home ----
chome="$(mktemp -d)/box-home"
if dbx create --yes --no-entry --image "${IMAGE}" --name dbx-cmd-home --home "${chome}" < /dev/null > /dev/null 2>&1; then
	warmup dbx-cmd-home
	eq "--home sets a custom HOME" "$(enter_out dbx-cmd-home printenv HOME)" "${chome}"
	dbx rm --rm-home --force dbx-cmd-home < /dev/null > /dev/null 2>&1 || true
	if [ -d "${chome}" ]; then fail "rm --rm-home removes the custom home" "${chome} still present"; else pass "rm --rm-home removes the custom home"; fi
else
	fail "--home sets a custom HOME" "create --home failed"
	fail "rm --rm-home removes the custom home" "no box created"
fi
rm -rf "${chome%/*}"

# ---- assemble: create/rm boxes from a manifest ----
mani="$(mktemp)"
printf '[dbx-cmd-assemble]\nimage=%s\n' "${IMAGE}" > "${mani}"
ok "assemble create" dbx assemble create --file "${mani}"
if dbx list | grep -q dbx-cmd-assemble; then pass "assemble created the box"; else fail "assemble created the box" "dbx-cmd-assemble not in distrobox list"; fi
ok "assemble rm" dbx assemble rm --file "${mani}"
if dbx list | grep -q dbx-cmd-assemble; then fail "assemble rm removed the box" "dbx-cmd-assemble still in distrobox list"; else pass "assemble rm removed the box"; fi
rm -f "${mani}"

# ---- assemble: --dry-run, --name (single entry), --replace ----
mani2="$(mktemp)"
printf '[dbx-cmd-asm1]\nimage=%s\n\n[dbx-cmd-asm2]\nimage=%s\n' "${IMAGE}" "${IMAGE}" > "${mani2}"
if dbx assemble create --file "${mani2}" --dry-run < /dev/null > /dev/null 2>&1 && ! dbx list 2> /dev/null | grep -q dbx-cmd-asm; then pass "assemble --dry-run creates nothing"; else fail "assemble --dry-run creates nothing" "$(dbx list 2> /dev/null | grep dbx-cmd-asm || true)"; fi
ok "assemble create --name (single entry)" dbx assemble create --file "${mani2}" --name dbx-cmd-asm1
if dbx list 2> /dev/null | grep -q dbx-cmd-asm1 && ! dbx list 2> /dev/null | grep -q dbx-cmd-asm2; then pass "assemble --name builds only the named entry"; else fail "assemble --name builds only the named entry" "$(dbx list 2> /dev/null | grep dbx-cmd-asm || true)"; fi
ok "assemble create --replace" dbx assemble create --file "${mani2}" --name dbx-cmd-asm1 --replace
dbx assemble rm --file "${mani2}" < /dev/null > /dev/null 2>&1 || true
rm -f "${mani2}"

# ---- multi-box --all flags (act on EVERY distrobox; guarded for safety) ----
# Runs only from a clean start so a dev's own boxes are never touched (CI is clean).
if [ "${pre_boxes}" = "0" ]; then
	dbx create --yes --no-entry --image "${IMAGE}" --name dbx-cmd-all1 < /dev/null > /dev/null 2>&1 || true
	dbx create --yes --no-entry --image "${IMAGE}" --name dbx-cmd-all2 < /dev/null > /dev/null 2>&1 || true
	warmup dbx-cmd-all1
	warmup dbx-cmd-all2
	appdir="${XDG_DATA_HOME:-${HOME}/.local/share}/applications"
	icon="${HOME}/.dbx-cmd-icon.png"
	: > "${icon}"
	dbx generate-entry --all < /dev/null > /dev/null 2>&1 || true
	if [ -f "${appdir}/dbx-cmd-all1.desktop" ] && [ -f "${appdir}/dbx-cmd-all2.desktop" ]; then pass "generate-entry --all covers every box"; else fail "generate-entry --all covers every box" "missing .desktop in ${appdir}"; fi
	dbx generate-entry dbx-cmd-all1 --icon "${icon}" < /dev/null > /dev/null 2>&1 || true
	contains "generate-entry --icon sets a custom icon" "$(cat "${appdir}/dbx-cmd-all1.desktop" 2> /dev/null)" "Icon=${icon}"
	dbx generate-entry --all --delete < /dev/null > /dev/null 2>&1 || true
	rm -f "${icon}"
	ok "stop --all stops every box" dbx stop --all --yes
	ok "upgrade --all upgrades every box" dbx upgrade --all
	dbx rm --all --force < /dev/null > /dev/null 2>&1 || true
	if [ "$(dbx list 2> /dev/null | tail -n +2 | grep -c .)" = "0" ]; then pass "rm --all removes every box"; else fail "rm --all removes every box" "boxes remain"; fi
else
	printf '  \033[1;33mSKIP\033[0m --all flag tests (%s pre-existing distrobox(es); would be destructive)\n' "${pre_boxes}"
fi

# ---- cleanup ----
dbx rm --force "${gbox}" "${cbox}" "${fbox}" < /dev/null > /dev/null 2>&1 || true
if [ -z "${DBX_E2E_KEEP_IMAGE:-}" ]; then
	engine rmi -f "${IMAGE}" > /dev/null 2>&1 || true
fi

printf '== commands result: %d failed ==\n' "${fails}"
[ "${fails}" -eq 0 ]
