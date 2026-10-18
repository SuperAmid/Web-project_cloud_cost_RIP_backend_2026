package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// applyMigrations выполняет версионированные SQL-файлы migrations/*.up.sql по порядку
// и записывает выполненные версии в таблицу schema_migrations.
// Если таблицы уже созданы раньше (ЛР2 до исправлений), 001_init считается выполненной.
func applyMigrations(db *gorm.DB, dir string) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(80) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`).Error; err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	var applied int64
	if err := db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&applied).Error; err != nil {
		return err
	}
	var existing bool
	if err := db.Raw("SELECT to_regclass('public.provider_routers') IS NOT NULL").Scan(&existing).Error; err != nil {
		return err
	}
	if applied == 0 && existing && len(files) > 0 {
		baseline := strings.TrimSuffix(filepath.Base(files[0]), ".up.sql")
		if err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", baseline).Error; err != nil {
			return err
		}
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), ".up.sql")
		var done int64
		if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&done).Error; err != nil {
			return err
		}
		if done > 0 {
			continue
		}
		sql, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sql)).Error; err != nil {
				return err
			}
			return tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", version, time.Now()).Error
		})
		if err != nil {
			return fmt.Errorf("миграция %s: %w", version, err)
		}
	}
	return nil
}
