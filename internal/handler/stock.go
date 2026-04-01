package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Most-Fullstack/stock-watcher/internal/model"
	"github.com/Most-Fullstack/stock-watcher/internal/store"
	"github.com/gin-gonic/gin"
)

// Handler serves stock watchlist HTTP API.
type Handler struct {
	store *store.Memory
}

// New returns a Handler backed by store.
func New(s *store.Memory) *Handler {
	return &Handler{store: s}
}

// Health reports service readiness.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ListStocks returns all watched symbols with mock prices.
func (h *Handler) ListStocks(c *gin.Context) {
	stocks := h.store.List()
	c.JSON(http.StatusOK, stocks)
}

// GetStock returns one watched stock.
func (h *Handler) GetStock(c *gin.Context) {
	symbol := c.Param("symbol")
	st, err := h.store.Get(symbol)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		slog.Error("get stock", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, st)
}

// AddStock adds a symbol to the watchlist.
func (h *Handler) AddStock(c *gin.Context) {
	var req model.AddStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: expect {\"symbol\":\"AAPL\"}"})
		return
	}
	sym := model.NormalizeSymbol(req.Symbol)
	if sym == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol required"})
		return
	}
	h.store.Add(sym)
	st, err := h.store.Get(sym)
	if err != nil {
		slog.Error("add stock get", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, st)
}

// RemoveStock removes a symbol from the watchlist.
func (h *Handler) RemoveStock(c *gin.Context) {
	symbol := c.Param("symbol")
	if err := h.store.Remove(symbol); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		slog.Error("remove stock", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateAlert sets a price alert on a watched symbol.
func (h *Handler) CreateAlert(c *gin.Context) {
	var req model.CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: expect {\"symbol\":\"AAPL\",\"target_price\":200,\"direction\":\"above\"}"})
		return
	}
	alert, err := h.store.AddAlert(req.Symbol, req.Target, req.Direction)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "symbol not in watchlist — add it first"})
			return
		}
		slog.Error("create alert", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	slog.Info("alert created", "alert_id", alert.ID, "symbol", alert.Symbol, "target", alert.Target, "direction", alert.Direction)
	c.JSON(http.StatusCreated, alert)
}

// ListAlerts returns all alerts with current triggered status.
func (h *Handler) ListAlerts(c *gin.Context) {
	alerts := h.store.ListAlerts()
	c.JSON(http.StatusOK, alerts)
}

// DeleteAlert removes an alert by ID.
func (h *Handler) DeleteAlert(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteAlert(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
			return
		}
		slog.Error("delete alert", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

// SetHolding sets the share quantity for a watched symbol.
func (h *Handler) SetHolding(c *gin.Context) {
	symbol := c.Param("symbol")
	var req model.SetHoldingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: expect {\"quantity\":10}"})
		return
	}
	if err := h.store.SetHolding(symbol, req.Quantity); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "symbol not in watchlist — add it first"})
			return
		}
		slog.Error("set holding", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	slog.Info("holding updated", "symbol", model.NormalizeSymbol(symbol), "quantity", req.Quantity)
	c.JSON(http.StatusOK, gin.H{"symbol": model.NormalizeSymbol(symbol), "quantity": req.Quantity})
}

// GetPortfolio returns the portfolio summary with all holdings and total value.
func (h *Handler) GetPortfolio(c *gin.Context) {
	portfolio := h.store.GetPortfolio()
	c.JSON(http.StatusOK, portfolio)
}
