// Private account access and order placement.
//
//	export CTG_EXCHANGE_API_KEY=ak_...
//	export CTG_EXCHANGE_API_SECRET=sk_...
//	go run ./examples/account
//
// This example reads balances and fees. Order placement is shown but
// commented out — uncomment it only when you intend to send a real
// order to a real exchange.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cryptorg-io/ctg-exchange-go"
)

func main() {
	key, secret := os.Getenv("CTG_EXCHANGE_API_KEY"), os.Getenv("CTG_EXCHANGE_API_SECRET")
	if key == "" || secret == "" {
		log.Fatal("set CTG_EXCHANGE_API_KEY and CTG_EXCHANGE_API_SECRET")
	}

	ctx := context.Background()
	c := ctgexchange.NewClient(ctgexchange.Config{
		APIKey:     key,
		APISecret:  secret,
		MaxRetries: 3,
	})

	balances, err := c.GetBalances(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Balances:")
	for _, b := range balances {
		fmt.Printf("  %s: %s available, %s reserved\n",
			b.Asset, b.Available, b.Reserved)
	}

	fees, err := c.GetFees(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nFees: taker=%v maker=%v\n", fees.TakerFeeBps, fees.MakerFeeBps)

	// --- Placing an order (sends a REAL order) ---------------------------
	// res, err := c.PlaceOrder(ctx, "CTGUSDT", ctgexchange.PlaceOrderParams{
	//     Side:  "buy",
	//     Type:  "limit",
	//     Price: "100.00",
	//     Qty:   "1",
	//     // ClientOrderID is auto-generated for safe retries.
	// })
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Println("Placed:", res.Order.ID, res.Order.Status)
	//
	// _, _ = c.CancelOrder(ctx, "CTGUSDT", res.Order.ID)
}
