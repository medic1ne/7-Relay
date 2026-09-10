package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/medic1ne/7-Relay/pkg/router"
)

const (
	Version = "0.1.0"
	Banner  = `
    ╔═══════════════════════════════════════╗
    ║          7-RELAY  v%s              ║
    ║   Smart AI Router • Single Binary     ║
    ║   40+ Providers • Auto-Fallback       ║
    ╚═══════════════════════════════════════╝
`
)

func main() {
	fmt.Printf(Banner, Version)

	// Load config
	cfg, err := router.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create router
	r, err := router.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("[7-Relay] Listening on http://localhost%s/v1\n", addr)
	fmt.Printf("[7-Relay] Dashboard: http://localhost:%d/dashboard\n", cfg.Port)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := r.ListenAndServe(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigCh
	fmt.Println("\n[7-Relay] Shutting down...")
	r.Shutdown()
	fmt.Println("[7-Relay] Goodbye!")
}
