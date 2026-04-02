package store

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Most-Fullstack/stock-watcher/internal/model"
)

var ErrNotFound = errors.New("stock not in watchlist")

var basePrices = map[string]float64{
	"AAPL":  190,
	"GOOGL": 175,
	"TSLA":  250,
	"AMZN":  185,
	"MSFT":  420,
}

const maxHistoryPerSymbol = 100

// Memory holds the watchlist, alerts, holdings, price history, and generates mock prices.
type Memory struct {
	mu       sync.RWMutex
	rndMu    sync.Mutex
	symbols  map[string]struct{}
	holdings map[string]float64
	alerts   []model.Alert
	alertID  int
	rnd      *rand.Rand
	history  map[string][]model.PricePoint
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		symbols:  make(map[string]struct{}),
		holdings: make(map[string]float64),
		alerts:   make([]model.Alert, 0),
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
		history:  make(map[string][]model.PricePoint),
	}
}

// Add adds a symbol to the watchlist.
func (m *Memory) Add(symbol string) {
	sym := model.NormalizeSymbol(symbol)
	if sym == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.symbols[sym] = struct{}{}
}

// Remove deletes a symbol; returns ErrNotFound if it was not watched.
func (m *Memory) Remove(symbol string) error {
	sym := model.NormalizeSymbol(symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.symbols[sym]; !ok {
		return ErrNotFound
	}
	delete(m.symbols, sym)
	return nil
}

// List returns all watched symbols with current mock prices and records each price.
func (m *Memory) List() []model.Stock {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.Stock, 0, len(m.symbols))
	for sym := range m.symbols {
		price := m.mockPriceLocked(sym)
		m.appendHistory(sym, price)
		out = append(out, model.Stock{Symbol: sym, Price: price})
	}
	return out
}

// Get returns one watched stock or ErrNotFound and records the price.
func (m *Memory) Get(symbol string) (model.Stock, error) {
	sym := model.NormalizeSymbol(symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.symbols[sym]; !ok {
		return model.Stock{}, ErrNotFound
	}
	price := m.mockPriceLocked(sym)
	m.appendHistory(sym, price)
	return model.Stock{Symbol: sym, Price: price}, nil
}

// mockPriceLocked requires RLock or Lock held on mu.
func (m *Memory) mockPriceLocked(symbol string) float64 {
	base, ok := basePrices[symbol]
	if !ok {
		m.rndMu.Lock()
		n := m.rnd.Intn(400)
		m.rndMu.Unlock()
		base = 100 + float64(n)
	}
	m.rndMu.Lock()
	f := 0.98 + m.rnd.Float64()*0.04
	m.rndMu.Unlock()
	return round2(base * f)
}

func round2(x float64) float64 {
	return float64(int64(x*100+0.5)) / 100
}

// AddAlert creates a price alert for a watched symbol.
func (m *Memory) AddAlert(symbol string, target float64, direction string) (model.Alert, error) {
	sym := model.NormalizeSymbol(symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.symbols[sym]; !ok {
		return model.Alert{}, ErrNotFound
	}
	m.alertID++
	alert := model.Alert{
		ID:        fmt.Sprintf("alert-%d", m.alertID),
		Symbol:    sym,
		Target:    target,
		Direction: direction,
		CreatedAt: time.Now(),
	}
	m.alerts = append(m.alerts, alert)
	return alert, nil
}

// ListAlerts returns all alerts with triggered status evaluated against current prices.
func (m *Memory) ListAlerts() []model.Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Alert, len(m.alerts))
	copy(out, m.alerts)
	for i := range out {
		price := m.mockPriceLocked(out[i].Symbol)
		out[i].Triggered = isTriggered(price, out[i].Target, out[i].Direction)
	}
	return out
}

