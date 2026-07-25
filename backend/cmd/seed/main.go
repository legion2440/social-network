package main

import (
	"context"
	"fmt"
	"log"

	"social-network/backend/internal/config"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

const demoPassword = "LoopDemo123!"

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := sqlite.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	passwordHash, err := service.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}
	applied, err := sqlite.ApplyDemoSeedMigrations(ctx, db, passwordHash)
	if err != nil {
		return fmt.Errorf("apply demo seed migrations: %w", err)
	}
	if len(applied) == 0 {
		log.Print("demo seed is already current")
		return nil
	}
	log.Printf("applied demo seed migrations: %v", applied)
	return nil
}
