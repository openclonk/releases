#!/usr/bin/env bash

error() {
	echo error: "$@"
	exit 1
}

revision=$(git rev-parse --short HEAD)
[[ -n $revision ]] || error "could not get git revision"

branch=$(git symbolic-ref --short HEAD)
[[ -n $branch ]] || error "could not get branch name"

date=$(date --date="$(git show -s --format=%ci)" -u -Is | sed 's/+00:00$/Z/')
[[ -n $date ]] || error "could not get commit date"

file=${1:?no file to upload given}
: ${OC_REL_URL:?target URL not set}

curl -XPOST "$OC_REL_URL/snapshots/$date-$branch-$revision/$(basename "$file")" --data-binary "@$file" \
	|| error "file upload failed"
