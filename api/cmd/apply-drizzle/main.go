// Command apply-drizzle applies apps/auth/drizzle SQL to DATABASE_URL (used by make migrate / CI).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Judeadeniji/zenv-sh/api/internal/dbschema"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "ping:", err)
		os.Exit(1)
	}
	if err := dbschema.Sync(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
