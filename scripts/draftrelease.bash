#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
echo "DIR is $DIR"
cd "${DIR}"

# We already have set -u, but want to fail early if a required variable is not set.
: ${RELEASE_MINISIGN_PRIVATE_KEY}
: ${RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD}
# However, if you are already logged in for GitHub CLI locally, you can remove this line when running it locally.
: ${GH_TOKEN}

if [[ "${VERSION}" == v* ]]; then
  echo "error: VERSION ${VERSION} must not start with 'v'" >&2
  exit 1
fi

make release
unset RELEASE_MINISIGN_PRIVATE_KEY
unset RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD


# The second v${VERSION} is the tag, see https://cli.github.com/manual/gh_release_create
url=$(gh release create --draft --notes "replace me" --title "v${VERSION}" "v${VERSION}" .build/release/bufisk/assets/*)

echo "Release ${VERSION} has been drafted: ${url}"
