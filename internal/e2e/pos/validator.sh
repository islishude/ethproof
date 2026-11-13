#!/bin/sh
set -eu
if [ ! -f /validators/.initialized ]; then
  if [ -n "$(ls -A /validators)" ]; then
    echo 'Incomplete validator initialization; reset this local testnet explicitly' >&2
    exit 1
  fi
  cp -R /network/validator/keys/. /validators/
  touch /validators/.initialized
  set -- --init-slashing-protection "$@"
fi
exec lighthouse --testnet-dir=/network/metadata vc --validators-dir=/validators \
  --secrets-dir=/network/validator/secrets --beacon-nodes=http://lighthouse-1:5052 \
  --suggested-fee-recipient=0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266 \
  --graffiti=ethproof-pos-e2e "$@"
