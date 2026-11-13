#!/bin/sh
set -eu
if [ "$NODE_INDEX" != 1 ]; then
  set -- --boot-nodes="$(cat /network/bootnode.enr)" "$@"
fi
exec lighthouse --testnet-dir=/network/metadata bn --datadir=/beacon \
  --execution-endpoint="http://reth-${NODE_INDEX}:8551" \
  --execution-jwt=/network/jwt/jwtsecret \
  --http --http-address=0.0.0.0 --http-port=5052 \
  --disable-upnp --enable-private-discovery --disable-packet-filter \
  --disable-enr-auto-update --enr-address="lighthouse-${NODE_INDEX}" \
  --target-peers=2 --subscribe-all-subnets --supernode \
  --allow-insecure-genesis-sync \
  --suggested-fee-recipient=0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266 "$@"
