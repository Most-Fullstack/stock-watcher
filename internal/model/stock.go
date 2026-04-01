package model

import "strings"

// Stock is a watched symbol with a generated market price.
type Stock struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

// AddStockRequest is the JSON body for POST /api/stocks.
type AddStockRequest struct {
	Symbol string `json:"symbol" binding:"required"`
}

// NormalizeSymbol uppercases and trims a ticker symbol.
func NormalizeSymbol(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
