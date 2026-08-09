#!/bin/sh
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
set -u
#
# Runs inside the test VM (spawned by test-nvidia-integration.sh). Installs the
# nvidia userspace driver for the host distro, then for each guest image checks
# every driver file is mirrored into the container with a matching checksum.
# The distro comes from $1 or /mnt/share/distro; the result and diagnostics go
# to the serial console and the /mnt/out share.
#

DISTRO="${1:-$(cat /mnt/share/distro 2> /dev/null || true)}"
SERIAL=/dev/ttyS0
OUT=/mnt/out
BOX_IMAGES="quay.io/toolbx/ubuntu-toolbox:24.04 registry.fedoraproject.org/fedora-toolbox:44 quay.io/toolbx/arch-toolbox:latest quay.io/toolbx-images/debian-toolbox:latest"

log()
{
	printf '[test] %s\n' "$*"
	printf '[test] %s\n' "$*" > "${SERIAL}" 2> /dev/null || true
}
pass()
{
	printf '=== RESULT: PASS ===\n' > "${SERIAL}" 2> /dev/null || true
	sync
	sleep 2
	poweroff -f
	exit 0
}
fail()
{
	log "FAIL: $*"
	printf '=== RESULT: FAIL ===\n' > "${SERIAL}" 2> /dev/null || true
	sync
	sleep 2
	poweroff -f
	exit 1
}
dbx()
{
	/usr/local/bin/distrobox "$@"
}
outok()
{
	[ -d "${OUT}" ] && [ -w "${OUT}" ]
}
# run_log CMD...: append CMD's combined output to install.log (and echo it to
# the console via cloud-init), returning CMD's own exit status. A bare
# "cmd | tee" would return tee's status instead, and POSIX sh has no pipefail.
run_log()
{
	{
		"$@" 2>&1
		echo "$?" > /tmp/.rc
	} | tee -a "${OUT}/install.log"
	read -r _rc < /tmp/.rc
	return "${_rc}"
}

