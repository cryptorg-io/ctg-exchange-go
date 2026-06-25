// Public market data — no API key needed.
//
//	go run ./examples/marketdata
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cryptorg-io/ctg-exchange-go"
)

func main() {
	ctx := context.Background()
	c := ctgexchange.NewClient(ctgexchange.Config{})

	symbols, err := c.GetSymbols(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d trading symbols\n", len(symbols))
	for _, s := range symbols {
		fmt.Printf("  %s: tick=%s step=%s\n", s.Symbol, s.TickSize, s.StepSize)
	}
	if len(symbols) == 0 {
		return
	}
	sym := symbols[0].Symbol

	ticker, err := c.GetTicker(ctx, sym)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n%s last price: %s\n", sym, ticker.Price)

	book, err := c.GetOrderBook(ctx, sym)
	if err != nil {
		log.Fatal(err)
	}
	if len(book.Bids) > 0 && len(book.Asks) > 0 {
		fmt.Printf("best bid/ask: %s / %s\n", book.Bids[0].Price, book.Asks[0].Price)
	}

	trades, err := c.GetTrades(ctx, sym, 5)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range trades {
		fmt.Printf("  trade %s x %s\n", t.Price, t.Qty)
	}
}
