#!/usr/bin/env bash

set -euo pipefail

# Rebuild and reload gs_usb for Jetson kernels, then clean up the builder script.
# Requires internet access.

echo "[gs_usb] downloading builder script"
curl -sSLO https://raw.githubusercontent.com/lucianovk/jetson-gs_usb-kernel-builder/main/jetson-gs_usb-kernel-builder.sh
chmod +x jetson-gs_usb-kernel-builder.sh

echo "[gs_usb] building module for kernel $(uname -r)"
sudo ./jetson-gs_usb-kernel-builder.sh

echo "[gs_usb] reloading module"
sudo modprobe -r gs_usb || true
sudo modprobe gs_usb

echo "[gs_usb] loaded module info:"
modinfo gs_usb | egrep 'filename|vermagic|srcversion'

echo "[gs_usb] cleaning up builder script"
rm -f jetson-gs_usb-kernel-builder.sh || true

echo "[gs_usb] done"


