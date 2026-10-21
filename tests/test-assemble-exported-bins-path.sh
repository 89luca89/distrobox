#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
#
# Exercise manifest export-path handling with disposable local command stubs.
# Set DISTROBOX_TEST_TMPDIR to a task-local directory before running this test.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
: "${DISTROBOX_TEST_TMPDIR:?set DISTROBOX_TEST_TMPDIR to a task-local directory}"
mkdir -p "${DISTROBOX_TEST_TMPDIR}"
test_root=$(mktemp -d "${DISTROBOX_TEST_TMPDIR%/}/distrobox-assemble-issue-2237.XXXXXX")
trap 'rm -rf "$test_root"' 0 HUP INT TERM

bin_dir="${test_root}/bin"
home_dir="${test_root}/home"
mkdir -p "${bin_dir}" "${home_dir}"

# The copied script must not source real system or user distrobox configuration.
awk '
	/^config_files="/ {
		print "config_files=\"\""
		skipping = 1
		found = 1
		next
	}
	skipping && /^"/ {
		skipping = 0
		next
	}
	!skipping { print }
	END { if (!found) exit 1 }
' "${repo_root}/distrobox-assemble" > "${bin_dir}/distrobox-assemble"
chmod +x "${bin_dir}/distrobox-assemble"

cat > "${bin_dir}/distrobox-list" <<'EOF'
#!/bin/sh
exit 0
EOF
cat > "${bin_dir}/distrobox-create" <<'EOF'
#!/bin/sh
exit 0
EOF
cat > "${bin_dir}/distrobox" <<'EOF'
#!/bin/sh
printf '%s\n' "$@" >> "${DISTROBOX_TEST_LOG}"
exit 0
EOF
chmod +x "${bin_dir}/distrobox-list" "${bin_dir}/distrobox-create" "${bin_dir}/distrobox"
PATH="${bin_dir}:${PATH}"
test_shell="${DISTROBOX_TEST_SHELL:-sh}"
export PATH

run_manifest() {
	log_file="$1"
	manifest_file="$2"
	: > "${log_file}"
	(
		unset SUDO_USER DOAS_USER DBX_VERBOSE
		HOME="${home_dir}" \
		USER="fixture-user" \
		SHELL="/bin/sh" \
		XDG_CONFIG_HOME="${test_root}/xdg" \
		TMPDIR="${test_root}" \
		DISTROBOX_TEST_LOG="${log_file}" \
			"${test_shell}" "${bin_dir}/distrobox-assemble" create --file "${manifest_file}" > /dev/null
	)
}

assert_logged_path() {
	expected="$1"
	log_file="$2"
	if ! grep -F -x "${expected}" "${log_file}" > /dev/null; then
		printf >&2 'expected export path not forwarded: %s\n' "${expected}"
		return 1
	fi
}

blank_manifest="${test_root}/with-trailing-blank-lines.ini"
printf '%s\n' \
	'[my-container]' \
	'image=archlinux:latest' \
	'additional_packages="rsync"' \
	'exported_bins="/usr/bin/rsync"' \
	'exported_bins_path="~/.local/bin"' \
	'' \
	'' > "${blank_manifest}"
blank_log="${test_root}/blank-args.log"
run_manifest "${blank_log}" "${blank_manifest}"
assert_logged_path "${home_dir}/.local/bin" "${blank_log}"

unterminated_manifest="${test_root}/without-final-newline.ini"
printf '%s\n' \
	'[my-container]' \
	'image=archlinux:latest' \
	'exported_bins="/usr/bin/rsync"' > "${unterminated_manifest}"
printf '%s' 'exported_bins_path="~/custom-bin"' >> "${unterminated_manifest}"
unterminated_log="${test_root}/unterminated-args.log"
run_manifest "${unterminated_log}" "${unterminated_manifest}"
assert_logged_path "${home_dir}/custom-bin" "${unterminated_log}"

printf 'assemble exported-bins path tests passed\n'
