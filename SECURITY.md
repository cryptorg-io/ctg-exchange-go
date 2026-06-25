# Security policy

## Reporting a vulnerability

If you find a security issue in this SDK, **do not open a public
issue.** Email **security@ctg.exchange**.

Please include steps to reproduce and the SDK version. We aim to
acknowledge a report within a few business days. Please allow reasonable
time for a fix before any public disclosure.

## Handling API keys

This SDK signs requests with HMAC API keys. To keep them safe:

- **Never commit keys.** Keep them out of source — read them from
  environment variables or a secrets manager. This repo's `.gitignore`
  excludes `.env` files; do not override that.
- **Scope down.** Use a `read`-only key wherever you do not need to
  trade. Use an IP allowlist on the key when you can.
- **The secret is shown once.** If a secret is exposed — committed,
  logged, pasted into chat or a ticket — treat it as compromised and
  **revoke the key in the web app immediately**.
- API keys cannot withdraw funds; withdrawals require a wallet
  signature in the web app. A leaked key still allows trading and
  reading your account — rotate it promptly.

## Scope

This policy covers the SDK code in this repository. Vulnerabilities in
the CTG.EXCHANGE exchange or API itself should also go to
**security@ctg.exchange**.