[ -n "${DISTRO}" ] || fail "no distro given"
log "host ${DISTRO}: installing container manager + nvidia userspace driver"
case "${DISTRO}" in
	ubuntu)
		export DEBIAN_FRONTEND=noninteractive
		run_log apt-get update -y || fail "apt update"
		run_log apt-get install -y podman || fail "install podman"
		b="$(apt-cache search --names-only '^nvidia-utils-[0-9]+$' | grep -oE '[0-9]+' | sort -rn | head -1)"
		[ -n "${b}" ] || fail "no nvidia-utils package in this image"
		pkgs="libnvidia-compute-${b} libnvidia-gl-${b} libnvidia-decode-${b} libnvidia-encode-${b} libnvidia-extra-${b} nvidia-utils-${b}"
		# shellcheck disable=SC2086
		run_log apt-get install -y ${pkgs} || fail "install nvidia"
		dpkg --add-architecture i386 && run_log apt-get update -y
		run_log apt-get install -y "libnvidia-gl-${b}:i386" && pkgs="${pkgs} libnvidia-gl-${b}:i386"
		# shellcheck disable=SC2086
		pm_list()
		{
			dpkg -L ${pkgs} 2> /dev/null
		}
		;;
	debian)
		export DEBIAN_FRONTEND=noninteractive
		sed -i -E 's/^Components:.*/Components: main contrib non-free non-free-firmware/' \
			/etc/apt/sources.list.d/debian.sources
		run_log apt-get update -y || fail "apt update"
		run_log apt-get install -y podman || fail "install podman"
		pkgs="libcuda1 libnvidia-ml1 libglx-nvidia0 libegl-nvidia0 libgles-nvidia1 libgles-nvidia2 libnvidia-eglcore libnvidia-glcore libnvidia-glvkspirv libnvidia-gpucomp libnvidia-nvvm4 libnvidia-ptxjitcompiler1 libnvidia-rtcore libnvidia-encode1 libnvidia-cfg1 libnvidia-allocator1 libnvidia-fbc1 libnvidia-pkcs11-openssl3 libnvidia-egl-wayland1 nvidia-opencl-icd nvidia-egl-common nvidia-opencl-common nvidia-vulkan-common nvidia-vdpau-driver"
		# shellcheck disable=SC2086
		run_log apt-get install -y --no-install-recommends ${pkgs} || fail "install nvidia"
		# shellcheck disable=SC2086
		pm_list()
		{
			dpkg -L ${pkgs} 2> /dev/null
		}
		;;
	fedora)
		run_log dnf install -y podman "https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm" || {
			tail -25 "${OUT}/install.log" > "${SERIAL}" 2> /dev/null
			fail "install podman/rpmfusion (see install.log)"
		}
		pkgs="xorg-x11-drv-nvidia-libs xorg-x11-drv-nvidia-cuda-libs"
		# shellcheck disable=SC2086
		run_log dnf install -y ${pkgs} || {
			tail -25 "${OUT}/install.log" > "${SERIAL}" 2> /dev/null
			fail "install nvidia (see install.log)"
		}
		run_log dnf install -y xorg-x11-drv-nvidia-libs.i686 && pkgs="${pkgs} xorg-x11-drv-nvidia-libs.i686"
		# shellcheck disable=SC2086
		pm_list()
		{
			rpm -ql ${pkgs} 2> /dev/null
		}
		;;
	arch)
		sed -i '/^#\[multilib\]/,/^#Include/ s/^#//' /etc/pacman.conf
		run_log pacman -Sy --noconfirm --needed podman || fail "install podman"
		pkgs="nvidia-utils opencl-nvidia"
		# shellcheck disable=SC2086
		run_log pacman -S --noconfirm --needed ${pkgs} || fail "install nvidia"
		run_log pacman -S --noconfirm --needed lib32-nvidia-utils && pkgs="${pkgs} lib32-nvidia-utils"
		# shellcheck disable=SC2086
		pm_list()
		{
			pacman -Ql ${pkgs} 2> /dev/null | awk '{print $2}'
		}
		;;
	*)
		fail "unknown distro: ${DISTRO}"
		;;
esac
ldconfig 2> /dev/null || true

install -m0755 /mnt/share/distrobox /usr/local/bin/distrobox || fail "install distrobox"

