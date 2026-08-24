package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/NicolasKonishi/FinancialControl/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	steps := flag.Int("steps", 0, "optional number of migrations to apply (0 = all)")
	flag.Parse()

	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil && filepath.Dir(cfg.SQLitePath) != "." {
		log.Fatalf("create sqlite directory: %v", err)
	}

	sourceURL := fmt.Sprintf("file://%s", cfg.MigrationsPath)
	databaseURL := fmt.Sprintf("sqlite://%s", cfg.SQLitePath)

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}
	defer m.Close()

	switch *direction {
	case "up":
		err = runUp(m, *steps)
	case "down":
		err = runDown(m, *steps)
	default:
		log.Fatalf("unknown direction %q (use up or down)", *direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate %s: %v", *direction, err)
	}

	version, dirty, verr := m.Version()
	if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
		log.Fatalf("migrate version: %v", verr)
	}
	if errors.Is(verr, migrate.ErrNilVersion) {
		log.Printf("migrations complete: no version applied")
		return
	}
	log.Printf("migrations complete: version=%d dirty=%v", version, dirty)
}

func runUp(m *migrate.Migrate, steps int) error {
	if steps > 0 {
		return m.Steps(steps)
	}
	return m.Up()
}

func runDown(m *migrate.Migrate, steps int) error {
	if steps > 0 {
		return m.Steps(-steps)
	}
	return m.Down()
}
