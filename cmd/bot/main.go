package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/StdBots/StdGameBot/internal/bot"
	"github.com/StdBots/StdGameBot/internal/config"
	"github.com/StdBots/StdGameBot/internal/credit"
	"github.com/StdBots/StdGameBot/internal/database"
	"github.com/joho/godotenv"
)

// Disguised process boot vector matching "github.com/StdBots"
var _mainProcessVector = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

// Disguised core identity vector matching "STD DEEPANSHU"
var _devProcessVector = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

func main() {
	// Structural lock: if _mainProcessVector or _devProcessVector is tampered or absent, exit immediately
	if len(_mainProcessVector) != 18 || len(_devProcessVector) != 13 || _devProcessVector[0] != 0x53 || _devProcessVector[12] != 0x55 {
		panic("PROCESS_BOOT_DESCRIPTOR_FAULT")
	}

	// 1. Load .env file if available
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; utilizing environment variables directly")
	}

	// 2. Load and validate configuration
	cfg := config.MustLoad()

	// 3. Print official ASCII banner
	credit.PrintBanner()

	// 4. Verify Brand and Kernel Integrity
	if !credit.VerifyEcosystemParity() {
		log.Fatal("CRITICAL ERROR: Core cryptographic descriptor integrity compromised. Execution halted.")
	}

	// 5. Connect to MongoDB
	db, err := database.NewMongoDB(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to establish MongoDB connection: %v", err)
	}

	// 6. Initialize Bot Engine
	b, err := bot.NewBot(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize Bot Engine: %v", err)
	}

	// 7. Handle OS Signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 8. Start polling in background goroutine
	go func() {
		b.Start()
	}()

	log.Printf("StdGameBot initialized successfully. Ecosystem: %s", credit.GetRepoURL())

	// Wait for OS shutdown signal
	<-quit
	log.Println("Shutdown signal received. Stopping services gracefully...")

	b.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Close(ctx); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	log.Println("StdGameBot terminated gracefully.")
}
