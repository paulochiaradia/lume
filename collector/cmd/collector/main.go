package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/api"
	"github.com/paulochiaradia/lume/collector/internal/db"
	"github.com/paulochiaradia/lume/collector/internal/scheduler"
	"github.com/redis/go-redis/v9"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║      Lume — Collector Service        ║")
	fmt.Println("╚══════════════════════════════════════╝")

	log.Printf("ambiente: %s", env)

	if env != "production" {
		godotenv.Load("../../.env")
	}

	// ── Conecta no PostgreSQL ─────────────────────────────────
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("erro ao conectar no banco PostgreSQL: %v", err)
	}
	defer conn.Close()

	// ── Conecta no Redis ──────────────────────────────────────
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Padrão seguro para rodar fora do Docker
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // Usa o banco padrão do Redis
	})

	// Testa se o Redis está vivo antes de subir a aplicação
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("erro ao conectar no banco Redis: %v", err)
	}
	defer rdb.Close()

	log.Println("redis: conexão estabelecida com sucesso")

	// Em desenvolvimento roda o teste do pipeline
	if env == "development" {
		runPipelineTest()
	}

	// Inicia o scheduler em background
	s := scheduler.New(conn)
	if err := s.Start(); err != nil {
		log.Fatalf("erro ao iniciar scheduler: %v", err)
	}
	defer s.Stop()

	// Inicia o servidor HTTP em background (agora com PostgreSQL e Redis)
	server := api.New(conn, rdb)
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("erro no servidor HTTP: %v", err)
		}
	}()

	log.Println("collector rodando — pressione Ctrl+C para parar")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("encerrando collector...")
	server.Stop()
}