# Every driver file is required in the guest except these non-runtime bits.
is_excluded()
{
	case "$1" in
		*/share/doc/* | */share/man/* | */share/lintian/* | */share/licenses/* | */share/metainfo/* | */share/applications/* | */share/icons/* | */share/pixmaps/* | */share/bug/*) return 0 ;;
		*/include/* | *.h | *.hpp | *.a | *.la) return 0 ;;
		*/lib/modules/* | *.ko | *.ko.* | */firmware/*) return 0 ;;
		*/.build-id/* | */lib/debug/* | */systemd/* | *.service | *.socket | */udev/* | *.rules) return 0 ;;
		*/modprobe.d/* | */modules-load.d/* | */etc/alternatives/* | */etc/ld.so.conf.d/*) return 0 ;;
		*/sysusers.d/* | */dbus-1/system.d/* | */nvidia/files.d/* | */nvidia-powerd/* | *-key-documentation) return 0 ;;
		*copyright | *changelog* | *NEWS* | *README* | *.md) return 0 ;;
		*) return 1 ;;
	esac
}
elfclass()
{
	od -An -t u1 -j 4 -N 1 "$1" 2> /dev/null | tr -d ' '
}

# Build the manifest once (host file -> expected box path), reproducing
# distrobox-init routing: linker libs to the ELF-class bucket, fixed-path
# plugins to the guest-native lib root by class ({LIB32}/{LIB64}, expanded in
# the guest), configs/binaries keep their real path.
manifest="$(mktemp)"
pm_list | sort -u | while read -r f; do
	[ -f "${f}" ] || [ -L "${f}" ] || continue
	is_excluded "${f}" && continue
	# distrobox-init rewrites an absolute library_path in ICD/vendor JSONs to the
	# bare soname, so the guest copy differs from the host original. Checksum the
	# same rewrite here (a no-op for JSONs without an absolute path) so the mirror
	# check compares against what the guest will actually hold.
	case "${f}" in
		*.json) h="$(sed 's@\("library_path"[[:space:]]*:[[:space:]]*"\)/[^"]*/\([^"/]*"\)@\1\2@g' "${f}" 2> /dev/null | sha256sum | cut -d' ' -f1)" ;;
		*) h="$(sha256sum "${f}" 2> /dev/null | cut -d' ' -f1)" ;;
	esac
	[ -n "${h}" ] || continue
	real="${f}"
	[ -L "${f}" ] && real="$(readlink -f "${f}" 2> /dev/null)"
	case "${f}" in
		*/xorg/* | */gbm/* | */vdpau/* | */wine/*)
			# Fixed-path plugin: guest-native lib root by ELF class. The guest
			# check expands {LIB32}/{LIB64}; keep the "<plugin>/..." sub-path.
			sub="${f}"
			case "${sub}" in
				/usr/lib/*-linux-*/*) sub="${sub#/usr/lib/*-linux-*/}" ;;
				/usr/lib64/*) sub="${sub#/usr/lib64/}" ;;
				/usr/lib32/*) sub="${sub#/usr/lib32/}" ;;
				/usr/lib/*) sub="${sub#/usr/lib/}" ;;
				*) ;;
			esac
			if [ "$(elfclass "${real}")" = 1 ]; then bp="{LIB32}/${sub}"; else bp="{LIB64}/${sub}"; fi
			;;
		*)
			base="${f##*/}"
			case "${base}" in
				*nvidia*.so* | *libcuda* | libnvcuvid* | libnvoptix*)
					if [ "$(elfclass "${real}")" = 1 ]; then
						bp="/usr/lib/distrobox-nvidia/lib32/${base}"
					else bp="/usr/lib/distrobox-nvidia/lib64/${base}"; fi
					;;
				*) bp="${f}" ;;
			esac
			;;
	esac
	printf '%s %s\n' "${h}" "${bp}"
done > "${manifest}"
nreq="$(grep -c . "${manifest}")"
[ "${nreq}" -gt 0 ] || fail "empty manifest (install or oracle bug)"
# Coverage floor: the manifest must carry every entry-point stem, else a package
# rename or a find-glob miss would let every guest pass on a hollow file set.
for stem in libcuda.so libnvidia-ml.so libGLX_nvidia.so libEGL_nvidia.so libnvidia-opencl.so; do
	grep -qE "/${stem}[^/]*\$" "${manifest}" || fail "coverage floor: no ${stem}* in manifest (package set or glob changed)"
done
outok && pm_list | sort -u > "${OUT}/package-files.txt"

