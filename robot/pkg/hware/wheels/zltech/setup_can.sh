#!/usr/bin/env bash

set -euo pipefail

# Setup a SocketCAN interface with sane defaults for ZLTech wheels.
# Defaults: IF=can0, BITRATE=500000, RESTART_MS=100, TXQLEN=2048, LOOPBACK=off
# Usage: ./setup_can.sh [IF] [BITRATE]

IFACE="${1:-can0}"
BITRATE="${2:-500000}"
RESTART_MS="${RESTART_MS:-100}"
TXQLEN="${TXQLEN:-2048}"

echo "[setup_can] configuring ${IFACE} @ ${BITRATE} bps, restart-ms=${RESTART_MS}, loopback=off, txqueuelen=${TXQLEN}"

# Bring interface down if it exists
if ip link show "${IFACE}" >/dev/null 2>&1; then
  sudo ip link set "${IFACE}" down || true
else
  echo "[setup_can] warning: interface ${IFACE} not found yet; proceeding to configure type anyway"
fi

# Configure CAN type with restart-ms and loopback off
sudo ip link set "${IFACE}" type can bitrate "${BITRATE}" restart-ms "${RESTART_MS}" loopback off

# Increase TX queue length to reduce ENOBUFS under burst loads
sudo ip link set "${IFACE}" txqueuelen "${TXQLEN}"

# Bring interface up
sudo ip link set "${IFACE}" up

echo "[setup_can] link status:"
ip -details -statistics link show "${IFACE}"

echo "[setup_can] done"


