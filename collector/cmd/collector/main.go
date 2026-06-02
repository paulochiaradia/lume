package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/api"
	"github.com/paulochiaradia/lume/collector/internal/db"
	"github.com/paulochiaradia/lume/collector/internal/scheduler"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Lume — Collector Service         ║")
	fmt.Println("╚══════════════════════════════════════╝")

	log.Printf("ambiente: %s", env)

	if env != "production" {
		godotenv.Load("../../.env")
	}

	// Conecta no banco
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer conn.Close()

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

	// Inicia o servidor HTTP em background
	server := api.New(conn)
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