# Each guest image is a gating check against the same manifest.
bad=""
for box in ${BOX_IMAGES}; do
	name="nvtest-$(printf '%s' "${box##*/}" | sed 's/[:.]/-/g')"
	log "guest ${box}: create --nvidia"
	dbx create --yes --nvidia --image "${box}" --name "${name}" > "${SERIAL}" 2>&1 || fail "create ${box}"
	dbx enter "${name}" -- true > "${SERIAL}" 2>&1 || fail "enter ${box}"
	ok=0
	for _ in $(seq 1 60); do
		dbx enter "${name}" -- test -f /etc/ld.so.conf.d/00-distrobox-nvidia.conf && {
			ok=1
			break
		}
		sleep 2
	done
	[ "${ok}" = 1 ] || fail "${box}: mirror did not complete"

	# shellcheck disable=SC2016  # $path/$want expand in the guest sh
	res="$(dbx enter -T "${name}" -- sh -c '
		l64=/usr/lib; for d in /usr/lib/x86_64-linux-gnu /usr/lib64 /usr/lib; do [ -d "$d" ] && { l64=$d; break; }; done
		case "$l64" in
			*/x86_64-linux-gnu) l32=/usr/lib/i386-linux-gnu ;;
			*/lib64)
				# Fedora keeps 32-bit in /usr/lib; Arch /usr/lib64 -> /usr/lib, so its 32-bit tree is /usr/lib32.
				if [ "$(readlink -m /usr/lib64)" = "$(readlink -m /usr/lib)" ]; then l32=/usr/lib32; else l32=/usr/lib; fi ;;
			*) l32=/usr/lib32 ;;
		esac
		[ "$(readlink -m "$l32")" = "$(readlink -m "$l64")" ] && l32=""
		n=0; okc=0; skip=0
		while read -r want path; do
			n=$((n + 1))
			case "$path" in
				"{LIB32}/"*) [ -n "$l32" ] || { skip=$((skip + 1)); continue; }; path="$l32/${path#"{LIB32}/"}" ;;
				"{LIB64}/"*) path="$l64/${path#"{LIB64}/"}" ;;
			esac
			[ -e "$path" ] || { echo "ABSENT $path"; continue; }
			# Each 64-bit ELF we mirror (bucket libs, plugins, nvidia binaries)
			# must resolve its NEEDED *nvidia* libs through the bucket. Only
			# nvidia deps count - a missing system lib (e.g. libnvidia-pkcs11
			# wanting OpenSSL 1.1) belongs to the guest; non-ELF/PE and 32-bit skip.
			if [ "$(od -An -t x1 -N 4 "$path" 2>/dev/null | tr -d " ")" = 7f454c46 ] &&
				[ "$(od -An -t u1 -j 4 -N 1 "$path" 2>/dev/null | tr -d " ")" = 2 ]; then
				lddout="$(ldd "$path" 2>&1)"; lddrc=$?
				[ "$lddrc" -ne 0 ] && ! printf "%s" "$lddout" | grep -q "not a dynamic executable" && echo "LDDFAIL $path [rc=$lddrc]"
				miss="$(printf "%s\n" "$lddout" | sed -n "s/^[[:space:]]*\(.*\) => not found/\1/p" | grep -iE "nvidia|cuda|nvcuvid|nvoptix" | tr "\n" " ")"
				[ -n "$miss" ] && echo "UNRESOLVED $path [missing: $miss]"
			fi
			got="$(sha256sum "$path" 2>/dev/null | cut -d" " -f1)"
			[ "$got" = "$want" ] && { okc=$((okc + 1)); continue; }
			sz="$(wc -c < "$path" 2>/dev/null || echo "?")"
			real="$(readlink -f "$path" 2>/dev/null || echo "$path")"
			mnt="$(findmnt -no SOURCE "$path" 2>/dev/null || true)"
			own="$(pacman -Qoq "$path" 2>/dev/null || rpm -qf "$path" 2>/dev/null || dpkg -S "$path" 2>/dev/null || true)"
			echo "MISMATCH $path [box ${sz}B ${got} real=$real mnt=${mnt:-none} owner=${own:-unowned}]"
		done
		# Findability: each entry-point soname must resolve THROUGH our bucket.
		# ldd above only proves a mirrored file own NEEDED links; a vendor lib
		# loaded purely by dlopen-by-soname (libGLX_nvidia, ...) is nobody NEEDED.
		for so in libcuda.so.1 libnvidia-ml.so.1 libGLX_nvidia.so.0 libEGL_nvidia.so.0 libnvidia-opencl.so.1; do
			case "$(ldconfig -p 2>/dev/null | grep -F "$so (" | head -1)" in
				*/distrobox-nvidia/*) ;;
				*) echo "UNCACHED $so" ;;
			esac
		done
		echo "READ $n"
		echo "OK $okc"
		echo "SKIP32 $skip"' < "${manifest}")"
	rn="$(printf '%s\n' "${res}" | awk '$1=="READ"{print $2}')"
	[ "${rn:-0}" = "${nreq}" ] || fail "${box}: guest read ${rn:-0}/${nreq} (stdin not forwarded?)"

	# Positive accounting: OK + 32-bit-skip + ABSENT + MISMATCH must equal the
	# lines read, so a file slipping through unclassified cannot hide as a pass.
	okc="$(printf '%s\n' "${res}" | awk '$1=="OK"{print $2}')"
	skc="$(printf '%s\n' "${res}" | awk '$1=="SKIP32"{print $2}')"
	na="$(printf '%s\n' "${res}" | grep -c '^ABSENT ' || true)"
	nm="$(printf '%s\n' "${res}" | grep -c '^MISMATCH ' || true)"
	[ "$((${okc:-0} + ${skc:-0} + na + nm))" = "${nreq}" ] || fail "${box}: accounting off (ok=${okc:-0} skip=${skc:-0} absent=${na} mismatch=${nm} != ${nreq})"

	# Loadability probes (no GPU, no manifest needed - a separate dbx enter fed a
	# heredoc, so the body reads as ordinary shell). Each checks something the
	# per-file pass cannot: that every ICD names a resolvable library, that the
	# self-contained driver libs relocate cleanly, and that the fixed-path plugins
	# sit in their loader's own directory.
	probes="$(
		dbx enter -T "${name}" -- sh << 'PROBE'
soname_path() { ldconfig -p 2>/dev/null | grep -F "$1 (" | head -1 | sed 's/.*=> //'; }

# EGL vendor / OpenCL / EGL-platform configs get the lightweight static check:
# they name a bare soname the loaders find via ld.so, so resolving it in the
# cache is a faithful proxy. Vulkan ICDs get a real load check (below) instead.
icdn=0
for f in $(find /etc /usr/share /usr/lib /usr/lib64 -type f \
	\( -path '*/glvnd/egl_vendor.d/*nvidia*' -o -path '*/OpenCL/vendors/*nvidia*' \
	-o -path '*/egl_external_platform.d/*nvidia*' \) 2>/dev/null); do
	case "$f" in
		*.icd) lib=$(grep -m1 '\.so' "$f" 2>/dev/null | tr -d '[:space:]') ;;
		*) lib=$(tr -d '\n' < "$f" 2>/dev/null | sed -n 's/.*"library_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p') ;;
	esac
	[ -n "$lib" ] || continue
	icdn=$((icdn + 1))
	case "$lib" in
		*/*)
			# Absolute library_path must exist at that path. If it does not but the
			# basename resolves in the cache, the lib was relocated to the bucket and
			# the config points at the pre-mirror absolute path.
			if [ ! -e "$lib" ]; then
				if ldconfig -p 2>/dev/null | grep -qF "${lib##*/} ("; then
					echo "ICD_UNRESOLVED $f $lib (absolute path absent; ${lib##*/} is in the cache - lib relocated to bucket)"
				else
					echo "ICD_UNRESOLVED $f $lib"
				fi
			fi
			;;
		*) [ -n "$(soname_path "$lib")" ] || echo "ICD_UNRESOLVED $f $lib" ;;
	esac
