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
set -eu
#
# Generic throwaway-VM runner for the hack/test suite.
#
# Boots a cloud image of <distro> with <share-dir> mounted read-only at
# /mnt/share (9p) and <out-dir> read-write at /mnt/out, runs /mnt/share/run.sh,
# and waits for it to print "=== RESULT: PASS|FAIL ===" on the serial console.
# Every run boots a byte-fresh copy of the pristine (cached, read-only) base
# image, so the payload cannot corrupt state between runs.
#
# usage: vm-run.sh <ubuntu|fedora|arch|debian> <share-dir> <out-dir>
#        (<share-dir> must contain an executable run.sh entry point)
# exit:  0 = PASS, 1 = FAIL, 2 = infra error / no result before the timeout
#

DISTRO="${1:?usage: vm-run.sh <distro> <share-dir> <out-dir>}"
SHARE="${2:?share dir required}"
OUT="${3:?out dir required}"

case "${DISTRO}" in
	ubuntu) IMG_URL="https://cloud-images.ubuntu.com/resolute/current/resolute-server-cloudimg-amd64.img" ;;
	fedora) IMG_URL="https://download.fedoraproject.org/pub/fedora/linux/releases/44/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-44-1.7.x86_64.qcow2" ;;
	arch) IMG_URL="https://geo.mirror.pkgbuild.com/images/latest/Arch-Linux-x86_64-cloudimg.qcow2" ;;
	debian) IMG_URL="https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2" ;;
	*)
		printf 'unknown distro: %s\n' "${DISTRO}" >&2
		exit 2
		;;
esac

MEM=4096
CPUS=4
TIMEOUT=2400
CACHE="${XDG_CACHE_HOME:-${HOME}/.cache}/distrobox-vmtest"

msg()
{
	printf '\033[1;34m==>\033[0m %s\n' "$*" >&2
}
bail()
{
	printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2
	exit 2
}

command -v cloud-localds > /dev/null || bail "need cloud-localds (from cloud-image-utils)"
command -v curl > /dev/null || bail "need curl"
command -v qemu-img > /dev/null || bail "need qemu-img"
command -v qemu-system-x86_64 > /dev/null || bail "need qemu-system-x86_64"
[ -f "${SHARE}/run.sh" ] || bail "share dir has no run.sh entry point: ${SHARE}"

mkdir -p "${CACHE}"
base="${CACHE}/${DISTRO}-${IMG_URL##*/}"
if [ ! -f "${base}" ]; then
	msg "downloading ${DISTRO} cloud image"
	curl -fSL -o "${base}.part" "${IMG_URL}" || bail "download failed: ${IMG_URL}"
	mv "${base}.part" "${base}"
	chmod 0444 "${base}"
fi

work="$(mktemp -d)"
pidfile="${work}/pid"
serial="${work}/serial"
tailpid=""

# shellcheck disable=SC2317,SC2329  # cleanup runs via the EXIT/INT/TERM trap
cleanup()
{
	if [ -n "${tailpid}" ]; then kill "${tailpid}" 2> /dev/null || true; fi
	if [ -f "${pidfile}" ]; then kill "$(cat "${pidfile}")" 2> /dev/null || true; fi
	if [ -f "${serial}" ]; then cp -f "${serial}" "${OUT}/serial.log" 2> /dev/null || true; fi
	rm -rf "${work}"
}
trap cleanup EXIT INT TERM

disk="${work}/disk.qcow2"
cp --reflink=auto "${base}" "${disk}"
chmod u+w "${disk}"
qemu-img resize "${disk}" +20G > /dev/null

# A fixed instance-id is fine: the disk is always a fresh copy of the pristine
# base, so cloud-init sees a clean instance and re-provisions every boot.
printf 'instance-id: dbxtest\nlocal-hostname: dbxtest\n' > "${work}/meta-data"
cat > "${work}/user-data" << 'EOF'
#cloud-config
runcmd:
  - [ sh, -c, "modprobe 9pnet_virtio 9p 2>/dev/null || true" ]
  - [ mkdir, -p, /mnt/share, /mnt/out ]
  - [ sh, -c, "mount -t 9p -o trans=virtio,version=9p2000.L,ro dbxshare /mnt/share" ]
  - [ sh, -c, "mount -t 9p -o trans=virtio,version=9p2000.L,rw dbxout /mnt/out || true" ]
  - [ sh, -c, "sh /mnt/share/run.sh || { echo '=== RESULT: FAIL ===' > /dev/ttyS0; poweroff -f; }" ]
EOF

cloud-localds "${work}/seed.img" "${work}/user-data" "${work}/meta-data"

# Only the acceleration flags vary, so only they go through $@ (POSIX sh has no
# arrays); the rest of the command is inline with every path quoted.
if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
	set -- -enable-kvm -cpu host
else
	set -- -cpu max
fi

msg "booting ${DISTRO} VM (timeout ${TIMEOUT}s)"
qemu-system-x86_64 "$@" \
	-m "${MEM}" -smp "${CPUS}" \
	-drive "if=virtio,format=qcow2,file=${disk}" \
	-drive "if=virtio,format=raw,file=${work}/seed.img" \
	-fsdev "local,id=s,path=${SHARE},security_model=none,readonly=on" -device virtio-9p-pci,fsdev=s,mount_tag=dbxshare \
	-fsdev "local,id=o,path=${OUT},security_model=none" -device virtio-9p-pci,fsdev=o,mount_tag=dbxout \
	-netdev user,id=n -device virtio-net-pci,netdev=n \
	-display none -serial "file:${serial}" -monitor none -no-reboot -pidfile "${pidfile}" -daemonize < /dev/null

touch "${serial}"

# Strip CSI escapes from the live console dump
tail -f "${serial}" | sed -u 's/\x1b\[[0-9;?]*[[:alpha:]]//g' >&2 &
tailpid=$!

result=""
elapsed=0
while [ "${elapsed}" -lt "${TIMEOUT}" ]; do
	if grep -q 'RESULT:' "${serial}" 2> /dev/null; then
		result="$(grep -o 'RESULT: [A-Z]*' "${serial}" | tail -1 | awk '{print $2}')"
		break
	fi
	if [ -f "${pidfile}" ] && ! kill -0 "$(cat "${pidfile}")" 2> /dev/null; then break; fi
	sleep 5
	elapsed=$((elapsed + 5))
done
kill "${tailpid}" 2> /dev/null || true
tailpid=""
if [ -z "${result}" ]; then result="$(grep -o 'RESULT: [A-Z]*' "${serial}" 2> /dev/null | tail -1 | awk '{print $2}')"; fi

case "${result}" in
	PASS)
		msg "VM result: PASS (${DISTRO})"
		exit 0
		;;
	FAIL)
		msg "VM result: FAIL (${DISTRO})"
		exit 1
		;;
	*) bail "no result after ${TIMEOUT}s (see ${OUT}/serial.log)" ;;
esac
