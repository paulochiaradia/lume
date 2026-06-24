"use client"

import { useState, useEffect } from "react"
import Header from "@/components/layout/Header"
import api  from "@/lib/api"
import { 
  Package, TrendingUp, AlertOctagon, Info, 
  Zap, AlertCircle, Loader2, Trophy, TrendingDown, Minus,
  Link, Plus, ChevronLeft, ChevronRight 
} from "lucide-react"

// ── Tipos do Painel de Produtos ──────────────────────────────
type CatalogKPIs = {
  total_skus: number
  qtd_classe_a: number
  pct_faturamento_a: number
  margem_media: number
  dead_stock_skus: number
  capital_parado: number
}

type MatrizABCXYZ = {
  classe_abc: string
  classe_xyz: string
  qtd: number
}

type RankingProduto = {
  id: string
  nome: string
  categoria: string
  faturamento: number
  margem: number
  classe_abc: string
  classe_xyz: string
  tendencia: "subindo" | "estavel" | "caindo"
}

type ElasticidadeProduto = {
  produto_key: string
  elasticidade: number
  interpretacao: string
  receita: number
}

type RegraAssociacao = {
  antecedents: string
  consequents: string
  confidence: number
  lift: number
}

export interface DeadStockProduto {
  nome: string;
  quantidade: number;
  preco_custo: number;
  capital_parado: number;
}

type Insight = {
  tipo: string
  prioridade: number
  titulo: string
  mensagem: string
  acao?: string
  href?: string
  categoria: string
  icone: string
  gerado_em?: string
}

// ── Helpers de Formatação ────────────────────────────────────
const fmt = (v: number) => new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(v)

// ── Componente: KPI Card de Produtos ─────────────────────────
function KpiProductCard({ label, value, subtitle, icon, accent, valueColor }: {
  label: string; value: string; subtitle: React.ReactNode; icon: React.ReactNode; accent?: string; valueColor?: string
}) {
  return (
    <div className="rounded-xl p-5 flex flex-col gap-4 border"
      style={{ backgroundColor: "var(--color-surface-container-lowest)", borderColor: "var(--color-outline-variant)", boxShadow: "0px 1px 4px rgba(15,23,42,0.06)" }}>
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>
          {label}
        </span>
        <div className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style={{ backgroundColor: accent ?? "var(--color-surface-container-low)" }}>
          {icon}
        </div>
      </div>
      <div className="flex flex-col gap-1">
        <p className="text-2xl font-bold tracking-tight" style={{ fontFamily: "var(--font-display)", color: valueColor ?? "var(--color-on-surface)" }}>
          {value}
        </p>
        <p className="text-xs leading-relaxed" style={{ color: "var(--color-on-surface-variant)" }}>
          {subtitle}
        </p>
      </div>
    </div>
  )
}