done
echo "ICDN $icdn"

# Vulkan ICDs: drive the real loader instead of guessing. Install vulkaninfo +
# the loader, then let it try to load each mirrored nvidia ICD. Without a GPU the
# library still dlopens (device enumeration is a separate, later step), so a
# "Failed loading library"/"Ignoring this JSON" line naming an nvidia lib means
# its library_path did not resolve. A failed install is a visible SKIP, never a
# silent pass. sudo -n avoids a hang if the guest lacks passwordless sudo.
if ! command -v vulkaninfo > /dev/null 2>&1; then
	{
		if command -v apt-get > /dev/null 2>&1; then
			timeout 300 sudo -n apt-get update -qq && timeout 300 sudo -n apt-get install -y -qq vulkan-tools
		elif command -v dnf > /dev/null 2>&1; then
			timeout 300 sudo -n dnf install -y -q vulkan-tools vulkan-loader
		elif command -v pacman > /dev/null 2>&1; then
			# Full upgrade (not -Sy): the arch toolbox image can lag the repos, and a
			# partial upgrade can leave the loader/libs inconsistent.
			timeout 300 sudo -n pacman -Syu --noconfirm --needed vulkan-tools vulkan-icd-loader
		fi
	} > /dev/null 2>&1
fi
if command -v vulkaninfo > /dev/null 2>&1; then
	nicd=$(find /etc /usr/share -type f -path '*/vulkan/icd.d/*nvidia*' 2>/dev/null | wc -l)
	vkraw=$(VK_LOADER_DEBUG=warn,error timeout 60 vulkaninfo 2>&1)
	vkfail=$(printf '%s\n' "$vkraw" | grep -iE 'Ignoring this JSON|Failed loading library' | grep -i nvidia)
	if [ -n "$vkfail" ]; then
		# Classify the rejection. A missing *nvidia* lib means the mirror dropped
		# something -> hard fail. A missing guest lib (libGLdispatch/libX11, the GL
		# stack the image itself must ship) is not distrobox's job -> SKIP, named.
		# The lib loading but nvidia declining to return vkCreateInstance is the
		# driver refusing to initialise in this guest, downstream of the mirror
		# (ubuntu/fedora negotiate the same mirrored driver fine) -> SKIP. Only a
		# truly unattributable failure stays a hard fail.
		miss=$(printf '%s\n' "$vkraw" | sed -n 's/.* \([^ ]*\): cannot open shared object.*/\1/p' | sort -u)
		nvmiss=$(printf '%s\n' "$miss" | grep -iE 'nvidia|cuda' | tr '\n' ' ')
		if [ -n "$nvmiss" ]; then
			echo "ICD_LOAD_FAILED nvidia lib(s) unresolved: $nvmiss"
		elif [ -n "$miss" ]; then
			echo "SKIP icd-vulkan (nvidia ICD present, but guest image lacks: $(printf '%s' "$miss" | tr '\n' ' '))"
		elif printf '%s\n' "$vkraw" | grep -qi 'vk_icdGetInstanceProcAddr'; then
			echo "SKIP icd-vulkan (nvidia ICD loaded, driver declined to init - no vkCreateInstance; not a mirror fault)"
		else
			printf '%s\n' "$vkfail" | sed -n 's/.*ICD JSON \(.*\)\. Ignoring.*/\1/p' | sort -u | while read -r badlib; do echo "ICD_LOAD_FAILED $badlib"; done
		fi
		# Full loader failure chain (nvidia libs + the dlopen "cannot open" root
		# cause) for the OUT diagnostics, so the exact failing dependency is on
		# record rather than only the classified token.
		printf '%s\n' "$vkraw" | grep -iE 'nvidia|cannot open|Ignoring this JSON|Failed loading|undefined symbol|GLdispatch|glvnd' | while read -r l; do echo "VKLOG $l"; done
	fi
	echo "ICDVK $nicd"
