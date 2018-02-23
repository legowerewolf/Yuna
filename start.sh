#!/usr/bin/env bash

# export DBUS_SYSTEM_BUS_ADDRESS=unix:path=/host/run/dbus/system_bus_socket

wget --spider http://google.com 2>&1


if [ $? -eq 0 ]; then
    printf 'Skipping WiFi Connect\n'
else
    printf 'Starting WiFi Connect\n'
    /wifi/wifi-connect
fi

# Start your application here.
/yuna