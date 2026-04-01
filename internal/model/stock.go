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

// NormalizeSymbol uppercases and trims a ticker symbol.
func NormalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