// ── Página Principal ─────────────────────────────────────────
export default function ProdutosPage() {
  const [loading, setLoading] = useState(true)
  
  // ── Estados de Dados Globais ──
  const [kpis, setKpis] = useState<CatalogKPIs | null>(null)
  const [matrizData, setMatrizData] = useState<MatrizABCXYZ[]>([])
  const [rankingData, setRankingData] = useState<RankingProduto[]>([])
  const [elasticidadeData, setElasticidadeData] = useState<ElasticidadeProduto[]>([])
  const [basketData, setBasketData] = useState<RegraAssociacao[]>([])
  const [deadStockData, setDeadStockData] = useState<DeadStockProduto[]>([])
  const [insightsData, setInsightsData] = useState<Insight[]>([])

  // ── Estados de UI e Filtros ──
  const [quadranteAtivo, setQuadranteAtivo] = useState<"chama-cliente" | "ouro-oculto" | "gira-estoque" | "margem-segura">("chama-cliente")
  const [skuSimulado, setSkuSimulado] = useState<string>("")
  const [acaoSimulada, setAcaoSimulada] = useState<"subir" | "baixar" | null>(null)
  const [isDeadStockExpanded, setIsDeadStockExpanded] = useState(false)
  const [deadStockPage, setDeadStockPage] = useState(0)

  // Estados do Filtro em Cascata
  const [categoriaSelecionada, setCategoriaSelecionada] = useState<string>("todas");
  const [produtoSelecionado, setProdutoSelecionado] = useState<string>("todos");

  useEffect(() => {
    async function fetchProdutosDashboard() {
      setLoading(true)
      try {
        const [
          kpisRes, matrizRes, rankingRes, elasticidadeRes, 
          basketRes, deadStockRes, insightsRes
        ] = await Promise.all([
          api.get("/produtos/kpis"),
          api.get("/produtos/matriz"),
          api.get("/produtos/ranking"),
          api.get("/produtos/elasticidade"),
          api.get("/produtos/basket"),
          api.get("/produtos/dead-stock"),
          api.get("/vendas/insights?categoria=produtos")
        ])

        setKpis(kpisRes.data)
        setMatrizData(matrizRes.data || [])
        setRankingData(rankingRes.data || [])
        setElasticidadeData(elasticidadeRes.data || [])
        if(elasticidadeRes.data?.length > 0) setSkuSimulado(elasticidadeRes.data[0].produto_key)
        setBasketData(basketRes.data || [])
        setDeadStockData(deadStockRes.data || [])
        setInsightsData(insightsRes.data || [])

      } catch (err) {
        console.error("Erro ao carregar dados do painel de produtos:", err)
      } finally {
        setLoading(false)
      }
    }
    fetchProdutosDashboard()
  }, [])

  // ── LÓGICA DE DADOS DERIVADOS (FILTRAGEM CASCATA DINÂMICA) ──────────
  
  // 1. Extrai as Categorias Únicas do catálogo bruto
  const categoriasUnicas = Array.from(new Set(rankingData.map(item => item.categoria))).filter(Boolean).sort();

  // 2. Prepara a lista de Produtos para o segundo dropdown
  const produtosDisponiveis = categoriaSelecionada === "todas" 
    ? rankingData 
    : rankingData.filter(item => item.categoria === categoriaSelecionada);

  // 3. Obtém o Ranking Final que vai abastecer a tela
  const rankingFiltrado = produtosDisponiveis.filter(item => 
    produtoSelecionado === "todos" ? true : item.id === produtoSelecionado
  );

  // 4. Cria um "Set" de SKUs permitidos para filtrar todos os outros blocos da página com altíssima performance
  const skusFiltrados = new Set(rankingFiltrado.map(item => item.id));

  // 5. Filtra o resto do ecossistema de dados para exibir apenas o selecionado
  const elasticidadeFiltrada = elasticidadeData.filter(item => skusFiltrados.has(item.produto_key));
  const basketFiltrado = basketData.filter(item => skusFiltrados.has(item.antecedents) || skusFiltrados.has(item.consequents));
  
  const deadStockFiltrado = deadStockData.filter(item => {
    const prod = rankingData.find(r => r.nome === item.nome);
    return prod && skusFiltrados.has(prod.id);
  });


  return (
    <>
      <Header title="Catálogo e Produtos" />
      
      <div className="flex flex-col gap-6">

        {/* ── BARRA DE FERRAMENTAS / FILTRO CASCATA ────────────────────── */}
        <div className="flex justify-end w-full mb-2">
          <div className="flex flex-col sm:flex-row items-center bg-[var(--color-surface-container-lowest)] border border-[var(--color-outline-variant)] rounded-lg p-1.5 shadow-sm">
            
            {/* Filtro 1: Categoria */}
            <div className="flex items-center gap-2 px-3 border-b sm:border-b-0 sm:border-r border-[var(--color-outline-variant)] w-full sm:w-auto pb-2 sm:pb-0">
              <label className="text-[10px] font-bold text-[var(--color-on-surface-variant)] uppercase tracking-widest whitespace-nowrap">
                Categoria:
              </label>
              <select 
                value={categoriaSelecionada} 
                onChange={(e) => {
                  setCategoriaSelecionada(e.target.value);
                  setProdutoSelecionado("todos"); // Reseta o produto ao mudar categoria
                }}
                className="bg-transparent text-sm font-bold text-blue-600 dark:text-blue-400 outline-none border-none cursor-pointer focus:ring-0 w-full sm:max-w-[180px] truncate"
              >
                <option value="todas">Todas as Categorias</option>
                {categoriasUnicas.map(cat => (
                  <option key={cat} value={cat}>{cat}</option>
                ))}
              </select>
            </div>

            {/* Filtro 2: Produto */}
            <div className="flex items-center gap-2 px-3 pt-2 sm:pt-0 w-full sm:w-auto">
              <label className="text-[10px] font-bold text-[var(--color-on-surface-variant)] uppercase tracking-widest whitespace-nowrap">
                Produto:
              </label>
              <select 
                value={produtoSelecionado} 
                onChange={(e) => setProdutoSelecionado(e.target.value)}
                className="bg-transparent text-sm font-bold text-emerald-600 dark:text-emerald-400 outline-none border-none cursor-pointer focus:ring-0 w-full sm:max-w-[220px] truncate"
              >
                <option value="todos">Todos os Produtos</option>
                {produtosDisponiveis.map(prod => (
                  <option key={prod.id} value={prod.id}>{prod.nome}</option>
                ))}
              </select>
            </div>

          </div>
        </div>
        
        {/* ── BLOCO 1: KPIs DO CATÁLOGO ──────────────────────────────── */}
        <div className="flex flex-col gap-3">
          {loading || !kpis ? (
            <div className="h-[140px] rounded-xl flex justify-center items-center bg-[var(--color-surface-container-lowest)] border border-[var(--color-outline-variant)]">
               <Loader2 size={24} className="animate-spin text-blue-500" />
            </div>) : (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <KpiProductCard
                label="Mix de Produtos (Geral)"
                value={`${kpis.total_skus}`}
                subtitle={<><strong className="text-emerald-600 dark:text-emerald-400 font-semibold">{kpis.qtd_classe_a} produtos</strong> representam {kpis.pct_faturamento_a.toFixed(1).replace('.', ',')}% da receita.</>}
                icon={<Package size={16} color="var(--color-secondary)" />}
                accent="var(--color-primary-fixed)"
              />
              <KpiProductCard
                label="Margem Média do Catálogo"
                value={`${kpis.margem_media.toFixed(1)}%`}
                subtitle="Média geral baseada no custo e preço de venda atual."
                icon={<TrendingUp size={16} color="var(--color-on-surface-variant)" />}
                accent="var(--color-surface-container)"
              />
              <KpiProductCard
                label="Dead Stock (Geral 90 dias)"
                value={fmt(kpis.capital_parado)}
                valueColor="rgb(239, 68, 68)"
                subtitle={<>Capital travado em <strong className="font-semibold">{kpis.dead_stock_skus} produtos</strong> sem giro recente.</>}
                icon={<AlertOctagon size={16} color="rgb(239, 68, 68)" />}
                accent="rgba(239, 68, 68, 0.1)"
              />
            </div>
          )}
        </div>

        {/* ── BLOCO 2: INSIGHTS ──────────────────────────────────────── */}
        {!loading && insightsData.length > 0 && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {insightsData.map((insight, idx) => {
              const cfg = {
                success: { bg:"var(--color-primary-fixed)", border:"var(--color-primary-fixed-dim)", iconBg:"var(--color-secondary)", lc:"var(--color-on-primary-fixed-variant)", icon:<Zap size={15} color="#fff" /> },
                warning: { bg:"#fffbeb", border:"#fde68a", iconBg:"#f59e0b", lc:"#92400e", icon:<AlertCircle size={15} color="#fff" /> },
                danger:  { bg:"#fef2f2", border:"#fca5a5", iconBg:"#ef4444", lc:"#991b1b", icon:<AlertOctagon size={15} color="#fff" /> }
              }[insight.icone] || { bg:"var(--color-surface-container)", border:"var(--color-outline)", iconBg:"var(--color-on-surface-variant)", lc:"var(--color-on-surface)", icon:<Info size={15} color="#fff" /> }

              return (
                <div key={idx} className="rounded-xl border p-5 flex flex-col gap-3 shadow-sm transition-all hover:shadow-md" style={{ backgroundColor: cfg.bg, borderColor: cfg.border }}>
                  <div className="flex items-center gap-2">
                    <div className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ backgroundColor: cfg.iconBg }}>{cfg.icon}</div>
                    <span className="text-xs font-semibold uppercase tracking-wide" style={{ color: cfg.lc }}>{insight.titulo}</span>
                  </div>
                  <p className="text-sm leading-relaxed flex-1" style={{ color: "var(--color-on-surface)" }}>{insight.mensagem}</p>
                </div>
              )
            })}
          </div>
        )}

        {/* ── BLOCO 3: MATRIZ ABC × XYZ (Recalculada Dinamicamente) ───── */}
        <div className="rounded-xl p-6 border flex flex-col gap-5" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="flex flex-col gap-1">
            <h3 className="text-xs font-semibold uppercase tracking-wide flex items-center gap-2" style={{ color:"var(--color-on-surface-variant)" }}>
              Matriz de Estratégia (ABC × XYZ)
              <span title="Combina representatividade financeira (ABC) com previsibilidade de demanda (XYZ)" className="flex items-center cursor-help">
                <Info size={14} />
              </span>
            </h3>
            <p className="text-sm" style={{ color: "var(--color-on-surface-variant)" }}>
              Classificação cruzada do seu catálogo. Priorize proteger o estoque dos itens AX e descontinuar os itens CZ.
            </p>
          </div>

          <div className="overflow-x-auto">
            {loading ? (
              <div className="h-64 flex items-center justify-center">
                <Loader2 size={24} className="animate-spin text-blue-500" />
              </div>
            ) : (
              <div className="min-w-[600px] grid grid-cols-4 gap-2">
                {/* Cabeçalho X (XYZ) */}
                <div className="col-span-1"></div>
                <div className="text-center font-semibold text-emerald-700 bg-emerald-100/50 dark:bg-emerald-900/20 py-2 rounded-lg text-sm">X (Demanda Estável)</div>
                <div className="text-center font-semibold text-amber-700 bg-amber-100/50 dark:bg-amber-900/20 py-2 rounded-lg text-sm">Y (Demanda Variável)</div>
                <div className="text-center font-semibold text-red-700 bg-red-100/50 dark:bg-red-900/20 py-2 rounded-lg text-sm">Z (Demanda Errática)</div>

                {/* Linha A */}
                <div className="flex items-center justify-end pr-4 font-semibold text-emerald-700 text-sm">A (Alto Valor)</div>
                {['X', 'Y', 'Z'].map(xyz => {
                  const qtdDinâmica = rankingFiltrado.filter(r => r.classe_abc === 'A' && r.classe_xyz === xyz).length;
                  const bg = xyz === 'X' ? 'bg-emerald-200/50 dark:bg-emerald-800/40' : xyz === 'Y' ? 'bg-emerald-100/50 dark:bg-emerald-900/20' : 'bg-amber-100/50 dark:bg-amber-900/20';
                  return (
                    <div key={`A${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{qtdDinâmica}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}

                {/* Linha B */}
                <div className="flex items-center justify-end pr-4 font-semibold text-amber-700 text-sm">B (Médio Valor)</div>
                {['X', 'Y', 'Z'].map(xyz => {
                  const qtdDinâmica = rankingFiltrado.filter(r => r.classe_abc === 'B' && r.classe_xyz === xyz).length;
                  const bg = xyz === 'X' ? 'bg-emerald-100/50 dark:bg-emerald-900/20' : xyz === 'Y' ? 'bg-amber-200/50 dark:bg-amber-800/40' : 'bg-red-100/50 dark:bg-red-900/20';
                  return (
                    <div key={`B${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{qtdDinâmica}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}

                {/* Linha C */}
                <div className="flex items-center justify-end pr-4 font-semibold text-red-700 text-sm">C (Baixo Valor)</div>
                {['X', 'Y', 'Z'].map(xyz => {
                  const qtdDinâmica = rankingFiltrado.filter(r => r.classe_abc === 'C' && r.classe_xyz === xyz).length;
                  const bg = xyz === 'X' ? 'bg-amber-100/50 dark:bg-amber-900/20' : xyz === 'Y' ? 'bg-red-100/50 dark:bg-red-900/20' : 'bg-red-200/50 dark:bg-red-800/40';
                  return (
                    <div key={`C${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{qtdDinâmica}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>

        {/* ── BLOCO 4: RANKING DE PRODUTOS ────────────────────────────── */}
        <div className="rounded-xl border overflow-hidden mt-2" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
            <div className="flex items-center gap-2">
              <Trophy size={16} color="var(--color-on-surface-variant)" />
              <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                Ranking e Performance de Produtos
              </h3>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)" }}>
                <tr>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Produto</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Categoria</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-center">Matriz</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Faturamento</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Margem</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-center">Tendência</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--color-outline-variant)]">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="h-32">
                      <div className="flex justify-center"><Loader2 size={24} className="animate-spin text-blue-500" /></div>
                    </td>
                  </tr>
                ) : rankingFiltrado.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="h-32 text-center text-[var(--color-on-surface-variant)]">Nenhum produto encontrado com os filtros atuais.</td>
                  </tr>
                ) : (
                  rankingFiltrado.slice(0, 10).map((prod, idx) => (
                    <tr key={prod.id || idx} className="transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20">
                      <td className="px-5 py-3">
                        <div className="flex items-center gap-3">
                          <span className="flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold" 
                                style={{ backgroundColor: idx < 3 ? "#fef08a" : "var(--color-surface-container-high)", color: idx < 3 ? "#854d0e" : "inherit" }}>
                            {idx + 1}
                          </span>
                          <div className="flex flex-col">
                            <span className="font-semibold" style={{ color: "var(--color-on-surface)" }}>
                               {prod.nome}
                            </span>
                            <span className="text-[10px] uppercase font-medium text-[var(--color-on-surface-variant)]">
                              SKU: {prod.id || `P-${idx.toString().padStart(4, '0')}`}
                            </span>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-3 font-medium" style={{ color: "var(--color-on-surface-variant)" }}>
                        {prod.categoria}
                      </td>
                      <td className="px-5 py-3 text-center">
                        <span className="px-2 py-1 rounded-md text-[10px] font-bold tracking-widest bg-[var(--color-surface-container-high)] border border-[var(--color-outline-variant)]" style={{ color: "var(--color-on-surface)" }}>
                          {prod.classe_abc}{prod.classe_xyz}
                        </span>
                      </td>
                      <td className="px-5 py-3 text-right font-semibold text-blue-600 dark:text-blue-400">
                        {fmt(prod.faturamento)}
                      </td>
                      <td className="px-5 py-3 text-right">
                        <span className={`px-2 py-1 rounded-md text-xs font-medium ${prod.margem < 15 ? 'bg-red-100 text-red-700' : 'bg-emerald-100 text-emerald-700'}`}>
                          {prod.margem.toFixed(1).replace('.', ',')}%
                        </span>
                      </td>
                      <td className="px-5 py-3">
                        <div className="flex justify-center items-center gap-1">
                          {prod.tendencia === 'subindo' && <><TrendingUp size={16} className="text-emerald-500" /><span className="text-xs font-medium text-emerald-600 dark:text-emerald-400 hidden lg:inline">Em alta</span></>}
                          {prod.tendencia === 'caindo' && <><TrendingDown size={16} className="text-red-500" /><span className="text-xs font-medium text-red-600 dark:text-red-400 hidden lg:inline">Em queda</span></>}
                          {prod.tendencia === 'estavel' && <><Minus size={16} className="text-slate-400" /><span className="text-xs font-medium text-slate-500 hidden lg:inline">Estável</span></>}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* ── BLOCO 5: MARKET BASKET (Comprados Juntos) ────────────────── */}
        <div className="rounded-xl border overflow-hidden mt-2" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Link size={16} color="var(--color-on-surface-variant)" />
                <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                  Sugestões de Venda Casada (Kits)
                </h3>
              </div>
              <span title="Análise baseada no histórico de vendas." className="cursor-help">
                <Info size={14} className="text-[var(--color-on-surface-variant)]" />
              </span>
            </div>
          </div>
          
          <div className="p-5 bg-[var(--color-surface-container-lowest)]">
            {loading ? (
              <div className="h-32 flex justify-center items-center">
                <Loader2 size={24} className="animate-spin text-blue-500" />
              </div>
            ) : basketFiltrado.length === 0 ? (
              <div className="h-32 flex justify-center items-center text-[var(--color-on-surface-variant)] text-sm">
                Nenhum padrão forte de compra conjunta encontrado para o filtro atual.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {(() => {
                  const regrasUnicas: RegraAssociacao[] = [];
                  const paresVistos = new Set<string>();

                  basketFiltrado.forEach(rule => {
                    const chavePar = [rule.antecedents, rule.consequents].sort().join('|');
                    if (!paresVistos.has(chavePar)) {
                      paresVistos.add(chavePar);
                      regrasUnicas.push(rule);
                    }
                  });

                  return regrasUnicas.slice(0, 6).map((rule, idx) => {
                    let conf = rule.confidence;
                    if (conf > 100) conf = conf / 100;
                    else if (conf < 1 && conf > 0) conf = conf * 100;

                    const proporcao = Math.max(1, Math.round(100 / (conf || 1)));
                    const textoAceitacao = proporcao === 1 ? "Quase todos levam" : `1 a cada ${proporcao} clientes leva`;

                    const getNomeProduto = (sku: string) => {
                      // IMPORTANTE: Busca os nomes no catalogo global original (rankingData)
                      const prod = rankingData.find(r => r.id === sku);
                      return prod ? prod.nome : sku;
                    };

                    return (
                      <div key={idx} className="flex flex-col gap-4 p-4 rounded-lg border bg-[var(--color-surface-container-lowest)] shadow-sm hover:shadow-md transition-shadow" style={{ borderColor:"var(--color-outline-variant)" }}>
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex-1 text-center bg-[var(--color-surface-container-high)] border border-[var(--color-outline-variant)] rounded-lg p-2 flex flex-col items-center justify-center h-20 shadow-sm">
                            <span className="text-[9px] uppercase font-bold text-[var(--color-on-surface-variant)] mb-1">Na compra de</span>
                            <span className="text-xs font-bold text-[var(--color-on-surface)] line-clamp-2" title={getNomeProduto(rule.antecedents)}>
                              {getNomeProduto(rule.antecedents)}
                            </span>
                          </div>
                          
                          <div className="flex-shrink-0 flex items-center justify-center w-6 h-6 rounded-full bg-[var(--color-surface-container-high)] text-[var(--color-on-surface-variant)] shadow-sm">
                            <Plus size={12} strokeWidth={3} />
                          </div>
                          
                          <div className="flex-1 text-center bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-300 dark:border-emerald-700/50 rounded-lg p-2 flex flex-col items-center justify-center h-20 shadow-sm">
                            <span className="text-[9px] uppercase font-bold text-[var(--color-on-surface-variant)] mb-1">Ofereça</span>
                            <span className="text-xs font-bold text-[var(--color-on-surface)] line-clamp-2" title={getNomeProduto(rule.consequents)}>
                              {getNomeProduto(rule.consequents)}
                            </span>
                          </div>
                        </div>

                        <div className="flex flex-col pt-3 border-t border-[var(--color-outline-variant)] gap-2">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-medium text-[var(--color-on-surface-variant)]">Aceitação da oferta:</span>
                            <span className="text-sm font-bold text-emerald-600 dark:text-emerald-400">
                              {conf.toFixed(1).replace('.', ',')}%
                            </span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-medium text-[var(--color-on-surface-variant)]">Na prática:</span>
                            <span className="text-xs font-bold px-2 py-0.5 rounded bg-[var(--color-surface-container-high)] text-[var(--color-on-surface)]">
                              {textoAceitacao}
                            </span>
                          </div>
                        </div>
                      </div>
                    );
                  });
                })()}
              </div>
            )}
          </div>
        </div>
      
        {/* ── BLOCO 6: INTELIGÊNCIA DE PRECIFICAÇÃO AVANÇADA ─────────── */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-4">
          
          {/* COLUNA ESQUERDA: MATRIZ DE ESTRATÉGIA 2x2 */}
          <div className="rounded-xl border p-6 flex flex-col gap-4" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            <div className="flex items-center gap-2 pb-2 border-b border-[var(--color-outline-variant)]">
              <TrendingUp size={16} color="var(--color-on-surface-variant)" />
              <div className="flex flex-col">
                <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                  Matriz de Posicionamento de Preços
                </h3>
                <p className="text-[11px]" style={{ color:"var(--color-on-surface-variant)" }}>Clique em um quadrante para filtrar a lista abaixo</p>
              </div>
            </div>

            {(() => {
              const faturamentos = rankingFiltrado.map(r => r.faturamento);
              const faturamentoMedio = faturamentos.length > 0 ? faturamentos.reduce((a, b) => a + b, 0) / faturamentos.length : 0;

              const listaChamaCliente = elasticidadeFiltrada.filter(item => {
                const prod = rankingData.find(r => r.id === item.produto_key);
                const ehElastico = item.interpretacao.toLowerCase().includes("elást") || item.elasticidade < -1;
                return ehElastico && (prod ? prod.faturamento >= faturamentoMedio : true);
              });

              const listaOuroOculto = elasticidadeFiltrada.filter(item => {
                const prod = rankingData.find(r => r.id === item.produto_key);
                const ehElastico = item.interpretacao.toLowerCase().includes("elást") || item.elasticidade < -1;
                return !ehElastico && (prod ? prod.faturamento >= faturamentoMedio : true);
              });

              const listaGiraEstoque = elasticidadeFiltrada.filter(item => {
                const prod = rankingData.find(r => r.id === item.produto_key);
                const ehElastico = item.interpretacao.toLowerCase().includes("elást") || item.elasticidade < -1;
                return ehElastico && (prod ? prod.faturamento < faturamentoMedio : false);
              });

              const listaMargemSegura = elasticidadeFiltrada.filter(item => {
                const prod = rankingData.find(r => r.id === item.produto_key);
                const ehElastico = item.interpretacao.toLowerCase().includes("elást") || item.elasticidade < -1;
                return !ehElastico && (prod ? prod.faturamento < faturamentoMedio : false);
              });

              return (
                <div className="flex flex-col gap-4">
                  <div className="grid grid-cols-2 gap-3">
                    
                    <button onClick={() => setQuadranteAtivo("ouro-oculto")}
                      className={`p-4 rounded-xl border text-left transition-all ${quadranteAtivo === 'ouro-oculto' ? 'ring-2 ring-emerald-500 border-emerald-500 bg-emerald-50/30' : 'bg-[var(--color-surface-container-low)] border-[var(--color-outline-variant)]'}`}>
                      <div className="flex justify-between items-center mb-1">
                        <span className="text-xs font-bold text-emerald-700 dark:text-emerald-400">💎 Ouro Oculto</span>
                        <span className="text-xs bg-emerald-100 dark:bg-emerald-900/40 text-emerald-800 px-2 py-0.5 rounded-full font-bold">{listaOuroOculto.length}</span>
                      </div>
                      <p className="text-[10px] leading-tight text-[var(--color-on-surface-variant)]">Giro alto e pouca sensibilidade.</p>
                    </button>

                    <button onClick={() => setQuadranteAtivo("chama-cliente")}
                      className={`p-4 rounded-xl border text-left transition-all ${quadranteAtivo === 'chama-cliente' ? 'ring-2 ring-blue-500 border-blue-500 bg-blue-50/30' : 'bg-[var(--color-surface-container-low)] border-[var(--color-outline-variant)]'}`}>
                      <div className="flex justify-between items-center mb-1">
                        <span className="text-xs font-bold text-blue-700 dark:text-blue-400">🔥 Chama-Cliente</span>
                        <span className="text-xs bg-blue-100 dark:bg-blue-900/40 text-blue-800 px-2 py-0.5 rounded-full font-bold">{listaChamaCliente.length}</span>
                      </div>
                      <p className="text-[10px] leading-tight text-[var(--color-on-surface-variant)]">Giro alto e alta sensibilidade.</p>
                    </button>

                    <button onClick={() => setQuadranteAtivo("margem-segura")}
                      className={`p-4 rounded-xl border text-left transition-all ${quadranteAtivo === 'margem-segura' ? 'ring-2 ring-slate-500 border-slate-500 bg-slate-50/30' : 'bg-[var(--color-surface-container-low)] border-[var(--color-outline-variant)]'}`}>
                      <div className="flex justify-between items-center mb-1">
                        <span className="text-xs font-bold text-slate-700 dark:text-slate-400">🛡️ Margem Segura</span>
                        <span className="text-xs bg-slate-200 dark:bg-slate-700 text-slate-800 dark:text-slate-200 px-2 py-0.5 rounded-full font-bold">{listaMargemSegura.length}</span>
                      </div>
                      <p className="text-[10px] leading-tight text-[var(--color-on-surface-variant)]">Giro baixo e pouca sensibilidade.</p>
                    </button>

                    <button onClick={() => setQuadranteAtivo("gira-estoque")}
                      className={`p-4 rounded-xl border text-left transition-all ${quadranteAtivo === 'gira-estoque' ? 'ring-2 ring-orange-500 border-orange-500 bg-orange-50/30' : 'bg-[var(--color-surface-container-low)] border-[var(--color-outline-variant)]'}`}>
                      <div className="flex justify-between items-center mb-1">
                        <span className="text-xs font-bold text-orange-700 dark:text-orange-400">📦 Gira Estoque</span>
                        <span className="text-xs bg-orange-100 dark:bg-orange-900/40 text-orange-800 px-2 py-0.5 rounded-full font-bold">{listaGiraEstoque.length}</span>
                      </div>
                      <p className="text-[10px] leading-tight text-[var(--color-on-surface-variant)]">Giro baixo com alta sensibilidade.</p>
                    </button>

                  </div>

                  <div className="rounded-lg border overflow-hidden" style={{ borderColor:"var(--color-outline-variant)" }}>
                    <div className="bg-[var(--color-surface-container-low)] p-2 text-[11px] font-bold uppercase tracking-wide text-[var(--color-on-surface-variant)] border-b border-[var(--color-outline-variant)]">
                      Produtos no quadrante selecionado:
                    </div>
                    <div className="max-h-[160px] overflow-y-auto divide-y divide-[var(--color-outline-variant)] bg-[var(--color-surface-container-lowest)]">
                      {(() => {
                        const listaAlvo = quadranteAtivo === 'chama-cliente' ? listaChamaCliente 
                                        : quadranteAtivo === 'ouro-oculto' ? listaOuroOculto 
                                        : quadranteAtivo === 'gira-estoque' ? listaGiraEstoque 
                                        : listaMargemSegura;

                        if (listaAlvo.length === 0) return <div className="p-4 text-xs text-center text-[var(--color-on-surface-variant)]">Nenhum produto neste quadrante.</div>;

                        return listaAlvo.map(item => {
                          const p = rankingData.find(r => r.id === item.produto_key);
                          return (
                            <div key={item.produto_key} className="p-3 flex justify-between items-center text-xs">
                              <span className="font-semibold text-[var(--color-on-surface)]">{p ? p.nome : item.produto_key}</span>
                              <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-[var(--color-surface-container-high)] text-[var(--color-on-surface-variant)]">SKU: {item.produto_key}</span>
                            </div>
                          );
                        });
                      })()}
                    </div>
                  </div>
                </div>
              );
            })()}
          </div>

          {/* COLUNA DIREITA: SIMULADOR INTERATIVO "WHAT-IF" */}
          <div className="rounded-xl border p-6 flex flex-col gap-4" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            <div className="flex items-center gap-2 pb-2 border-b border-[var(--color-outline-variant)]">
              <Zap size={16} className="text-blue-500" />
              <div className="flex flex-col">
                <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                  Simulador de Cenários ("O que acontece se...?")
                </h3>
                <p className="text-[11px]" style={{ color:"var(--color-on-surface-variant)" }}>Selecione um produto e simule uma ação de preço no caixa</p>
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-bold text-[var(--color-on-surface-variant)]">Escolha o Produto:</label>
              <select value={skuSimulado} onChange={(e) => { setSkuSimulado(e.target.value); setAcaoSimulada(null); }}
                className="w-full p-2.5 text-xs font-medium rounded-lg border bg-[var(--color-surface-container-low)] border-[var(--color-outline-variant)] text-[var(--color-on-surface)] outline-none focus:border-blue-500">
                {elasticidadeFiltrada.map(item => {
                  const p = rankingData.find(r => r.id === item.produto_key);
                  return (
                    <option key={item.produto_key} value={item.produto_key}>
                      {p ? p.nome : item.produto_key} (SKU: {item.produto_key})
                    </option>
                  );
                })}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3 mt-1">
              <button onClick={() => setAcaoSimulada("subir")}
                className={`py-2 px-3 rounded-lg border text-xs font-bold transition-all ${acaoSimulada === 'subir' ? 'bg-slate-800 text-white border-slate-800' : 'bg-[var(--color-surface-container-high)] text-[var(--color-on-surface)] border-[var(--color-outline-variant)] hover:bg-slate-100 dark:hover:bg-slate-800'}`}>
                📈 Simular Aumento (+5%)
              </button>
              <button onClick={() => setAcaoSimulada("baixar")}
                className={`py-2 px-3 rounded-lg border text-xs font-bold transition-all ${acaoSimulada === 'baixar' ? 'bg-slate-800 text-white border-slate-800' : 'bg-[var(--color-surface-container-high)] text-[var(--color-on-surface)] border-[var(--color-outline-variant)] hover:bg-slate-100 dark:hover:bg-slate-800'}`}>
                📉 Simular Desconto (-5%)
              </button>
            </div>

            <div className="flex-1 flex flex-col justify-center">
              {!acaoSimulada ? (
                <div className="p-4 text-center border-2 border-dashed rounded-xl border-[var(--color-outline-variant)] text-xs text-[var(--color-on-surface-variant)]">
                  Selecione uma das ações acima para rodar a simulação preditiva baseada na elasticidade do produto.
                </div>
              ) : (() => {
                const item = elasticidadeFiltrada.find(e => e.produto_key === skuSimulado);
                if (!item) return <div className="text-xs text-center">Produto não encontrado na análise.</div>;

                const ehElastico = item.interpretacao.toLowerCase().includes("elást") || item.elasticidade < -1;

                if (acaoSimulada === 'subir') {
                  if (ehElastico) {
                    return (
                      <div className="p-4 rounded-xl border border-orange-300 bg-orange-50/40 text-xs flex flex-col gap-1.5 text-orange-900 dark:text-orange-300">
                        <span className="font-bold flex items-center gap-1">⚠️ Risco Alto de Queda no Caixa</span>
                        <p className="leading-relaxed">
                          Como este produto é <strong className="font-bold">altamente sensível a preço</strong>, subir 5% causará um recuo estimado de <strong className="font-bold">-12% no volume de vendas</strong>. Os clientes vão migrar para a concorrência rapidamente, gerando perda líquida de faturamento.
                        </p>
                      </div>
                    );
                  } else {
                    return (
                      <div className="p-4 rounded-xl border border-emerald-300 bg-emerald-50/40 text-xs flex flex-col gap-1.5 text-emerald-900 dark:text-emerald-300">
                        <span className="font-bold flex items-center gap-1">✅ Recomendação Ativa: Margem Extra</span>
                        <p className="leading-relaxed">
                          Excelente oportunidade! Esse produto possui <strong className="font-bold">baixa sensibilidade</strong>. Um reajuste de +5% será absorvido pelo mercado quase sem nenhuma perda de volume. Esse aumento vira <strong className="font-bold">lucro líquido direto para o caixa da empresa</strong>.
                        </p>
                      </div>
                    );
                  }
                } else { 
                  if (ehElastico) {
                    return (
                      <div className="p-4 rounded-xl border border-emerald-300 bg-emerald-50/40 text-xs flex flex-col gap-1.5 text-emerald-900 dark:text-emerald-300">
                        <span className="font-bold flex items-center gap-1">✅ Recomendação Ativa: Alavanca de Giro</span>
                        <p className="leading-relaxed">
                          Gatilho comercial acionado! Um desconto de -5% neste item disparará o volume vendido em cerca de <strong className="font-bold">+15%</strong>. A elasticidade vai compensar a margem menor trazendo muito mais clientes para dentro da loja.
                        </p>
                      </div>
                    );
                  } else {
                    return (
                      <div className="p-4 rounded-xl border border-orange-300 bg-orange-50/40 text-xs flex flex-col gap-1.5 text-orange-900 dark:text-orange-300">
                        <span className="font-bold flex items-center gap-1">⚠️ Alerta: Destruição de Lucro</span>
                        <p className="leading-relaxed">
                          Evite essa ação! O produto possui <strong className="font-bold">baixa sensibilidade</strong>. Dar 5% de desconto <strong className="font-bold">não fará o cliente comprar nenhuma unidade a mais</strong>. Você estará apenas abrindo mão de lucro de um produto que já venderia normalmente.
                        </p>
                      </div>
                    );
                  }
                }
              })()}
            </div>
          </div>         
        </div>

        {/* ── BLOCO 7: DEAD STOCK (RESGATE DE CAPITAL / CARROSSEL) ─────── */}
        <div className="rounded-xl border overflow-hidden mt-6 mb-8" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <AlertOctagon size={16} className="text-red-500" />
                <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                  Painel de Resgate de Capital (Estoque Parado)
                </h3>
              </div>
              <span title="Foca no dinheiro total travado." className="cursor-help">
                <Info size={14} className="text-[var(--color-on-surface-variant)]" />
              </span>
            </div>
          </div>

          <div className="p-6 bg-[var(--color-surface-container-lowest)]">
            {loading ? (
              <div className="h-32 flex justify-center items-center"><Loader2 size={24} className="animate-spin text-blue-500" /></div>
            ) : deadStockFiltrado.length === 0 ? (
              <div className="h-32 flex flex-col justify-center items-center gap-2 text-emerald-600 dark:text-emerald-400">
                 <Trophy size={32} />
                 <span className="font-bold text-sm">Parabéns! Nenhum capital travado.</span>
                 <span className="text-xs text-[var(--color-on-surface-variant)]">Nenhum estoque parado com o filtro selecionado.</span>
              </div>
            ) : (
              (() => {
                const totalTravado = deadStockFiltrado.reduce((acc, curr) => acc + (curr.capital_parado || 0), 0);
                const totalItens = deadStockFiltrado.reduce((acc, curr) => acc + (curr.quantidade || 0), 0);
                
                const sortedDeadStock = [...deadStockFiltrado].sort((a, b) => (b.capital_parado || 0) - (a.capital_parado || 0));
                
                const itensPorPagina = isDeadStockExpanded ? 6 : 3;
                const totalPaginas = Math.ceil(sortedDeadStock.length / itensPorPagina);
                const currentItems = sortedDeadStock.slice(deadStockPage * itensPorPagina, (deadStockPage + 1) * itensPorPagina);

                return (
                  <div className="flex flex-col lg:flex-row gap-8">
                    
                    <div className="lg:w-1/3 rounded-2xl border-2 border-red-500 bg-[var(--color-surface-container-lowest)] p-6 flex flex-col justify-center items-center text-center shadow-sm relative overflow-hidden">
                      <div className="absolute top-0 left-0 w-full h-1.5 bg-red-500"></div>
                      
                      <div className="w-12 h-12 rounded-full border border-red-200 bg-red-50 dark:bg-red-950/30 flex items-center justify-center text-red-600 dark:text-red-500 mb-4 mt-2">
                        <AlertOctagon size={24} />
                      </div>
                      <span className="text-[10px] font-bold uppercase tracking-widest text-[var(--color-on-surface-variant)] mb-1">
                        Dinheiro Travado
                      </span>
                      <h2 className="text-3xl font-extrabold text-red-600 dark:text-red-500 mb-2">
                        {totalTravado.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}
                      </h2>
                      <p className="text-xs font-medium text-[var(--color-on-surface-variant)] mb-6">
                        Preso em {totalItens} unidades
                      </p>
                      <button className="w-full py-3 bg-red-600 hover:bg-red-700 text-white text-xs font-bold rounded-xl shadow-md transition-all active:scale-95 flex justify-center items-center gap-2">
                        <span>Gerar Lista Completa p/ Saldão</span>
                      </button>
                    </div>

                    <div className="lg:w-2/3 flex flex-col justify-start pt-2">
                      <div className="flex justify-between items-end mb-4">
                        <h4 className="text-xs font-bold uppercase text-[var(--color-on-surface-variant)]">
                          {isDeadStockExpanded ? "Todos os Ofensores" : "Maiores Ofensores (Top 3)"}
                        </h4>
                        
                        {isDeadStockExpanded && totalPaginas > 1 && (
                          <div className="flex items-center gap-3">
                            <button 
                              onClick={() => setDeadStockPage(p => Math.max(0, p - 1))}
                              disabled={deadStockPage === 0}
                              className="p-1.5 rounded-md border border-[var(--color-outline-variant)] bg-[var(--color-surface-container-high)] text-[var(--color-on-surface)] disabled:opacity-30 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                            >
                              <ChevronLeft size={16} />
                            </button>
                            <span className="text-[10px] font-bold text-[var(--color-on-surface-variant)]">
                              Página {deadStockPage + 1} de {totalPaginas}
                            </span>
                            <button 
                              onClick={() => setDeadStockPage(p => Math.min(totalPaginas - 1, p + 1))}
                              disabled={deadStockPage === totalPaginas - 1}
                              className="p-1.5 rounded-md border border-[var(--color-outline-variant)] bg-[var(--color-surface-container-high)] text-[var(--color-on-surface)] disabled:opacity-30 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                            >
                              <ChevronRight size={16} />
                            </button>
                          </div>
                        )}
                      </div>

                      <div className={`grid grid-cols-1 md:grid-cols-3 gap-4 ${isDeadStockExpanded ? 'md:grid-rows-2' : ''}`}>
                        {currentItems.map((item, idx) => {
                          // Busca o SKU usando os dados do catálogo base
                          const prod = rankingData.find(r => r.nome === item.nome);
                          const sku = prod ? prod.id : "N/D";
                          
                          return (
                            <div key={idx} className="rounded-xl border bg-[var(--color-surface-container-highest)] border-[var(--color-outline-variant)] p-4 flex flex-col justify-between relative overflow-hidden shadow-sm hover:border-red-300 transition-colors">
                               <div className="absolute top-0 left-0 w-full h-1 bg-red-400 dark:bg-red-500"></div>
                               
                               <div className="flex flex-col mb-4 mt-2">
                                 <span className="text-[10px] font-bold text-red-500 mb-1 flex items-center gap-1">
                                    <AlertOctagon size={10} /> 
                                    {(deadStockPage * itensPorPagina) + idx + 1}º IMPACTO
                                 </span>
                                 <span className="text-sm font-bold text-[var(--color-on-surface)] line-clamp-2 leading-tight mb-1" title={item.nome}>
                                   {item.nome}
                                 </span>
                                 <span className="text-[10px] uppercase font-medium text-[var(--color-on-surface-variant)]">
                                   SKU: {sku}
                                 </span>
                               </div>
                               
                               <div className="flex justify-between items-end border-t border-[var(--color-outline-variant)] pt-3">
                                  <div className="flex flex-col">
                                    <span className="text-[10px] uppercase font-bold text-[var(--color-on-surface-variant)]">Estoque</span>
                                    <span className="text-xs font-bold text-[var(--color-on-surface)]">{item.quantidade} un.</span>
                                  </div>
                                  <div className="flex flex-col text-right">
                                    <span className="text-[10px] uppercase font-bold text-[var(--color-on-surface-variant)]">Dinheiro Parado</span>
                                    <span className="text-sm font-extrabold text-red-600 dark:text-red-400">
                                      {item.capital_parado.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}
                                    </span>
                                  </div>
                               </div>
                            </div>
                          )
                        })}
                      </div>
                      
                      <div className="mt-5 flex justify-center border-t border-[var(--color-outline-variant)] pt-4">
                        {!isDeadStockExpanded && deadStockFiltrado.length > 3 ? (
                          <button 
                            onClick={() => setIsDeadStockExpanded(true)}
                            className="text-xs font-bold text-blue-600 dark:text-blue-400 hover:text-blue-800 transition-colors flex items-center gap-1"
                          >
                            <Plus size={14} />
                            Ver mais {deadStockFiltrado.length - 3} itens parados
                          </button>
                        ) : isDeadStockExpanded ? (
                          <button 
                            onClick={() => { setIsDeadStockExpanded(false); setDeadStockPage(0); }}
                            className="text-xs font-bold text-[var(--color-on-surface-variant)] hover:text-[var(--color-on-surface)] transition-colors flex items-center gap-1"
                          >
                            <Minus size={14} />
                            Recolher painel
                          </button>
                        ) : null}
                      </div>

                    </div>
                  </div>
                );
              })()
            )}
          </div>
        </div>

      </div>
    </>
  )
}