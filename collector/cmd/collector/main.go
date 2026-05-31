package main

import (
	"fmt"
	"log"
	"os"
	"time"
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
	log.Printf("collector iniciado com sucesso")
	log.Printf("aguardando implementação dos conectores...")

	// Mantém o processo rodando — será substituído pelo scheduler
	for {
		time.Sleep(time.Hour)
	}
}
