package ctgexchange

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the production REST base URL.
const DefaultBaseURL = "https://api.ctg.exchange"

// Config configures a Client. APIKey and APISecret are required only
// for the private (/api/v1/me/...) endpoints; public market data works
// without them.
type Config struct {
	APIKey     string
	APISecret  string
	BaseURL    string        // default DefaultBaseURL
	Timeout    time.Duration // default 10s; ignored when HTTPClient is set
	MaxRetries int           // retry 429s, honouring Retry-After; default 0
	HTTPClient *http.Client  // optional custom HTTP client
}

// Client is a REST client for the CTG.EXCHANGE API. It is safe for
// concurrent use.
type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	maxRetries int
	http       *http.Client
}

// NewClient builds a Client from cfg.
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	hc := cfg.HTTPClient
	if hc == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 10 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}
	return &Client{
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		baseURL:    strings.TrimRight(baseURL, "/"),
		maxRetries: cfg.MaxRetries,
		http:       hc,
	}
}

type reqOpts struct {
	query url.Values
	body  any
	auth  bool
}

func request[T any](
	ctx context.Context, c *Client, method, path string, opts reqOpts,
) (T, error) {
	var zero T

	// request-uri must be the path+query exactly as sent; the body hash
	// must cover the exact bytes on the wire.
	requestURI := path
	if enc := opts.query.Encode(); enc != "" {
		requestURI += "?" + enc
	}

	var bodyBytes []byte
	if opts.body != nil {
		var err error
		if bodyBytes, err = json.Marshal(opts.body); err != nil {
			return zero, err
		}
	}

	if opts.auth && (c.apiKey == "" || c.apiSecret == "") {
		return zero, ErrMissingCredentials
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(
			ctx, method, c.baseURL+requestURI, bytes.NewReader(bodyBytes))
		if err != nil {
			return zero, err
		}
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if opts.auth {
			h := RESTHeaders(c.apiKey, c.apiSecret, method,
				requestURI, string(bodyBytes), time.Now().Unix())
			for k, v := range h {
				req.Header.Set(k, v)
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return zero, err
		}
		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return zero, err
		}

		if resp.StatusCode < 300 {
			if len(respBody) == 0 {
				return zero, nil
			}
			var out T
			if err := json.Unmarshal(respBody, &out); err != nil {
				return zero, err
			}
			return out, nil
		}

		apiErr := parseAPIError(resp.StatusCode, respBody,
			resp.Header.Get("Retry-After"))
		if apiErr.StatusCode == 429 && attempt < c.maxRetries {
			delay := time.Duration(apiErr.RetryAfter) * time.Second
			if delay <= 0 {
				delay = time.Second
			}
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return zero, ctx.Err()
			}
		}
		return zero, apiErr
	}
}

