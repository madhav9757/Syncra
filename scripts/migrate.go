package main

import (
	"context"
	"fmt"
	"log"
	"syncra/internal/server/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// Migration: Add public_key column
	query := `ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT;`

	fmt.Println("Running migration: ADD COLUMN public_key...")
	_, err = db.Pool.Exec(ctx, query)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration successful!")
}
