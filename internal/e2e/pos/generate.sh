#!/bin/bash
set -euo pipefail
if [ -f /data/ready ]; then exit 0; fi
# Public development keys only. This isolated chain must never hold real funds.
export PRESET_BASE=mainnet CHAIN_ID=31337 NUMBER_OF_VALIDATORS=64
export SLOT_DURATION_MS=3000 SLOT_DURATION_IN_SECONDS=3
export GENESIS_TIMESTAMP=$(($(date +%s) + 60)) GENESIS_DELAY=0
export GENESIS_GASLIMIT=60000000
export EL_AND_CL_MNEMONIC='test test test test test test test test test test test junk'
export BEACON_STATIC_ENR=''
export GLOAS_FORK_EPOCH=18446744073709551615 HEZE_FORK_EPOCH=18446744073709551615
export BPO_1_EPOCH=18446744073709551615 BPO_2_EPOCH=18446744073709551615
if ! /work/entrypoint.sh all > /data/generation.log 2>&1; then
  tail -n 80 /data/generation.log >&2
  exit 1
fi
printf '[]\n' > /data/metadata/boot_enr.yaml
rm -f /data/metadata/bootstrap_nodes.txt
eth2-val-tools keystores --insecure --source-min=0 --source-max=64 \
  --source-mnemonic="$EL_AND_CL_MNEMONIC" --out-loc=/data/validator > /data/keystores.log 2>&1
rm -rf /data/parsed /data/metadata/mnemonics.yaml
touch /data/ready
