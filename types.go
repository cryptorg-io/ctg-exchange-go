package ctgexchange

import "encoding/json"

// The API's decimal contract: every monetary value — price, qty,
// volume, fee amounts — is a JSON string ("3500.55"), never a number.
// This SDK keeps those fields as string: that is lossless and leaves
// the choice of big-decimal library to you. Basis-point fields,
// counters, ids and epoch timestamps are real integers.

// Symbol is a trading pair plus its order filters.
type Symbol struct {
	Symbol          string `json:"symbol"`
	BaseAsset       string `json:"base_asset"`
	QuoteAsset      string `json:"quote_asset"`
	PriceScale      int    `json:"price_scale"`
	QtyScale        int    `json:"qty_scale"`
	QuoteAssetScale int    `json:"quote_asset_scale"`
	TickSize        string `json:"tick_size"`
	StepSize        string `json:"step_size"`
	MinPrice        string `json:"min_price"`
	MaxPrice        string `json:"max_price"`
	MinQty          string `json:"min_qty"`
	MaxQty          string `json:"max_qty"`
	MinNotional     string `json:"min_notional"`
}

// Ticker is a 24h rolling ticker.
type Ticker struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"` // base-asset volume
	Trades    int    `json:"trades"`
	ChangeBps int    `json:"change_bps"`
	Ts        int64  `json:"ts"` // epoch
}

// Candle is a single OHLC candle.
type Candle struct {
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	OpenTime  int64  `json:"open_time"`
	CloseTime int64  `json:"close_time"`
	Trades    int    `json:"trades"`
}

// BookLevel is one price level of an order book.
type BookLevel struct {
	Price string `json:"price"`
	Qty   string `json:"qty"`
}

// OrderBook is a full order book snapshot.
type OrderBook struct {
	Symbol       string      `json:"symbol"`
	LastUpdateID int64       `json:"last_update_id"`
	Bids         []BookLevel `json:"bids"`
	Asks         []BookLevel `json:"asks"`
}

// Balance is a per-asset balance.
type Balance struct {
	Asset     string `json:"asset"`
	Available string `json:"available"`
	Reserved  string `json:"reserved"`
}

// FeeRebate is the fraction of a charged fee credited to a referrer.
// An empty object (zero fields) when no rebate applies.
type FeeRebate struct {
	AccountID   string `json:"account_id"`   // referrer wallet address
	FractionBps int    `json:"fraction_bps"` // share of the fee in bps
}

// Order is an order as returned by the API.
type Order struct {
	ID                string     `json:"id"`
	ClientOrderID     string     `json:"client_order_id"`
	UID               string     `json:"uid"` // owner wallet address
	Symbol            string     `json:"symbol"`
	Side              string     `json:"side"` // buy / sell
	Type              string     `json:"type"` // limit / market
	TimeInForce       string     `json:"time_in_force"`
	Status            string     `json:"status"`
	Price             string     `json:"price"`
	AvgExecutionPrice string     `json:"avg_execution_price"`
	Qty               string     `json:"qty"`
	FilledQty         string     `json:"filled_qty"`
	RemainingQty      string     `json:"remaining_qty"`
	FeeAmount         string     `json:"fee_amount"` // decimal string in fee_asset scale
	FeeAsset          string     `json:"fee_asset"`
	TakerFeeBps       int        `json:"taker_fee_bps"` // per-order override; 0 = default
	MakerFeeBps       int        `json:"maker_fee_bps"`
	Rebate            *FeeRebate `json:"rebate"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

// Trade is a matched trade. Both sides' addresses, order ids and fees
// are exposed — see "matcher transparency".
type Trade struct {
	ID            string     `json:"id"`
	Symbol        string     `json:"symbol"`
	Price         string     `json:"price"`
	Qty           string     `json:"qty"`
	BuyOrderID    string     `json:"buy_order_id"`
	SellOrderID   string     `json:"sell_order_id"`
	BuyUID        string     `json:"buy_uid"`  // buyer wallet address
	SellUID       string     `json:"sell_uid"` // seller wallet address
	AggressorSide string     `json:"aggressor_side"`
	MakerOrderID  string     `json:"maker_order_id"`
	TakerOrderID  string     `json:"taker_order_id"`
	BuyFeeBps     int        `json:"buy_fee_bps"`  // omitted when 0
	SellFeeBps    int        `json:"sell_fee_bps"` // omitted when 0
	BuyRebate     *FeeRebate `json:"buy_rebate"`
	SellRebate    *FeeRebate `json:"sell_rebate"`
	BuyFeeAmount  string     `json:"buy_fee_amount"`
	BuyFeeAsset   string     `json:"buy_fee_asset"`
	SellFeeAmount string     `json:"sell_fee_amount"`
	SellFeeAsset  string     `json:"sell_fee_asset"`
	CreatedAt     string     `json:"created_at"`
}

// Rebate is the referral-rebate entry inside a UserFees snapshot.
type Rebate struct {
	ReferrerAccount string `json:"referrer_account"`
	FractionBps     int    `json:"fraction_bps"`
}

// UserFees is a fee/rebate snapshot. A nil pointer integer means no
// override applies — the platform default is then in effect.
type UserFees struct {
	TakerFeeBps *int    `json:"taker_fee_bps"`
	MakerFeeBps *int    `json:"maker_fee_bps"`
	Rebate      *Rebate `json:"rebate"`
}

// OrderResult is the response to placing or modifying an order: the
// order plus any fills that happened immediately.
type OrderResult struct {
	Order  Order   `json:"order"`
	Trades []Trade `json:"trades"`
}

// PlaceOrderParams are the parameters for placing an order. Price and
// Qty are decimal strings. ClientOrderID is generated when empty.
type PlaceOrderParams struct {
	Side          string // buy / sell
	Type          string // limit / market
	Price         string
	Qty           string
	ClientOrderID string
}

// ModifyOrderParams are the parameters for modifying an order. Send
// both fields — the API expects the full new state.
type ModifyOrderParams struct {
	NewPrice string
	NewQty   string
}

// CandleOptions are the optional query parameters for GetCandles.
// Leave a field zero to omit it. Omit From/To for the latest Limit
// candles; set them (Unix ms) for a historical window [From, To).
type CandleOptions struct {
	Limit int
	From  int64
	To    int64
}

// OrderQuery are the optional filters for GetOrders.
type OrderQuery struct {
	Status string // open / partially_filled / filled / canceled
	Limit  int
	Offset int
}

// TradeQuery are the optional filters for GetMyTrades.
type TradeQuery struct {
	Limit  int
	Offset int
}

// StreamMessage is one server WebSocket message: a snapshot or update.
type StreamMessage struct {
	Type    string          // "snapshot" or "update"
	Channel string          // e.g. "orderbook"
	Symbol  string          // e.g. "CTGUSDT"; empty for private channels
	Data    json.RawMessage // channel payload — unmarshal as needed
}
