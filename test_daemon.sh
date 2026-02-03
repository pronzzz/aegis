#!/bin/bash
export AEGIS_PASSPHRASE=secret123
./aegis init
./aegis start --config config.json &
PID=$!
sleep 5
kill $PID
