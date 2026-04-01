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
