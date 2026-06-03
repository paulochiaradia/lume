package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/paulochiaradia/lume/collector/internal/connector"
	"github.com/paulochiaradia/lume/collector/internal/db"
	"github.com/paulochiaradia/lume/collector/internal/loader"
	"github.com/paulochiaradia/lume/collector/internal/normalizer"
)

func runPipelineTest() {
	log.Println("iniciando pipeline com dados sinteticos")

	if os.Getenv("ENV") != "production" {
		godotenv.Load("../../.env")
	}

	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer conn.Close()

	clientKey := "loja_teste"
	l := loader.New(conn, clientKey)

	// ── Clientes ─────────────────────────────────────────────
	fmt.Println("\n-- Clientes --")
	clientesCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/clientes_sinteticos.csv",
	}
	clientesConn, _ := connector.Factory(clientesCfg)
	rawClientes, _ := clientesConn.Extract()
	clientes, _ := normalizer.NormalizeClientes(rawClientes)
	writtenClientes, _ := l.LoadClientes(clientes)
	log.Printf("clientes: %d gravados", writtenClientes)

	// ── Produtos ─────────────────────────────────────────────
	fmt.Println("\n-- Produtos --")
	produtosCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/produtos_sinteticos.csv",
	}
	produtosConn, _ := connector.Factory(produtosCfg)
	rawProdutos, _ := produtosConn.Extract()
	produtos, _ := normalizer.NormalizeProdutos(rawProdutos)
	writtenProdutos, _ := l.LoadProdutos(produtos)
	log.Printf("produtos: %d gravados", writtenProdutos)

	// ── Vendas ───────────────────────────────────────────────
	fmt.Println("\n-- Vendas --")
	vendasCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/vendas_sinteticas.csv",
	}
	vendasConn, _ := connector.Factory(vendasCfg)
	rawVendas, _ := vendasConn.Extract()
	vendas, errosVendas := normalizer.NormalizeVendas(rawVendas)
	log.Printf("vendas: %d normalizadas (%d erros)", len(vendas), len(errosVendas))
	writtenVendas, _ := l.LoadVendas(vendas)
	log.Printf("vendas: %d gravadas", writtenVendas)

	// ── Itens de Venda ───────────────────────────────────────
	fmt.Println("\n-- Itens de Venda --")
	itensCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/itens_venda_sinteticos.csv",
	}
	itensConn, _ := connector.Factory(itensCfg)
	rawItens, _ := itensConn.Extract()
	itens, _ := normalizer.NormalizeItensVenda(rawItens)
	writtenItens, _ := l.LoadItensVenda(itens)
	log.Printf("itens: %d gravados", writtenItens)

	fmt.Println("\nPipeline concluido com sucesso!")
}
