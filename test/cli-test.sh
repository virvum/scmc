#!/bin/sh

set -xe

tmpdir=$(mktemp --directory)
fifo=$(mktemp --dry-run)
mkfifo $fifo

cleanup() {
	rm -rf $tmpdir
	rm -f $fifo
}

trap cleanup EXIT

(
	tail -f $fifo \
		| go run cmd/scmc/*.go cli 2>&1 \
		| sed "s/.*/$(printf '\033[1;34m&\033[0m')/"
) &
clipid=$!

rdir=/$(basename "$(mktemp --dry-run)")
echo ls > $fifo
echo put test > $fifo
echo quit > $fifo

wait
echo >&2
echo 'Tests successfully finished.' >&2
