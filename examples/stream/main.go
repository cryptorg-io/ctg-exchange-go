// Streaming market data and private account updates over WebSocket.
//
//	go run ./examples/stream
//
// The public stream needs no credentials. The private stream reads
// CTG_EXCHANGE_API_KEY / CTG_EXCHANGE_API_SECRET from the environment.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cryptorg-io/ctg-exchange-go"
)

func main() {
	ctx := context.Background()
	watchMarketData(ctx)
	watchAccount(ctx)
}

func watchMarketData(ctx context.Context) {
	stream := ctgexchange.NewMarketDataStream(ctgexchange.StreamConfig{
		Channels: []string{"trades@CTGUSDT", "ticker@CTGUSDT"},
	})
	defer stream.Close()

	for i := 0; i < 10; i++ {
		msg, err := stream.Next(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[public] %s/%s: %s\n", msg.Channel, msg.Type, msg.Data)
	}
}

func watchAccount(ctx context.Context) {
	key := os.Getenv("CTG_EXCHANGE_API_KEY")
	secret := os.Getenv("CTG_EXCHANGE_API_SECRET")
	if key == "" || secret == "" {
		fmt.Println("[private] set CTG_EXCHANGE_API_KEY / CTG_EXCHANGE_API_SECRET to run")
		return
	}

	stream, err := ctgexchange.NewUserStream(key, secret, ctgexchange.StreamConfig{
		Channels: []string{"orders", "balances"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	for i := 0; i < 10; i++ {
		msg, err := stream.Next(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[private] %s/%s: %s\n", msg.Channel, msg.Type, msg.Data)
	}
}
