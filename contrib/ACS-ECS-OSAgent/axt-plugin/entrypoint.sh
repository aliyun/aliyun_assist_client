#!/bin/sh
OSAGENT_INSTALL_DIR="${0%/*}"
if [ X"$OSAGENT_INSTALL_DIR" = X"$0" ]; then
    # No slash means locating in current working directory
    OSAGENT_INSTALL_DIR="."
fi

export OSAGENT_NODE=$OSAGENT_INSTALL_DIR/bin/node
exec $OSAGENT_NODE $OSAGENT_NODE_ARGS $OSAGENT_INSTALL_DIR/dist/cli.js "$@"
