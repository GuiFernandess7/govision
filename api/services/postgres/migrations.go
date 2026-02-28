package postgres

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// RunMigrations reads and executes all .sql files from the given directory
// in alphabetical order. Each file is executed as a single raw SQL statement.
// Uses IF NOT EXISTS clauses in the SQL files to ensure idempotency.
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	log.Printf("[MIGRATIONS] - Running migrations from: %s", migrationsDir)

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	if len(files) == 0 {
		log.Println("[MIGRATIONS] - No migration files found, skipping.")
		return nil
	}

	sort.Strings(files)

	for _, file := range files {
		log.Printf("[MIGRATIONS] - Executing: %s", filepath.Base(file))

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filepath.Base(file), err)
		}

		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filepath.Base(file), err)
		}

		log.Printf("[MIGRATIONS] - Completed: %s", filepath.Base(file))
	}

	log.Printf("[MIGRATIONS] - All migrations executed successfully (%d file(s))", len(files))
	return nil
}
