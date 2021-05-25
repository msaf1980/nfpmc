#!/bin/bash

FPM="./nfpmc"

die() {
    echo "$2" >&2
    exit $1
}

GIT_VERSION="$(git describe --always --tags --abbrev=6)" && {
    set -f; IFS='-' ; set -- ${GIT_VERSION}
    VERSION=$1; [ -z "$3" ] && RELEASE=$2 || RELEASE=$2.$3
    set +f; unset IFS

    [ "$RELEASE" == "" -a "$VERSION" != "" ] && RELEASE=0 

    if echo $VERSION | egrep '^v[0-9]+\.[0-9]+(\.[0-9]+)?$' >/dev/null; then
      VERSION=${VERSION:1:${#VERSION}}
      printf "'%s' '%s'\n" "$VERSION" "$RELEASE"
    fi
} || {
    exit 1
}

make || exit 1
./nfpmc -s dir -t rpm -n nfpmc \
    -v "${VERSION}" -i "${RELEASE}" \
    -p NAME-VERSION.ITERATION.ARCH.rpm \
    --description "nfpmc: RPM/DEB/APK package builder" \
    --license MIT \
    --url "https://github.com/msaf1980/nfpmc" \
    nfpmc=/usr/bin/nfpmc || die 1 "Can't create package!"

die 0 "Success"
