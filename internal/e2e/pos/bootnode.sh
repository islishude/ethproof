#!/bin/sh
set -eu
for attempt in $(seq 1 120); do
  if curl -fsS --max-time 2 http://lighthouse-1:5052/eth/v1/node/identity > /tmp/identity.json; then
    jq -er '.data.enr' /tmp/identity.json > /data/bootnode.enr
    exit 0
  fi
  sleep 1
done
echo 'Beacon bootstrap timed out' >&2
exit 1
