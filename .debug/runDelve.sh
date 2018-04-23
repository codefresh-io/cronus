#!/bin/bash

# run debugger
dlv debug --listen=localhost:2345 --headless=true --log=true ./cmd -- --log-level=debug --json=false  server --store=/var/tmp/events.db
# --store="$TELEPRESENCE_ROOT$STORE_FILE"