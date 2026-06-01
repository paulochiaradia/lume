package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/db"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Lume — Migrate Service           ║")
	fmt.Println("╚══════════════════════════════════════╝")

	// Carrega .env em desenvolvimento
	if os.Getenv("ENV") != "production" {
		godotenv.Load("../.env")
	}

	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("erro ao conectar: %v", err)
	}
	defer conn.Close()

	if err := runMigrations(conn); err != nil {
		log.Fatalf("erro ao rodar migrations: %v", err)
	}

	log.Println("migrations concluídas com sucesso")
}

func runMigrations(conn *sql.DB) error {
	// Busca todos os arquivos SQL da pasta de migrations
	pattern := "/app/migrations/*.sql"
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("erro ao listar migrations: %w", err)
	}

	if len(files) == 0 {
		log.Println("nenhuma migration encontrada")
		return nil
	}

	// Garante execução em ordem numérica
	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)

		// Verifica se já foi executada
		var count int
		err := conn.QueryRow(
			"SELECT COUNT(*) FROM lume_system.schema_migrations WHERE version = $1",
			version,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("erro ao verificar migration %s: %w", version, err)
		}

		if count > 0 {
			log.Printf("migration %s já aplicada — pulando", version)
			continue
		}

		// Lê e executa o arquivo SQL
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("erro ao ler %s: %w", version, err)
		}

		log.Printf("aplicando migration %s...", version)

		// Executa dentro de uma transação — se falhar, faz rollback
		tx, err := conn.Begin()
		if err != nil {
			return fmt.Errorf("erro ao iniciar transação: %w", err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			// Ignora erros de "já existe" — idempotência
			if strings.Contains(err.Error(), "already exists") {
				log.Printf("migration %s — objetos já existem, continuando", version)
			} else {
				return fmt.Errorf("erro ao executar %s: %w", version, err)
			}
		}

		// Registra como aplicada
		if _, err := tx.Exec(
			"INSERT INTO lume_system.schema_migrations (version) VALUES ($1)",
			version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("erro ao registrar migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("erro ao commitar migration %s: %w", version, err)
		}

		log.Printf("migration %s aplicada com sucesso", version)
	}

	return nil
}
