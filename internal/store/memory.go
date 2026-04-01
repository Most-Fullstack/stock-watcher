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

// Memory holds the watchlist, alerts, and generates mock prices.
type Memory struct {
	mu      sync.RWMutex
	rndMu   sync.Mutex
	symbols map[string]struct{}
	alerts  []model.Alert
	alertID int
	rnd     *rand.Rand
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		symbols: make(map[string]struct{}),
		alerts:  make([]model.Alert, 0),
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
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

// List returns all watched symbols with current mock prices.
func (m *Memory) List() []model.Stock {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Stock, 0, len(m.symbols))
	for sym := range m.symbols {
		out = append(out, model.Stock{Symbol: sym, Price: m.mockPriceLocked(sym)})
	}
	return out
}

// Get returns one watched stock or ErrNotFound.
func (m *Memory) Get(symbol string) (model.Stock, error) {
	sym := model.NormalizeSymbol(symbol)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.symbols[sym]; !ok {
		return model.Stock{}, ErrNotFound
	}
	return model.Stock{Symbol: sym, Price: m.mockPriceLocked(sym)}, nil
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
