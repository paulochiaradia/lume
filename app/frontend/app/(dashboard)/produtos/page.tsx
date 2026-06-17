"use client"

import { useState, useEffect } from "react"
import Header from "@/components/layout/Header"
import api  from "@/lib/api"
import { 
  Package, TrendingUp, AlertOctagon, Info, 
  Zap, AlertCircle, Loader2 
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

type DeadStockProduto = {
  nome: string
  quantidade: number
  preco_custo: number
  capital_parado: number
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
  
  // Estados
  const [kpis, setKpis] = useState<CatalogKPIs | null>(null)
  const [matrizData, setMatrizData] = useState<MatrizABCXYZ[]>([])
  const [rankingData, setRankingData] = useState<RankingProduto[]>([])
  const [elasticidadeData, setElasticidadeData] = useState<ElasticidadeProduto[]>([])
  const [basketData, setBasketData] = useState<RegraAssociacao[]>([])
  const [deadStockData, setDeadStockData] = useState<DeadStockProduto[]>([])
  const [insightsData, setInsightsData] = useState<Insight[]>([])

  useEffect(() => {
    async function fetchProdutosDashboard() {
      setLoading(true)
      try {
        const [
          kpisRes,
          matrizRes,
          rankingRes,
          elasticidadeRes,
          basketRes,
          deadStockRes,
          insightsRes
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

  return (
    <>
      <Header title="Catálogo e Produtos" />
      
      <div className="flex flex-col gap-6">

        {/* ── BLOCO 1: KPIs DO CATÁLOGO ──────────────────────────────── */}
        <div className="flex flex-col gap-3">
          {loading || !kpis ? (
            <div className="h-[140px] rounded-xl flex items-center justify-center border" style={{ backgroundColor: "var(--color-surface-container-lowest)", borderColor: "var(--color-outline-variant)" }}>
               <Loader2 size={24} className="animate-spin text-blue-500" />
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <KpiProductCard
                label="Mix de Produtos"
                value={`${kpis.total_skus}`}
                subtitle={<><strong className="text-emerald-600 dark:text-emerald-400 font-semibold">{kpis.qtd_classe_a} produtos</strong> representam {(kpis.pct_faturamento_a * 100).toFixed(1)}% da receita.</>}
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
                label="Dead Stock (90 dias)"
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

        {/* ── BLOCO 3: MATRIZ ABC × XYZ ──────────────────────────────── */}
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
                  const item = matrizData.find(m => m.classe_abc === 'A' && m.classe_xyz === xyz);
                  const bg = xyz === 'X' ? 'bg-emerald-200/50 dark:bg-emerald-800/40' : xyz === 'Y' ? 'bg-emerald-100/50 dark:bg-emerald-900/20' : 'bg-amber-100/50 dark:bg-amber-900/20';
                  return (
                    <div key={`A${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{item?.qtd || 0}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}

                {/* Linha B */}
                <div className="flex items-center justify-end pr-4 font-semibold text-amber-700 text-sm">B (Médio Valor)</div>
                {['X', 'Y', 'Z'].map(xyz => {
                  const item = matrizData.find(m => m.classe_abc === 'B' && m.classe_xyz === xyz);
                  const bg = xyz === 'X' ? 'bg-emerald-100/50 dark:bg-emerald-900/20' : xyz === 'Y' ? 'bg-amber-200/50 dark:bg-amber-800/40' : 'bg-red-100/50 dark:bg-red-900/20';
                  return (
                    <div key={`B${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{item?.qtd || 0}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}

                {/* Linha C */}
                <div className="flex items-center justify-end pr-4 font-semibold text-red-700 text-sm">C (Baixo Valor)</div>
                {['X', 'Y', 'Z'].map(xyz => {
                  const item = matrizData.find(m => m.classe_abc === 'C' && m.classe_xyz === xyz);
                  const bg = xyz === 'X' ? 'bg-amber-100/50 dark:bg-amber-900/20' : xyz === 'Y' ? 'bg-red-100/50 dark:bg-red-900/20' : 'bg-red-200/50 dark:bg-red-800/40';
                  return (
                    <div key={`C${xyz}`} className={`flex flex-col items-center justify-center py-5 rounded-lg border border-[var(--color-outline-variant)] ${bg}`}>
                      <span className="text-2xl font-bold" style={{ color: "var(--color-on-surface)" }}>{item?.qtd || 0}</span>
                      <span className="text-[10px] font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>SKUs</span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>

      </div>
    </>
  )
}