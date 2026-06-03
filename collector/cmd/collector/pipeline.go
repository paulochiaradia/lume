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
	log.Println("iniciando teste do pipeline CSV → normalizer → PostgreSQL")

	// Carrega .env
	if os.Getenv("ENV") != "production" {
		godotenv.Load("../../.env")
	}

	// Conecta no banco
	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer conn.Close()

	clientKey := "loja_teste"

	// ── Teste de Vendas ──────────────────────────────────────
	fmt.Println("\n── Testando pipeline de vendas ──")

	vendaCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/vendas_teste.csv",
	}

	vendaConnector, err := connector.Factory(vendaCfg)
	if err != nil {
		log.Fatalf("erro ao criar conector: %v", err)
	}

	if err := vendaConnector.Validate(); err != nil {
		log.Fatalf("erro na validação: %v", err)
	}

	rawVendas, err := vendaConnector.Extract()
	if err != nil {
		log.Fatalf("erro ao extrair vendas: %v", err)
	}
	log.Printf("extraídos %d registros brutos de vendas", len(rawVendas))

	vendas, erros := normalizer.NormalizeVendas(rawVendas)
	log.Printf("normalizadas %d vendas (%d erros)", len(vendas), len(erros))
	for _, e := range erros {
		log.Printf("  erro: %s", e)
	}

	l := loader.New(conn, clientKey)
	written, err := l.LoadVendas(vendas)
	if err != nil {
		log.Fatalf("erro ao carregar vendas: %v", err)
	}
	log.Printf("gravadas %d vendas no banco", written)

	// ── Teste de Produtos ────────────────────────────────────
	fmt.Println("\n── Testando pipeline de produtos ──")

	produtoCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/produtos_teste.csv",
	}

	produtoConnector, _ := connector.Factory(produtoCfg)
	rawProdutos, err := produtoConnector.Extract()
	if err != nil {
		log.Fatalf("erro ao extrair produtos: %v", err)
	}
	log.Printf("extraídos %d registros brutos de produtos", len(rawProdutos))

	produtos, errosProd := normalizer.NormalizeProdutos(rawProdutos)
	log.Printf("normalizados %d produtos (%d erros)", len(produtos), len(errosProd))

	writtenProd, err := l.LoadProdutos(produtos)
	if err != nil {
		log.Fatalf("erro ao carregar produtos: %v", err)
	}
	log.Printf("gravados %d produtos no banco", writtenProd)

	// ── Teste de Itens de Venda ──────────────────────────────────
	fmt.Println("\n── Testando pipeline de itens de venda ──")

	itensCfg := connector.Config{
		ClientKey: clientKey,
		ERPType:   "csv",
		FilePath:  "/app/testdata/itens_venda_teste.csv",
	}

	itensConnector, _ := connector.Factory(itensCfg)
	rawItens, err := itensConnector.Extract()
	if err != nil {
		log.Fatalf("erro ao extrair itens: %v", err)
	}
	log.Printf("extraídos %d registros brutos de itens", len(rawItens))

	itens, errosItens := normalizer.NormalizeItensVenda(rawItens)
	log.Printf("normalizados %d itens (%d erros)", len(itens), len(errosItens))

	writtenItens, err := l.LoadItensVenda(itens)
	if err != nil {
		log.Fatalf("erro ao carregar itens: %v", err)
	}
	log.Printf("gravados %d itens no banco", writtenItens)

	fmt.Println("\n✅ Pipeline testado com sucesso!")
}
