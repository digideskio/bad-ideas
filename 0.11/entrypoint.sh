#!/bin/bash
set -e

if [ "${1:0:1}" = '-' ]; then
    set -- chronograf "$@"
fi

if [ "$1" = 'chronograf' ]; then
    set -- "$@"
fi

exec "$@"
