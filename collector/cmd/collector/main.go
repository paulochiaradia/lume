package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/api"
	"github.com/paulochiaradia/lume/collector/internal/db"
	"github.com/paulochiaradia/lume/collector/internal/mailer"
	"github.com/paulochiaradia/lume/collector/internal/scheduler"
	"github.com/redis/go-redis/v9"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Mostra tudo de Info para cima (Warn, Error)
	}))
	// Define este logger como o padrão global da aplicação
	slog.SetDefault(logger)

	slog.Info("iniciando Lume API...", slog.String("env", env))

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║      Lume — Collector Service        ║")
	fmt.Println("╚══════════════════════════════════════╝")

	slog.Info("ambiente", slog.String("env", env))

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
		slog.Error("erro ao conectar no banco Redis", slog.Any("err", err))
		os.Exit(1)
	}
	defer rdb.Close()

	slog.Info("redis: conexão estabelecida com sucesso")

	// Em desenvolvimento roda o teste do pipeline
	if env == "development" {
		runPipelineTest()
	}

	// Inicia o scheduler em background
	s := scheduler.New(conn)
	if err := s.Start(); err != nil {
		slog.Error("erro ao iniciar scheduler", slog.Any("err", err))
		os.Exit(1)
	}
	defer s.Stop()

	//Inicia o serviço de envio de e-mails
	mailService, err := mailer.New(os.Getenv("RESEND_API_KEY"))
	if err != nil {
		slog.Error("erro ao iniciar serviço de e-mail", slog.Any("err", err))
		os.Exit(1)
	}

	// Inicia o servidor HTTP em background (agora com PostgreSQL, Redis e mailer)
	server := api.New(conn, rdb, mailService)
	go func() {
		if err := server.Start(); err != nil {
			slog.Error("erro no servidor HTTP", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	slog.Info("collector rodando — pressione Ctrl+C para parar")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("encerrando collector...")
	server.Stop()
}
