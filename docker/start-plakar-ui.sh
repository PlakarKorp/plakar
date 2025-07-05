#!/bin/bash
set -e

echo "Starting Plakar agent..."
# Start the agent in foreground mode to avoid syslog issues in container
plakar agent -foreground &
AGENT_PID=$!

echo "Waiting for agent to be ready..."
# Wait for the agent to initialize
sleep 5

# Check if agent is still running
if ! kill -0 $AGENT_PID 2>/dev/null; then
    echo "Agent failed to start"
    exit 1
fi

echo "Starting Plakar UI..."
# Start the UI server with passed arguments
exec plakar ui "$@"
