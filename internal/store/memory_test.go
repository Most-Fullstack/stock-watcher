package store

import (
	"testing"

	"github.com/Most-Fullstack/stock-watcher/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddListGetRemove(t *testing.T) {
	m := NewMemory()
	m.Add("aapl")
	list := m.List()
	if len(list) != 1 {
		t.Fatalf("list len: got %d want 1", len(list))
	}
	if list[0].Symbol != "AAPL" {
		t.Fatalf("symbol: got %q", list[0].Symbol)
	}
	if list[0].Price < 180 || list[0].Price > 200 {
		t.Fatalf("AAPL mock price out of ±2%% band: %v", list[0].Price)
	}
	st, err := m.Get("AAPL")
	if err != nil {
		t.Fatal(err)
	}
	if st.Symbol != "AAPL" {
		t.Fatalf("get symbol: %q", st.Symbol)
	}
	if err := m.Remove("AAPL"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get("AAPL"); err != ErrNotFound {
		t.Fatalf("get after remove: %v", err)
	}
}

func TestNormalizeViaModel(t *testing.T) {
	if model.NormalizeSymbol("  msft  ") != "MSFT" {
		t.Fatal("normalize")
	}
}

func TestAlertLifecycle(t *testing.T) {
	m := NewMemory()
	m.Add("AAPL")

	tests := []struct {
		name      string
		symbol    string
		target    float64
		direction string
		wantErr   error
	}{
		{name: "alert on watched stock", symbol: "AAPL", target: 200, direction: "above", wantErr: nil},
		{name: "alert below threshold", symbol: "AAPL", target: 100, direction: "below", wantErr: nil},
		{name: "alert on unwatched stock", symbol: "NVDA", target: 500, direction: "above", wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.AddAlert(tc.symbol, tc.target, tc.direction)
			if err != tc.wantErr {
				t.Fatalf("AddAlert() error = %v, want %v", err, tc.wantErr)
			}
		})
	}

	alerts := m.ListAlerts()
	if len(alerts) != 2 {
		t.Fatalf("ListAlerts() len = %d, want 2", len(alerts))
	}
	if alerts[0].Symbol != "AAPL" {
		t.Fatalf("alert symbol = %q, want AAPL", alerts[0].Symbol)
	}

	if err := m.DeleteAlert(alerts[0].ID); err != nil {
		t.Fatalf("DeleteAlert() error = %v", err)
	}
	if len(m.ListAlerts()) != 1 {
		t.Fatal("expected 1 alert after delete")
	}

	if err := m.DeleteAlert("nonexistent"); err != ErrNotFound {
		t.Fatalf("DeleteAlert(nonexistent) error = %v, want ErrNotFound", err)
	}
}

func TestPortfolioLifecycle(t *testing.T) {
	m := NewMemory()
	m.Add("AAPL")
	m.Add("MSFT")

	tests := []struct {
		name    string
		symbol  string
		qty     float64
		wantErr error
	}{
		{name: "set holding for watched stock", symbol: "AAPL", qty: 10, wantErr: nil},
		{name: "set holding for another stock", symbol: "MSFT", qty: 5, wantErr: nil},
		{name: "set holding for unwatched stock", symbol: "NVDA", qty: 3, wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := m.SetHolding(tc.symbol, tc.qty)
			if err != tc.wantErr {
				t.Fatalf("SetHolding() error = %v, want %v", err, tc.wantErr)
			}
		})
	}

	p := m.GetPortfolio()
	if p.StockCount != 2 {
		t.Fatalf("portfolio stock count = %d, want 2", p.StockCount)
	}
	if p.TotalValue <= 0 {
		t.Fatalf("portfolio total value = %f, want > 0", p.TotalValue)
	}

	if err := m.SetHolding("AAPL", 0); err != nil {
		t.Fatalf("SetHolding(0) error = %v", err)
	}
	p2 := m.GetPortfolio()
	if p2.StockCount != 1 {
		t.Fatalf("portfolio after remove: stock count = %d, want 1", p2.StockCount)
	}
}

func TestRecordPrice(t *testing.T) {
	m := NewMemory()
	m.Add("AAPL")

	tests := []struct {
		name    string
		symbol  string
		wantErr error
	}{
		{name: "record price for watched stock", symbol: "AAPL", wantErr: nil},
		{name: "record price for unwatched stock", symbol: "NVDA", wantErr: ErrNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := m.RecordPrice(tc.symbol)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, pt.Price, 0.0)
			assert.False(t, pt.Timestamp.IsZero())
		})
	}
}

func TestGetPriceHistory(t *testing.T) {
	m := NewMemory()
	m.Add("AAPL")

	for range 5 {
		_, err := m.RecordPrice("AAPL")
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		symbol     string
		limit      int
		wantErr    error
		wantPoints int
	}{
		{name: "all points", symbol: "AAPL", limit: 0, wantErr: nil, wantPoints: 5},
		{name: "limited to 3", symbol: "AAPL", limit: 3, wantErr: nil, wantPoints: 3},
		{name: "limit exceeds count", symbol: "AAPL", limit: 100, wantErr: nil, wantPoints: 5},
		{name: "unwatched stock", symbol: "NVDA", limit: 0, wantErr: ErrNotFound, wantPoints: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, err := m.GetPriceHistory(tc.symbol, tc.limit)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "AAPL", h.Symbol)
			assert.Equal(t, tc.wantPoints, h.PointCount)
			assert.Len(t, h.Points, tc.wantPoints)
			assert.Greater(t, h.High, 0.0)
			assert.Greater(t, h.Low, 0.0)
			assert.GreaterOrEqual(t, h.High, h.Low)
			assert.Greater(t, h.AvgPrice, 0.0)
		})
	}
}

func TestPriceHistoryRecordedOnGet(t *testing.T) {
	m := NewMemory()
	m.Add("TSLA")

	_, err := m.Get("TSLA")
	require.NoError(t, err)

	h, err := m.GetPriceHistory("TSLA", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, h.PointCount, "Get should record a price point")
}

func TestPriceHistoryRecordedOnList(t *testing.T) {
	m := NewMemory()
	m.Add("MSFT")

	_ = m.List()

	h, err := m.GetPriceHistory("MSFT", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, h.PointCount, "List should record a price point per symbol")
}

func TestPriceHistoryCapAtMax(t *testing.T) {
	m := NewMemory()
	m.Add("AAPL")

	for range maxHistoryPerSymbol + 20 {
		_, err := m.RecordPrice("AAPL")
		require.NoError(t, err)
	}

	h, err := m.GetPriceHistory("AAPL", 0)
	require.NoError(t, err)
	assert.Equal(t, maxHistoryPerSymbol, h.PointCount, "history should be capped at max")
}

func TestPriceHistoryEmptyForNewSymbol(t *testing.T) {
	m := NewMemory()
	m.Add("GOOGL")

	h, err := m.GetPriceHistory("GOOGL", 0)
	require.NoError(t, err)
	assert.Equal(t, 0, h.PointCount)
	assert.Empty(t, h.Points)
}
