package store

import (
	"errors"
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

// Memory holds the watchlist and generates mock prices.
type Memory struct {
	mu      sync.RWMutex
	rndMu   sync.Mutex
	symbols map[string]struct{}
	rnd     *rand.Rand
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		symbols: make(map[string]struct{}),
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
