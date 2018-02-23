#!/usr/bin/env bash

wget --spider http://google.com 2>&1

if [ $? -eq 0 ]; then
    printf 'Skipping WiFi Connect\n'
else
    printf 'Starting WiFi Connect\n'
    /wifi/wifi-connect
fi

/yuna