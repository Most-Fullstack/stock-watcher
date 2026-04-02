package model

import (
	"strings"
	"time"
)

// Stock is a watched symbol with a generated market price.
type Stock struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

// AddStockRequest is the JSON body for POST /api/stocks.
type AddStockRequest struct {
	Symbol string `json:"symbol" binding:"required"`
}

// Alert is a user-defined price threshold on a watched symbol.
type Alert struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Target    float64   `json:"target_price"`
	Direction string    `json:"direction"` // "above" or "below"
	Triggered bool      `json:"triggered"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAlertRequest is the JSON body for POST /api/alerts.
type CreateAlertRequest struct {
	Symbol    string  `json:"symbol" binding:"required"`
	Target    float64 `json:"target_price" binding:"required,gt=0"`
	Direction string  `json:"direction" binding:"required,oneof=above below"`
}

// Holding represents a position: shares owned for a symbol.
type Holding struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Value    float64 `json:"value"`
}

// Portfolio is the aggregate view of all holdings.
type Portfolio struct {
	Holdings   []Holding `json:"holdings"`
	TotalValue float64   `json:"total_value"`
	StockCount int       `json:"stock_count"`
}

// SetHoldingRequest is the JSON body for PUT /api/portfolio/:symbol.
type SetHoldingRequest struct {
	Quantity float64 `json:"quantity" binding:"required,gte=0"`
}

// MarketSummary is an aggregate snapshot of the entire watchlist.
type MarketSummary struct {
	TotalSymbols   int     `json:"total_symbols"`
	AveragePrice   float64 `json:"average_price"`
	HighestStock   *Stock  `json:"highest_stock,omitempty"`
	LowestStock    *Stock  `json:"lowest_stock,omitempty"`
	AlertCount     int     `json:"alert_count"`
	TriggeredCount int     `json:"triggered_count"`
	PortfolioValue float64 `json:"portfolio_value"`
}

// NormalizeSymbol uppercases and trims a ticker symbol.
func NormalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
