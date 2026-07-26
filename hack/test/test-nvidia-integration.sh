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
# hack/test/test-nvidia-integration.sh <ubuntu|fedora|arch|debian>
#
# distrobox --nvidia integration test. Builds distrobox, stages the host binary
# and the in-VM script (nvidia-test.sh, as the run.sh entry point vm-run.sh
# executes) into a share, then hands it to the generic vm-run.sh which boots a
# throwaway <distro> VM and waits for the result. The in-VM logic (install the
# driver, create --nvidia for each guest image, checksum every mirrored file)
# lives in nvidia-test.sh. Diagnostics land in
# ./last-out-<distro>/.
#

DISTRO="${1:?usage: test-nvidia-integration.sh <ubuntu|fedora|arch|debian>}"
HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(cd "${HERE}/../.." && pwd)"
OUT="${HERE}/out/last-out-${DISTRO}"

msg()
{
	printf '\033[1;34m==>\033[0m %s\n' "$*" >&2
}

msg "building distrobox"
make -C "${REPO}" build >&2

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT INT TERM

# Read-only share into the VM: the built binary, the in-VM script (as run.sh) and
# the host distro (nvidia-test.sh reads it from /mnt/share/distro).
share="${work}/share"
mkdir -p "${share}"
install -m0755 "${REPO}/bin/distrobox" "${share}/distrobox"
install -m0755 "${HERE}/nvidia-test.sh" "${share}/run.sh"
printf '%s\n' "${DISTRO}" > "${share}/distro"

rm -rf "${OUT}"
mkdir -p "${OUT}"

rc=0
"${HERE}/vm-run.sh" "${DISTRO}" "${share}" "${OUT}" || rc=$?

for ff in "${OUT}"/failures-*.txt; do
	[ -s "${ff}" ] || continue
	printf '\n\033[1m--- %s ---\033[0m\n' "${ff##*/}" >&2
	cat "${ff}" >&2
done
msg "review: ${OUT}/ (package-files.txt, box-files-*.txt, failures-*.txt, serial.log)"
exit "${rc}"