func parseAPIError(status int, body []byte, retryAfter string) *APIError {
	e := &APIError{StatusCode: status}
	var parsed struct {
		Error     string `json:"error"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		e.Code, e.Message, e.RequestID = parsed.Error, parsed.Message, parsed.RequestID
	}
	if retryAfter != "" {
		if n, err := strconv.Atoi(retryAfter); err == nil {
			e.RetryAfter = n
		}
	}
	return e
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// -- public market data ------------------------------------------------

// GetSymbols returns all trading symbols with their order filters.
func (c *Client) GetSymbols(ctx context.Context) ([]Symbol, error) {
	return request[[]Symbol](ctx, c, http.MethodGet, "/api/v1/symbols", reqOpts{})
}

// GetTickers returns the 24h ticker for every symbol.
func (c *Client) GetTickers(ctx context.Context) ([]Ticker, error) {
	return request[[]Ticker](ctx, c, http.MethodGet, "/api/v1/tickers", reqOpts{})
}

// GetOrderBook returns the current order book for symbol.
func (c *Client) GetOrderBook(ctx context.Context, symbol string) (OrderBook, error) {
	return request[OrderBook](ctx, c, http.MethodGet,
		"/api/v1/"+symbol+"/orderbook", reqOpts{})
}

// GetTicker returns the 24h ticker for symbol.
func (c *Client) GetTicker(ctx context.Context, symbol string) (Ticker, error) {
	return request[Ticker](ctx, c, http.MethodGet,
		"/api/v1/"+symbol+"/ticker", reqOpts{})
}

// GetCandles returns candles for symbol at interval (1m/5m/15m/1h/4h/1d;
// defaults to 1m when empty). See CandleOptions for windowed queries.
func (c *Client) GetCandles(
	ctx context.Context, symbol, interval string, opts CandleOptions,
) ([]Candle, error) {
	if interval == "" {
		interval = "1m"
	}
	q := url.Values{"interval": {interval}}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.From > 0 {
		q.Set("from", strconv.FormatInt(opts.From, 10))
	}
	if opts.To > 0 {
		q.Set("to", strconv.FormatInt(opts.To, 10))
	}
	return request[[]Candle](ctx, c, http.MethodGet,
		"/api/v1/"+symbol+"/candles", reqOpts{query: q})
}

// GetTrades returns recent public trade prints for symbol. A limit of 0
// uses the server default.
func (c *Client) GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return request[[]Trade](ctx, c, http.MethodGet,
		"/api/v1/"+symbol+"/trades", reqOpts{query: q})
}

// -- private: account --------------------------------------------------

// GetBalances returns the API key owner's per-asset balances (read scope).
func (c *Client) GetBalances(ctx context.Context) ([]Balance, error) {
	return request[[]Balance](ctx, c, http.MethodGet,
		"/api/v1/me/balances", reqOpts{auth: true})
}

// GetFees returns the API key owner's fee/rebate snapshot (read scope).
func (c *Client) GetFees(ctx context.Context) (UserFees, error) {
	return request[UserFees](ctx, c, http.MethodGet,
		"/api/v1/me/fees", reqOpts{auth: true})
}

// -- private: orders ---------------------------------------------------

// PlaceOrder places an order (trade scope). A ClientOrderID is generated
// when p.ClientOrderID is empty, so retries de-dup safely.
func (c *Client) PlaceOrder(
	ctx context.Context, symbol string, p PlaceOrderParams,
) (OrderResult, error) {
	coid := p.ClientOrderID
	if coid == "" {
		coid = randomID()
	}
	body := map[string]string{
		"client_order_id": coid,
		"side":            p.Side,
		"type":            p.Type,
	}
	if p.Price != "" {
		body["price"] = p.Price
	}
	if p.Qty != "" {
		body["qty"] = p.Qty
	}
	return request[OrderResult](ctx, c, http.MethodPost,
		"/api/v1/me/orders/"+symbol, reqOpts{body: body, auth: true})
}

// GetOrders lists the owner's orders for symbol (read scope).
func (c *Client) GetOrders(
	ctx context.Context, symbol string, q OrderQuery,
) ([]Order, error) {
	v := url.Values{}
	if q.Status != "" {
		v.Set("status", q.Status)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	return request[[]Order](ctx, c, http.MethodGet,
		"/api/v1/me/orders/"+symbol, reqOpts{query: v, auth: true})
}

// GetOpenOrders returns every open order across all symbols (read scope).
// Each Order carries its own Symbol; decimal fields are converted per
// that order's scales. For closed-order history use GetOrders per symbol.
func (c *Client) GetOpenOrders(ctx context.Context) ([]Order, error) {
	return request[[]Order](ctx, c, http.MethodGet,
		"/api/v1/me/orders/open", reqOpts{auth: true})
}

// GetOrder returns one order by its canonical server id (read scope).
func (c *Client) GetOrder(ctx context.Context, symbol, orderID string) (Order, error) {
	return request[Order](ctx, c, http.MethodGet,
		"/api/v1/me/orders/"+symbol+"/"+orderID, reqOpts{auth: true})
}

// CancelOrder cancels one order (trade scope).
func (c *Client) CancelOrder(ctx context.Context, symbol, orderID string) (Order, error) {
	return request[Order](ctx, c, http.MethodDelete,
		"/api/v1/me/orders/"+symbol+"/"+orderID, reqOpts{auth: true})
}

// CancelAllOrders cancels every open order for symbol (trade scope).
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) ([]Order, error) {
	res, err := request[struct {
		Cancelled int     `json:"cancelled"`
		Orders    []Order `json:"orders"`
	}](ctx, c, http.MethodDelete, "/api/v1/me/orders/"+symbol, reqOpts{auth: true})
	return res.Orders, err
}

// ModifyOrder modifies a resting order's price and quantity (trade
// scope). The API expects the full new state — set both fields.
func (c *Client) ModifyOrder(
	ctx context.Context, symbol, orderID string, p ModifyOrderParams,
) (Order, error) {
	body := map[string]string{}
	if p.NewPrice != "" {
		body["new_price"] = p.NewPrice
	}
	if p.NewQty != "" {
		body["new_qty"] = p.NewQty
	}
	raw, err := request[json.RawMessage](ctx, c, http.MethodPatch,
		"/api/v1/me/orders/"+symbol+"/"+orderID, reqOpts{body: body, auth: true})
	if err != nil {
		return Order{}, err
	}
	// The API wraps the modified order: {"order": {...}, "trades": [...]}.
	var wrapped struct {
		Order *Order `json:"order"`
	}
	if json.Unmarshal(raw, &wrapped) == nil && wrapped.Order != nil {
		return *wrapped.Order, nil
	}
	var bare Order
	if err := json.Unmarshal(raw, &bare); err != nil {
		return Order{}, err
	}
	return bare, nil
}

// -- private: trades ---------------------------------------------------

// GetMyTrades returns the owner's trade history for symbol (read scope).
func (c *Client) GetMyTrades(
	ctx context.Context, symbol string, q TradeQuery,
) ([]Trade, error) {
	v := url.Values{}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	return request[[]Trade](ctx, c, http.MethodGet,
		"/api/v1/me/trades/"+symbol, reqOpts{query: v, auth: true})
}
