package main

import (
	"fmt"
	"log"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     Lume — Migrate Service           ║")
	fmt.Println("╚══════════════════════════════════════╝")

	log.Printf("sistema de migrations será implementado")
}
