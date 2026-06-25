package ctgexchange

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"
)

// Live, read-only integration tests against a real CTG.EXCHANGE API. They
// run only when CTG_EXCHANGE_API_KEY / CTG_EXCHANGE_API_SECRET are set (a local
// .env file in the package root is loaded as a convenience); otherwise
// every test here is skipped.
//
// Read-only by design: market-data and account read endpoints only —
// they never place, modify or cancel an order.

func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if os.Getenv(k) == "" {
			os.Setenv(k, strings.TrimSpace(v))
		}
	}
}

func integrationClient(t *testing.T) *Client {
	t.Helper()
	loadDotEnv()
	key, secret := os.Getenv("CTG_EXCHANGE_API_KEY"), os.Getenv("CTG_EXCHANGE_API_SECRET")
	if key == "" || secret == "" {
		t.Skip("integration test — set CTG_EXCHANGE_API_KEY / CTG_EXCHANGE_API_SECRET")
	}
	return NewClient(Config{
		APIKey:    key,
		APISecret: secret,
		BaseURL:   os.Getenv("CTG_EXCHANGE_BASE_URL"), // "" -> production
	})
}

func TestLiveSymbols(t *testing.T) {
	if _, err := integrationClient(t).GetSymbols(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLiveTickers(t *testing.T) {
	if _, err := integrationClient(t).GetTickers(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLiveBalances(t *testing.T) {
	if _, err := integrationClient(t).GetBalances(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLiveFees(t *testing.T) {
	if _, err := integrationClient(t).GetFees(context.Background()); err != nil {
		t.Fatal(err)
	}
}