else
	echo "SKIP icd-vulkan (vulkaninfo unavailable / install failed)"
fi

# Symbol binding: self-contained driver libs must relocate cleanly. ldd -r forces
# function+data relocation, catching a newer-host-driver vs older-guest-glibc
# undefined-symbol break that plain ldd passes. GLVND vendor libs are excluded -
# they leave dispatch symbols for libGLdispatch to supply.
for so in libcuda.so.1 libnvidia-ml.so.1 libnvidia-encode.so.1 libnvcuvid.so.1; do
	p=$(soname_path "$so")
	[ -n "$p" ] || continue
	und=$(ldd -r "$p" 2>&1 | sed -n 's/^[[:space:]]*undefined symbol: \(.*\)/\1/p' | tr '\n' ' ')
	[ -n "$und" ] && echo "UNLOADABLE $so [undefined: $und]"
done

# Plugin colocation: Xorg/GBM/VDPAU drivers are dlopen'd from the consumer
# library's own <libdir>/<loader> dir, so the correct location is defined by the
# guest consumer, not the mirror. Anchor to the 64-bit consumer and check only
# 64-bit plugins - a 32-bit plugin lives in its own tree matched by a 32-bit
# loader - and canonicalise both sides so usr-merge/multiarch symlinks compare
# equal and overlapping find roots collapse.
for pair in libgbm.so.1:gbm libvdpau.so.1:vdpau; do
	consumer=${pair%%:*}
	sub=${pair#*:}
	cp=$(ldconfig -p 2>/dev/null | grep -F "$consumer (" | grep -F ',x86-64)' | head -1 | sed 's/.*=> //')
	[ -n "$cp" ] || { echo "SKIP plugin-$sub (guest has no 64-bit $consumer)"; continue; }
	exp=$(readlink -m "$(dirname "$cp")/$sub")
	for pl in $(find /usr/lib /usr/lib64 -type f -path "*/$sub/*nvidia*" 2>/dev/null | xargs -r readlink -f | sort -u); do
		[ "$(od -An -t u1 -j 4 -N 1 "$pl" 2>/dev/null | tr -d ' ')" = 2 ] || continue
		[ "$(readlink -m "$(dirname "$pl")")" = "$exp" ] || echo "PLUGIN_MISPLACED $pl [expected in $exp]"
	done
done
PROBE
	)"
	printf '%s\n' "${probes}" | grep '^SKIP ' | while read -r sk; do log "guest ${box}: ${sk}"; done
	icdn="$(printf '%s\n' "${probes}" | awk '$1=="ICDN"{print $2}')"
	vkicd="$(printf '%s\n' "${probes}" | awk '$1=="ICDVK"{print $2}')"
	log "guest ${box}: probes - ${icdn:-0} egl/cl config(s), ${vkicd:-0} vulkan icd(s) load-checked"

	probs="$(printf '%s\n' "${res}" "${probes}" | grep -E '^(ABSENT|MISMATCH|UNRESOLVED|LDDFAIL|UNCACHED|ICD_UNRESOLVED|ICD_LOAD_FAILED|UNLOADABLE|PLUGIN_MISPLACED) ' || true)"
	n="$(printf '%s\n' "${probs}" | grep -c . || true)"
	if outok; then
		printf '%s\n' "${probs}" | grep . > "${OUT}/failures-${name}.txt" 2> /dev/null || true
		# Raw Vulkan loader failure chain (only when the nvidia ICD did not load),
		# so the exact dlopen root cause is on record, not just the token.
		vklog="$(printf '%s\n' "${probes}" | sed -n 's/^VKLOG //p')"
		[ -z "${vklog}" ] || printf '%s\n' "${vklog}" > "${OUT}/vulkan-loader-${name}.txt"
		# Inventory for manual review: the bucket plus the fixed-path plugins
		# wherever they landed (nvidia-named, so guest Mesa backends stay out).
		dbx enter "${name}" -- sh -c '
			find /usr/lib/distrobox-nvidia -type f 2>/dev/null
			find /usr/lib /usr/lib32 /usr/lib64 /usr/lib/*-linux-gnu -type f \
				\( -path "*/gbm/*nvidia*" -o -path "*/vdpau/*nvidia*" -o -path "*/xorg/*nvidia*" -o -path "*/nvidia/wine/*" \) 2>/dev/null
		' | sort -u > "${OUT}/box-files-${name}.txt" || true
	fi
	if [ "${n}" -gt 0 ]; then
		log "guest ${box}: ${n} problem(s) (of ${nreq} files)"
		bad="${bad} ${box}"
	else log "guest ${box}: OK (${nreq} files)"; fi
done

[ -z "${bad}" ] || fail "guest images failed:${bad}"
log "all guests passed (${nreq} files each)"
pass
