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

	// Em desenvolvimento roda o teste do pipeline
	if env == "development" {
		runPipelineTest()
	}

	log.Printf("collector iniciado com sucesso")
	log.Printf("aguardando implementação do scheduler...")

	for {
		time.Sleep(time.Hour)
	}
}
