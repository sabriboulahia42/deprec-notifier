#!/bin/sh
set -e

# If ENCRYPTED_ENV is set, decrypt it
if [ -n "$ENCRYPTED_ENV" ]; then
  if [ -z "$PRIVATE_KEY_B64" ]; then
    echo "ERROR: ENCRYPTED_ENV set but PRIVATE_KEY_B64 missing" >&2
    exit 1
  fi
  echo "$ENCRYPTED_ENV" > /tmp/.env.enc
  export PRIVATE_KEY_B64="$PRIVATE_KEY_B64"
  /app/envcrypt decrypt /tmp/.env.enc /app/.env 2>&1
  echo "[entrypoint] .env decrypted successfully"
  rm -f /tmp/.env.enc
fi

# Run notifier (or pass args)
exec /app/notifier "$@"
