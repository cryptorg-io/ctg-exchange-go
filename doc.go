// Package ctgexchange is a Go client for the CTG.EXCHANGE exchange API.
//
// CTG.EXCHANGE is a hybrid crypto exchange (off-chain matcher, on-chain
// custody on BNB Smart Chain). This package is a thin, typed client over
// its public REST + WebSocket API.
//
//	c := ctgexchange.NewClient(ctgexchange.Config{APIKey: key, APISecret: secret})
//	balances, err := c.GetBalances(ctx)
//
// Every monetary value is a decimal string (the API's decimal contract),
// never a float — parse it with a big-decimal library if you need
// arithmetic. Withdrawals are intentionally not part of the API.
package ctgexchange

// Version is the SDK version.
const Version = "0.1.0"
