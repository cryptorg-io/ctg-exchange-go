# CTG.EXCHANGE Go SDK

Go client for the [CTG.EXCHANGE](https://ctg.exchange) exchange
API — a thin, typed wrapper over the public REST + WebSocket interface.

## Install

```sh
go get github.com/cryptorg-io/ctg-exchange-go
```

Requires Go 1.23+.

## Quick start

### Public market data — no key needed

```go
ctx := context.Background()
c := ctg-exchange.NewClient(ctg-exchange.Config{})

symbols, err := c.GetSymbols(ctx)
// ...
book, err := c.GetOrderBook(ctx, "CTGUSDT")
```

### Private account & trading

API keys are created in the CTG.EXCHANGE web app (Account → API keys). Read
them from the environment — never hard-code them.

```go
c := ctg-exchange.NewClient(ctg-exchange.Config{
    APIKey:    os.Getenv("CTG_EXCHANGE_API_KEY"),
    APISecret: os.Getenv("CTG_EXCHANGE_API_SECRET"),
})

balances, err := c.GetBalances(ctx)

res, err := c.PlaceOrder(ctx, "CTGUSDT", ctg-exchange.PlaceOrderParams{
    Side: "buy", Type: "limit", Price: "100.00", Qty: "1",
})
// res.Order, res.Trades
```

### WebSocket streams

```go
stream := ctg-exchange.NewMarketDataStream(ctg-exchange.StreamConfig{
    Channels: []string{"trades@CTGUSDT"},
})
defer stream.Close()

for {
    msg, err := stream.Next(ctx)
    if err != nil {
        break
    }
    fmt.Println(msg.Channel, msg.Type, string(msg.Data))
}
```

`NewUserStream` opens the private stream, authenticating in-band with
a signed first frame. Subscribe to any of `orders`, `trades`,
`balances`:

```go
stream, err := ctg-exchange.NewUserStream(
    os.Getenv("CTG_EXCHANGE_API_KEY"),
    os.Getenv("CTG_EXCHANGE_API_SECRET"),
    ctg-exchange.StreamConfig{Channels: []string{"orders", "balances"}},
)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    msg, err := stream.Next(ctx)
    if err != nil {
        break
    }
    fmt.Println(msg.Channel, msg.Type, string(msg.Data))
}
```

Both streams auto-reconnect and re-subscribe on a dropped socket.
`StreamMessage.Data` is a `json.RawMessage` — unmarshal it into the
shape you need.

## The decimal contract

Every monetary value — `Price`, `Qty`, `Volume`, fee amounts — is a
`string` (`"3500.55"`), never a float. That is lossless; parse it with
a big-decimal library if you need arithmetic.

## Errors

A non-2xx response is returned as an `*APIError`. Use `errors.As` and
the `Is*` helpers:

```go
var apiErr *ctg-exchange.APIError
if errors.As(err, &apiErr) && apiErr.IsRateLimited() {
    time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
}
```

`APIError` carries `StatusCode`, `Code`, `Message` and `RequestID`. Set
`Config.MaxRetries` to auto-retry `429`s.

## What this SDK does not do

Withdrawals are not part of the CTG.EXCHANGE API and not in this SDK — they
require a wallet signature and happen only in the web app.

## Development

```sh
go vet ./...
go test ./...   # offline tests run with no credentials
```

Integration tests run only when `CTG_EXCHANGE_API_KEY` / `CTG_EXCHANGE_API_SECRET`
are set, and are read-only — they never place orders.

## Links

- Docs: <https://docs.ctg.exchange>
- API reference: <https://docs.ctg.exchange/api/reference/>
- Security policy: [SECURITY.md](SECURITY.md)

## License

[Apache-2.0](LICENSE)
