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
