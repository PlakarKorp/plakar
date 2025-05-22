#!/bin/sh

plakar at http://host.docker.internal:9888 mount >/dev/null 2>&1

exec "$@"