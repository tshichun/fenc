#!/bin/bash -e

if [ -n "$1" ]; then
    if [ "$1" != "exe" -a "$1" != "mac" ]; then
        echo "Usage: $0 [exe|mac]"
        exit 1
    fi
    EXT=\.$1
else
    EXT=
fi

go build -ldflags "-s -w" -o dist/fenc$EXT
