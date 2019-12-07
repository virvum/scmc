#!/bin/sh

set -xe

config="$HOME"/.scmc.yaml
username=$(python -c "import yaml; print(yaml.safe_load(open('$config'))['username'])")
password=$(python -c "import yaml; print(yaml.safe_load(open('$config'))['password'])")
api_url="http://$username:$password@127.0.0.1:9000"
backup_dir=restic-test-$(basename $(mktemp --dry-run))
repo_url="rest:$api_url/$backup_dir"

scmc restic-rest-server &
scmcpid=$!

export RESTIC_PASSWORD="$(dd if=/dev/urandom ibs=32 count=1 2>/dev/null | base64 -w 0)"
export RESTIC_REPOSITORY="$repo_url"

restic() {
	command restic --verbose=2 --no-cache "$@"
}

rand_files() {
	mkdir -p "$tmpdir"/a
	for f in {a..z}
	do
		head -20 /dev/urandom > "$tmpdir"/a/$f
	done
}

hash() {
	sha256sum "$1" | awk '{ print $1 }'
}

cleanup() {
	curl -vXDELETE "$api_url/$backup_dir/"
	kill $scmcpid
	rm -rf "$tmpdir"
}

tmpdir=$(mktemp -d)
trap cleanup EXIT

restic init
restic backup Makefile

for i in 1 2 3 4
do
	rand_files
	restic backup "$tmpdir"/a
	rand_files
	restic backup "$tmpdir"/a
	rand_files
	restic backup "$tmpdir"/a

	restic check

	mv "$tmpdir"/a "$tmpdir"/b

	restic snapshots
	restic restore --verify --target / latest

	for f in {a..z}
	do
		a="$(hash "$tmpdir"/a/$f)"
		b="$(hash "$tmpdir"/b/$f)"
		if test "$a" != "$b"
		then
			echo 'Hashes do not match!'
			exit 1
		fi
	done

	rm -rf "$tmpdir"/{a,b}
done
