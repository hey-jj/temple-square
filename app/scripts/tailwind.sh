#!/bin/bash
# Tailwind CSS CLI wrapper

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/tailwindcss" "$@"
