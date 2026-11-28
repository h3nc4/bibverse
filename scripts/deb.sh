#!/bin/sh

# Builds dist/bibverse_<version>_<arch>.deb for the architecture given, or the host's. The binary is
# static, so one build installs on every Debian and Devuan release.

set -eu

cd "$(dirname "$0")/../"

umask 022

arch="${1:-$(dpkg --print-architecture)}"

# A tagged commit is that release, and any other is a snapshot after the last tag, sorted by the
# UTC commit time and pointing at its source by the hash.
commit="$(git rev-parse --short=8 HEAD)"
date="$(git log -1 --format=%cD)"
if tag="$(git describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null)"; then
	version="${tag#v}"
else
	tag="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || echo v0.0.0)"
	stamp="$(TZ=UTC git log -1 --format=%cd --date=format-local:%Y%m%d%H%M%S)"
	version="${tag#v}+git${stamp}.${commit}"
fi
deb="dist/bibverse_${version}_${arch}.deb"

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
pkg="${work}/pkg"
doc="${pkg}/usr/share/doc/bibverse"
mkdir -p "${pkg}/usr/bin" "${pkg}/usr/share/man/man1" "${doc}" "${pkg}/DEBIAN" dist

CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -buildvcs=false -trimpath \
	-ldflags="-s -w" -o "${pkg}/usr/bin/bibverse" ./cmd/bibverse
gzip -9nc bibverse.1 >"${pkg}/usr/share/man/man1/bibverse.1.gz"
mkdir -p "${pkg}/usr/share/lintian/overrides"
printf '%s\n' \
	'# Static on purpose, so it runs on a minimal host or an empty container with its own TLS roots.' \
	'bibverse: statically-linked-binary [usr/bin/bibverse]' \
	>"${pkg}/usr/share/lintian/overrides/bibverse"

printf '%s\n' \
	'Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/' \
	'Upstream-Name: bibverse' \
	'Source: https://github.com/h3nc4/bibverse' \
	'' \
	'Files: *' \
	'Copyright: 2026 Henrique Almeida <me@h3nc4.com>' \
	'License: GPL-3+' \
	' On Debian systems, the full text of the GNU General Public License version 3' \
	' can be found in /usr/share/common-licenses/GPL-3.' \
	>"${doc}/copyright"
printf '%s\n' \
	"bibverse (${version}) unstable; urgency=medium" \
	'' \
	"  * Build of commit ${commit}." \
	'' \
	" -- Dreamweaver <me@h3nc4.com>  ${date}" |
	gzip -9n >"${doc}/changelog.gz"

size="$(du -sk "${pkg}/usr")"
printf '%s\n' \
	'Package: bibverse' \
	"Version: ${version}" \
	"Architecture: ${arch}" \
	'Maintainer: Dreamweaver <me@h3nc4.com>' \
	"Installed-Size: ${size%%[[:space:]]*}" \
	'Section: utils' \
	'Priority: optional' \
	'Homepage: https://github.com/h3nc4/bibverse' \
	"Description: print the verse before today's Gospel" \
	' bibverse reads the USCCB daily readings and prints the acclamation sung' \
	" before the Gospel at today's Catholic Mass as one line, with its reference." \
	>"${pkg}/DEBIAN/control"
(cd "${pkg}" && find usr -type f -exec md5sum {} +) >"${pkg}/DEBIAN/md5sums"

dpkg-deb --root-owner-group --build "${pkg}" "${deb}"
dpkg-deb --info "${deb}"
