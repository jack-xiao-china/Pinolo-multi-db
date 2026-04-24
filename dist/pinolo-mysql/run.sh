#!/bin/bash
# PINOLO MySQL Logic Bug Detector - Linux/macOS Launcher
# Usage: ./run.sh [task|taskpool] <config_file>

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Detect OS and set executable
OS=$(uname -s)
ARCH=$(uname -m)

case "$OS" in
    Linux)
        EXECUTABLE="$SCRIPT_DIR/impomysql_linux"
        ;;
    Darwin)
        if [ "$ARCH" = "arm64" ]; then
            EXECUTABLE="$SCRIPT_DIR/impomysql_darwin_arm64"
        else
            EXECUTABLE="$SCRIPT_DIR/impomysql_darwin_amd64"
        fi
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
esac

# Make executable if needed
chmod +x "$EXECUTABLE" 2>/dev/null || true

# Check command
CMD="${1:-task}"
if [ "$CMD" != "task" ] && [ "$CMD" != "taskpool" ]; then
    echo "Usage: $0 [task|taskpool] <config_file>"
    echo "  task      - Run single testing task"
    echo "  taskpool  - Run tasks in parallel"
    echo ""
    echo "Example:"
    echo "  $0 task task_template.json"
    echo "  $0 taskpool taskpool_template.json"
    exit 1
fi

# Check config file
CONFIG="${2:-task_template.json}"
if [ ! -f "$CONFIG" ]; then
    echo "Config file not found: $CONFIG"
    exit 1
fi

echo "=== PINOLO MySQL Logic Bug Detector ==="
echo "OS: $OS ($ARCH)"
echo "Command: $CMD"
echo "Config: $CONFIG"
echo ""

# Run
"$EXECUTABLE" "$CMD" "$CONFIG"

echo ""
echo "=== Results ==="
echo "Output directory: ./output/mysql/"
echo "Check result.json for statistics"
echo "Check bugs/ directory for detected logical bugs"