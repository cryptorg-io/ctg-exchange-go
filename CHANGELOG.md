# Changelog

All notable changes to the CTG.EXCHANGE Go SDK are documented here.
This project follows [Semantic Versioning](https://semver.org/).

## [0.1.2] - 2026-06-01

### Documentation

- README: added a private `NewUserStream` code snippet alongside the
  existing public `NewMarketDataStream` one.
- `examples/stream/main.go`: split into `watchMarketData` /
  `watchAccount`, with the private stream gated on
  `CTG_EXCHANGE_API_KEY` / `CTG_EXCHANGE_API_SECRET`. No SDK code changes.

## [0.1.1] - 2026-06-01

### Added

- `Client.GetOpenOrders(ctx)` — wraps `GET /api/v1/me/orders/open`, the
  cross-symbol open-orders endpoint.

## [0.1.0] - 2026-05-22

### Added

- Initial release: `Client` covering the full `/api/v1` REST surface —
  public market data and private account, order and trade endpoints.
- HMAC request signing for REST and the in-band WebSocket auth.
- `MarketDataStream` and `UserStream` WebSocket streams with
  auto-reconnect and subscription replay.
- Typed payload structs following the API's decimal-string contract.
- `APIError` with `StatusCode`, `RequestID`, `RetryAfter` and `Is*`
  helpers; `errors.As`-friendly.
- Context-aware: every call takes a `context.Context`.
