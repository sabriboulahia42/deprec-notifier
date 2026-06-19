# Deprecation Notifier

Notifies by Email (MailerSend or SMTP), Discord, and Facebook Messenger when Node.js or Python interpreter versions approach End-of-Life.

Deployment: Render (recommended) or GitHub Actions / Fly. The service includes an AES-256-GCM encrypt/decrypt helper and supports running once (for scheduled CI runs) or continuously with a daily ticker.

Quick Start (local)

1. Clone the repo and install Go 1.20+
2. Create `.env` with:
   ```
   SMTP_HOST=smtp.mailersend.net
   SMTP_PORT=587
   SMTP_USERNAME=...
   SMTP_PASSWORD=...
   SMTP_FROM_EMAIL=...
   NOTIFY_TO_EMAIL=...
   PRIVATE_KEY_B64=...
   ```
3. Generate encryption key (one-time):
   ```bash
   go build -o envcrypt encrypt_decrypt.go
   ./envcrypt gen-key   # copy output -> PRIVATE_KEY_B64 env var
   ```
4. Encrypt .env (optional, for storing encrypted blob in repo or as a secret):
   ```bash
   export PRIVATE_KEY_B64="<the-key>"
   ./envcrypt encrypt .env .env.enc
   ```
5. Build and run:
   ```bash
   go build -o notifier main.go
   ./notifier              # runs continuously with daily ticker
   ./notifier --run-once   # runs once and exits
   ```

Render deployment (UI - quick)
1. Create a new Web Service, choose the repo and set Environment to Docker.
2. Set the Dockerfile path to `.` and branch to `main`.
3. Add Environment Variables / Secrets (SMTP_*, NOTIFY_TO_EMAIL, PRIVATE_KEY_B64 if used).
4. Deploy and verify `https://<service>.onrender.com/health`.

Security notes
- Do NOT commit plaintext `.env` or `PRIVATE_KEY_B64`.
- Store runtime secrets in Render/Fly secrets and only the deploy token in GitHub Secrets.