// DeleteAlert removes an alert by ID.
func (m *Memory) DeleteAlert(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, a := range m.alerts {
		if a.ID == id {
			m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func isTriggered(price, target float64, direction string) bool {
	if direction == "above" {
		return price >= target
	}
	return price <= target
}

// SetHolding sets the share quantity for a watched symbol. Quantity 0 removes the holding.
func (m *Memory) SetHolding(symbol string, quantity float64) error {
	sym := model.NormalizeSymbol(symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.symbols[sym]; !ok {
		return ErrNotFound
	}
	if quantity <= 0 {
		delete(m.holdings, sym)
	} else {
		m.holdings[sym] = quantity
	}
	return nil
}

// GetMarketSummary returns an aggregate snapshot of the watchlist.
func (m *Memory) GetMarketSummary() model.MarketSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := model.MarketSummary{
		TotalSymbols: len(m.symbols),
		AlertCount:   len(m.alerts),
	}

	if summary.TotalSymbols == 0 {
		return summary
	}

	var total float64
	var highest, lowest *model.Stock

	for sym := range m.symbols {
		price := m.mockPriceLocked(sym)
		total += price
		s := &model.Stock{Symbol: sym, Price: price}
		if highest == nil || price > highest.Price {
			highest = s
		}
		if lowest == nil || price < lowest.Price {
			lowest = s
		}
	}

	summary.AveragePrice = round2(total / float64(summary.TotalSymbols))
	summary.HighestStock = highest
	summary.LowestStock = lowest

	for _, a := range m.alerts {
		price := m.mockPriceLocked(a.Symbol)
		if isTriggered(price, a.Target, a.Direction) {
			summary.TriggeredCount++
		}
	}

	var portfolioTotal float64
	for sym, qty := range m.holdings {
		if _, watched := m.symbols[sym]; !watched {
			continue
		}
		portfolioTotal += m.mockPriceLocked(sym) * qty
	}
	summary.PortfolioValue = round2(portfolioTotal)

	return summary
}

// GetPortfolio returns all holdings with current prices and total value.
func (m *Memory) GetPortfolio() model.Portfolio {
	m.mu.RLock()
	defer m.mu.RUnlock()
	holdings := make([]model.Holding, 0, len(m.holdings))
	var total float64
	for sym, qty := range m.holdings {
		if _, watched := m.symbols[sym]; !watched {
			continue
		}
		price := m.mockPriceLocked(sym)
		value := round2(price * qty)
		total += value
		holdings = append(holdings, model.Holding{
			Symbol:   sym,
			Quantity: qty,
			Price:    price,
			Value:    value,
		})
	}
	return model.Portfolio{
		Holdings:   holdings,
		TotalValue: round2(total),
		StockCount: len(holdings),
	}
}

// appendHistory adds a price point capped at maxHistoryPerSymbol. Caller must hold mu.
func (m *Memory) appendHistory(symbol string, price float64) {
	pts := m.history[symbol]
	pts = append(pts, model.PricePoint{Price: price, Timestamp: time.Now()})
	if len(pts) > maxHistoryPerSymbol {
		pts = pts[len(pts)-maxHistoryPerSymbol:]
	}
	m.history[symbol] = pts
}

// RecordPrice explicitly records a price snapshot for a watched symbol.
func (m *Memory) RecordPrice(symbol string) (model.PricePoint, error) {
	sym := model.NormalizeSymbol(symbol)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.symbols[sym]; !ok {
		return model.PricePoint{}, ErrNotFound
	}
	price := m.mockPriceLocked(sym)
	pt := model.PricePoint{Price: price, Timestamp: time.Now()}
	m.appendHistory(sym, price)
	return pt, nil
}

// GetPriceHistory returns recorded price points and stats for a symbol.
// limit <= 0 returns all points.
func (m *Memory) GetPriceHistory(symbol string, limit int) (model.PriceHistory, error) {
	sym := model.NormalizeSymbol(symbol)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.symbols[sym]; !ok {
		return model.PriceHistory{}, ErrNotFound
	}

	pts := m.history[sym]
	if len(pts) == 0 {
		return model.PriceHistory{Symbol: sym, Points: []model.PricePoint{}}, nil
	}

	if limit > 0 && limit < len(pts) {
		pts = pts[len(pts)-limit:]
	}

	out := make([]model.PricePoint, len(pts))
	copy(out, pts)

	high := pts[0].Price
	low := pts[0].Price
	var sum float64
	for _, p := range pts {
		if p.Price > high {
			high = p.Price
		}
		if p.Price < low {
			low = p.Price
		}
		sum += p.Price
	}
	avg := round2(sum / float64(len(pts)))
	first := pts[0].Price
	last := pts[len(pts)-1].Price
	change := round2(last - first)
	var changePct float64
	if first != 0 {
		changePct = round2((change / first) * 100)
	}

	return model.PriceHistory{
		Symbol:     sym,
		Points:     out,
		High:       high,
		Low:        low,
		AvgPrice:   avg,
		Change:     change,
		ChangePct:  changePct,
		PointCount: len(out),
	}, nil
}
