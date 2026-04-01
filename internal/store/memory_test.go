package store

import (
	"testing"

	"github.com/Most-Fullstack/stock-watcher/internal/model"
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
