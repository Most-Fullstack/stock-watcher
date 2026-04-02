package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Most-Fullstack/stock-watcher/internal/handler"
	"github.com/Most-Fullstack/stock-watcher/internal/store"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mem := store.NewMemory()
	h := handler.New(mem)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	r.GET("/health", h.Health)
	r.GET("/api/stocks", h.ListStocks)
	r.GET("/api/stocks/:symbol", h.GetStock)
	r.POST("/api/stocks", h.AddStock)
	r.DELETE("/api/stocks/:symbol", h.RemoveStock)

	r.POST("/api/alerts", h.CreateAlert)
	r.GET("/api/alerts", h.ListAlerts)
	r.DELETE("/api/alerts/:id", h.DeleteAlert)

	r.PUT("/api/portfolio/:symbol", h.SetHolding)
	r.GET("/api/portfolio", h.GetPortfolio)

	r.GET("/api/market/summary", h.GetMarketSummary)

	r.POST("/api/stocks/:symbol/prices", h.RecordPrice)
	r.GET("/api/stocks/:symbol/prices", h.GetPriceHistory)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
		)
	}
}
