"use client"

import { useEffect, useState } from "react"
import Header from "@/components/layout/Header"
import {
  TrendingUp, ShoppingCart, Tag, Package, // Ícone Package adicionado
  Loader2, HelpCircle, Zap, AlertCircle, TrendingDown, ChevronRight
} from "lucide-react"
import api from "@/lib/api"
import { Insight } from "@/lib/api"
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, BarChart, Bar, Cell, PieChart, Pie
} from "recharts"
import { HomeKPIs, VendaDia, EstoqueItem, VendaHora, MixCategoria } from "@/types"

// ── Tipos ────────────────────────────────────────────────────
type PeriodoComparativo = "week" | "month" | "year"

type ComparativePoint = {
  label: string
  atual: number
  anterior: number
}

type ChartPointProps = {
  cx?: number
  cy?: number
  index?: number
}

const PERIOD_CONFIG: Record<PeriodoComparativo, {
  currentLabel:  string
  previousLabel: string
  title:         string
}> = {
  week:  { currentLabel: "Semana atual",  previousLabel: "Semana passada", title: "Faturamento semanal"  },
  month: { currentLabel: "Mês atual",     previousLabel: "Mês anterior",   title: "Faturamento mensal"   },
  year:  { currentLabel: "Ano atual",     previousLabel: "Ano anterior",   title: "Faturamento anual"    },
}

const MONTH_NAMES  = ["Jan","Fev","Mar","Abr","Mai","Jun","Jul","Ago","Set","Out","Nov","Dez"]
const DAY_NAMES    = ["Dom","Seg","Ter","Qua","Qui","Sex","Sáb"]

// ── Helpers de data ──────────────────────────────────────────
function parseLocal(d: string) { return new Date(`${d}T00:00:00`) }
function fmtKey(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,"0")}-${String(d.getDate()).padStart(2,"0")}`
}
function addDays(d: Date, n: number) {
  const r = new Date(d); r.setDate(r.getDate()+n); return r
}
function startOfWeekMon(d: Date) {
  const r = new Date(d); r.setHours(0,0,0,0)
  const day = r.getDay(); r.setDate(r.getDate() - (day === 0 ? 6 : day - 1))
  return r
}

function buildComparativo(vendas: VendaDia[], periodo: PeriodoComparativo): ComparativePoint[] {
  const entries = vendas
    .map(v => { const d = parseLocal(v.dia); return isNaN(d.getTime()) ? null : { date: d, fat: v.faturamento } })
    .filter((e): e is { date: Date; fat: number } => e !== null)
    .sort((a,b) => a.date.getTime() - b.date.getTime())

  if (!entries.length) return []

  const last = entries[entries.length - 1].date
  const map  = new Map(entries.map(e => [fmtKey(e.date), e.fat]))
  const at   = (d: Date) => map.get(fmtKey(d)) ?? 0

  if (periodo === "week") {
    const ws = startOfWeekMon(last)
    return Array.from({length: 7}, (_, i) => {
      const cur = addDays(ws, i)
      return { label: DAY_NAMES[cur.getDay()], atual: at(cur), anterior: at(addDays(cur, -7)) }
    })
  }

  if (periodo === "month") {
    const ms = new Date(last.getFullYear(), last.getMonth(), 1)
    return Array.from({length: last.getDate()}, (_, i) => {
      const cur = addDays(ms, i)
      const prev = new Date(cur); prev.setMonth(prev.getMonth()-1)
      return { label: String(cur.getDate()), atual: at(cur), anterior: at(prev) }
    })
  }

  // year
  const yr = last.getFullYear()
  const sumMonth = (y: number, m: number) =>
    entries.filter(e => e.date.getFullYear()===y && e.date.getMonth()===m).reduce((s,e)=>s+e.fat, 0)

  return Array.from({length: last.getMonth()+1}, (_, m) => ({
    label: MONTH_NAMES[m], atual: sumMonth(yr, m), anterior: sumMonth(yr-1, m)
  }))
}

// ── Formatação ───────────────────────────────────────────────
function fmt(v: number) {
  return new Intl.NumberFormat("pt-BR", { style:"currency", currency:"BRL", maximumFractionDigits:0 }).format(v)
}
function fmtSafe(v: unknown) {
  const n = Number(v); return isFinite(n) ? fmt(n) : "-"
}

// ── Dot do gráfico ───────────────────────────────────────────
function Dot({ cx, cy, stroke }: { cx?: number; cy?: number; stroke: string }) {
  if (cx == null || cy == null) return <g />
  return <circle cx={cx} cy={cy} r={3} fill="#fff" stroke={stroke} strokeWidth={2} />
}

// ── KPI Card ─────────────────────────────────────────────────
function KpiCard({ label, value, icon, accent }: {
  label: string; value: string; icon: React.ReactNode; accent?: string
}) {
  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-4 border"
      style={{
        backgroundColor: "var(--color-surface-container-lowest)",
        borderColor:     "var(--color-outline-variant)",
        boxShadow:       "0px 1px 4px rgba(15,23,42,0.06)",
      }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-wide"
          style={{ color: "var(--color-on-surface-variant)" }}>
          {label}
        </span>
        <div className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
          style={{ backgroundColor: accent ?? "var(--color-surface-container-low)" }}>
          {icon}
        </div>
      </div>
      <p className="text-2xl font-bold tracking-tight"
        style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}>
        {value}
      </p>
    </div>
  )
}

// ── Tooltip customizado (AreaChart) ──────────────────────────
function CustomTooltip({ active, payload, label, config }: {
  active?: boolean
  payload?: Array<{ value: number; dataKey: string; color: string }>
  label?: string
  config: { currentLabel: string; previousLabel: string }
}) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg px-4 py-3 flex flex-col gap-2"
      style={{ backgroundColor: "#213145", color: "#eaf1ff", fontSize: "13px", minWidth: "160px" }}>
      <p className="font-semibold text-xs uppercase tracking-wide opacity-60">{label}</p>
      {payload.map((p) => (
        <div key={p.dataKey} className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full flex-shrink-0"
              style={{ backgroundColor: p.color }} />
            <span className="text-xs opacity-80">
              {p.dataKey === "atual" ? config.currentLabel : config.previousLabel}
            </span>
          </div>
          <span className="font-semibold text-xs">{fmtSafe(p.value)}</span>
        </div>
      ))}
    </div>
  )
}

const CORES_MIX = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6"]

// ── Página principal ─────────────────────────────────────────
export default function HomePage() {
  const [loading, setLoading] = useState(true)

  // Estados de dados dinâmicos sincronizados com a API
  const [kpis, setKpis] = useState<any>(null) // Utilizado any localmente para não quebrar caso a tipagem HomeKPIs não esteja atualizada no import
  const [vendasComparativo, setVendasComparativo] = useState<VendaDia[]>([])
  const [topDias, setTopDias] = useState<VendaDia[]>([])
  const [alertas, setAlertas] = useState<EstoqueItem[]>([])
  const [insights, setInsights] = useState<Insight[]>([])
  const [vendasHora, setVendasHora] = useState<VendaHora[]>([])
  const [vendasCategoria, setVendasCategoria] = useState<MixCategoria[]>([])
  
  // Filtros Independentes Server-Side (Exclusivos para os gráficos agora)
  const [periodo, setPeriodo] = useState<PeriodoComparativo>("week")
  const [periodoTopDias, setPeriodoTopDias] = useState<PeriodoComparativo>("week")
  const [periodoMix, setPeriodoMix] = useState<PeriodoComparativo>("week")
  const [periodoPico, setPeriodoPico] = useState<PeriodoComparativo>("week")

  const comparativo = buildComparativo(vendasComparativo, periodo)
  const periodoConfig = PERIOD_CONFIG[periodo]

  // Carga inicial estrutural (Alertas, Insights e KPIs MTD)
  useEffect(() => {
    async function load() {
      try {
        const [alertasRes, insightsRes, kpisRes] = await Promise.all([
          api.get("/estoque/alertas"),
          api.get("/insights"),
          api.get("/home/kpis"),
        ])
        setAlertas(alertasRes.data ?? [])
        setInsights(insightsRes.data ?? [])
        setKpis(kpisRes.data)
      } catch (err) {
        console.error("Erro ao buscar dados estruturais", err)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  // Server-Side Trigger: Faturamento com Comparativo
  useEffect(() => {
    async function fetchComparativo() {
      try {
        const res = await api.get(`/vendas/por-dia?periodo=${periodo}`)
        setVendasComparativo(res.data ?? [])
      } catch (err) {
        console.error("Erro ao carregar vendas comparativas", err)
      }
    }
    fetchComparativo()
  }, [periodo])

  // Server-Side Trigger: Top Dias do Período
  useEffect(() => {
    async function fetchTopDias() {
      try {
        const res = await api.get(`/vendas/top-dias?periodo=${periodoTopDias}`)
        setTopDias(res.data ?? [])
      } catch (err) {
        console.error("Erro ao carregar top dias", err)
      }
    }
    fetchTopDias()
  }, [periodoTopDias])

  // Server-Side Trigger: Pico de Atendimento
  useEffect(() => {
    async function fetchPico() {
      try {
        const res = await api.get(`/vendas/por-hora?periodo=${periodoPico}`)
        setVendasHora(res.data ?? [])
      } catch (err) {
        console.error("Erro ao carregar pico de atendimento", err)
      }
    }
    fetchPico()
  }, [periodoPico])

  // Server-Side Trigger: Mix de Categorias
  useEffect(() => {
    async function fetchMix() {
      try {
        const res = await api.get(`/vendas/mix?periodo=${periodoMix}`)
        const dadosComCor = (res.data ?? []).map((item: any, idx: number) => ({
          ...item,
          fill: CORES_MIX[idx % CORES_MIX.length]
        }))
        setVendasCategoria(dadosComCor)
      } catch (err) {
        console.error("Erro ao carregar mix de categorias", err)
      }
    }
    fetchMix()
  }, [periodoMix])

  if (loading) {
    return (
      <>
        <Header title="Dashboard" />
        <div className="flex items-center justify-center h-64">
          <Loader2 size={32} className="animate-spin" style={{ color: "var(--color-secondary)" }} />
        </div>
      </>
    )
  }

  return (
    <>
      <Header title="Dashboard" />
      <div className="flex flex-col gap-6">

        {/* KPIs MTD */}
        {kpis && (
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wide"
                style={{ color: "var(--color-on-surface-variant)" }}>
                Visão Geral (Mês Atual)
              </h2>
            </div>

            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <KpiCard label="Faturamento Total"  value={fmt(kpis.faturamento)}
                icon={<TrendingUp size={16} color="var(--color-secondary)" />}
                accent="var(--color-primary-fixed)" />
              <KpiCard label="Ticket Médio"       value={fmt(kpis.ticket_medio)}
                icon={<Tag size={16} color="var(--color-secondary)" />}
                accent="var(--color-primary-fixed)" />
              <KpiCard label="Total de Vendas"     value={String(kpis.total_vendas)}
                icon={<ShoppingCart size={16} color="var(--color-on-surface-variant)" />}
                accent="var(--color-surface-container)" />
              
              {/* Novo Card: Itens por Venda (UPA) */}
              <KpiCard label="Itens por Venda (UPA)" 
                value={Number(kpis.itens_por_venda).toFixed(1).replace('.', ',')}
                icon={<Package size={16} color="var(--color-secondary)" />}
                accent="var(--color-primary-fixed)" />
            </div>
          </div>
        )}

        {/* Insights */}
        {insights.length > 0 && (
          <div className="flex flex-col gap-3">
            <h2 className="text-xs font-semibold uppercase tracking-wide"
              style={{ color: "var(--color-on-surface-variant)" }}>
              Insights do Sistema
            </h2>
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
              {insights.slice(0,3).map((insight) => {
                const cfg = {
                  success: { bg:"var(--color-primary-fixed)", border:"var(--color-primary-fixed-dim)",
                    iconBg:"var(--color-secondary)", lc:"var(--color-on-primary-fixed-variant)",
                    icon:<Zap size={15} color="#fff" /> },
                  warning: { bg:"#fffbeb", border:"#fde68a", iconBg:"#f59e0b", lc:"#92400e",
                    icon:<AlertCircle size={15} color="#fff" /> },
                  danger:  { bg:"var(--color-error-container)", border:"#fca5a5",
                    iconBg:"var(--color-error)", lc:"var(--color-on-error-container)",
                    icon:<TrendingDown size={15} color="#fff" /> },
                }[insight.icone] ?? { bg:"var(--color-primary-fixed)", border:"var(--color-primary-fixed-dim)",
                  iconBg:"var(--color-secondary)", lc:"var(--color-on-primary-fixed-variant)",
                  icon:<Zap size={15} color="#fff" /> }

                return (
                  <div key={insight.tipo} className="rounded-xl border p-5 flex flex-col gap-3"
                    style={{ backgroundColor: cfg.bg, borderColor: cfg.border }}>
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-lg flex items-center justify-center"
                        style={{ backgroundColor: cfg.iconBg }}>{cfg.icon}</div>
                      <span className="text-xs font-semibold uppercase tracking-wide"
                        style={{ color: cfg.lc }}>{insight.titulo}</span>
                    </div>
                    <p className="text-sm leading-relaxed flex-1"
                      style={{ color: "var(--color-on-surface)" }}>{insight.mensagem}</p>
                    {insight.acao && insight.href && (
                      <a href={insight.href}
                        className="flex items-center gap-1 text-xs font-semibold mt-auto"
                        style={{ color: cfg.lc }}>
                        {insight.acao} <ChevronRight size={13} />
                      </a>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* === LINHA 1 DE GRÁFICOS === */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">

          {/* Faturamento comparativo — 2/3 */}
          <div className="lg:col-span-2 rounded-xl p-6 border"
            style={{ backgroundColor:"var(--color-surface-container-lowest)",
              borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>

            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wide"
                  style={{ color:"var(--color-on-surface-variant)" }}>
                  Faturamento com comparativo
                </span>
                <span title="Compara o período atual com o mesmo período anterior."
                  className="inline-flex items-center justify-center w-4 h-4 rounded-full cursor-help"
                  style={{ backgroundColor:"var(--color-surface-container)", color:"var(--color-on-surface-variant)" }}>
                  <HelpCircle size={11} />
                </span>
              </div>

              <div className="relative flex rounded-lg border overflow-hidden w-[240px] flex-shrink-0"
                style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
                <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                  style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["week", "month", "year"].indexOf(periodo) * 100}%)` }} />
                {(["week","month","year"] as PeriodoComparativo[]).map((p) => (
                  <button key={p} onClick={() => setPeriodo(p)}
                    className="relative z-10 flex-1 py-1.5 text-xs font-semibold transition-colors duration-300 text-center outline-none"
                    style={{ color: periodo === p ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                    {{ week:"Semana", month:"Mês", year:"Ano" }[p]}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex items-center gap-5 mb-4">
              {[
                { color:"#94a3b8", label: periodoConfig.previousLabel },
                { color:"#3b82f6", label: periodoConfig.currentLabel  },
              ].map(l => (
                <div key={l.label} className="flex items-center gap-2">
                  <span className="w-3 h-3 rounded-full" style={{ backgroundColor: l.color }} />
                  <span className="text-xs font-medium" style={{ color:"var(--color-on-surface-variant)" }}>
                    {l.label}
                  </span>
                </div>
              ))}
            </div>

            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={comparativo} margin={{ top:4, right:4, left:0, bottom:0 }}>
                <defs>
                  <linearGradient id="gAtual" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#3b82f6" stopOpacity={0.20} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.02} />
                  </linearGradient>
                  <linearGradient id="gAnterior" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#94a3b8" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#94a3b8" stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#eff4ff" />
                <XAxis dataKey="label" tick={{ fontSize:11, fill:"#45464d" }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize:11, fill:"#45464d" }} tickFormatter={v => `R$${(v/1000).toFixed(0)}k`} axisLine={false} tickLine={false} />
                <Tooltip content={<CustomTooltip config={periodoConfig} />} />
                <Area type="monotone" dataKey="anterior" stroke="#94a3b8" strokeWidth={2} fill="url(#gAnterior)" strokeDasharray="5 3"
                  dot={(p: ChartPointProps) => <Dot key={p.index} cx={p.cx} cy={p.cy} stroke="#94a3b8" />} activeDot={{ r:4 }} />
                <Area type="monotone" dataKey="atual" stroke="#3b82f6" strokeWidth={2.5} fill="url(#gAtual)"
                  dot={(p: ChartPointProps) => <Dot key={p.index} cx={p.cx} cy={p.cy} stroke="#3b82f6" />} activeDot={{ r:5 }} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Top Dias — 1/3 */}
          <div className="rounded-xl p-6 border flex flex-col gap-4"
            style={{ backgroundColor:"var(--color-surface-container-lowest)",
              borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wide"
                  style={{ color:"var(--color-on-surface-variant)" }}>
                  Top Dias
                </span>
                <span title="Exibe os dias com maior faturamento no período selecionado."
                  className="inline-flex items-center justify-center w-4 h-4 rounded-full cursor-help"
                  style={{ backgroundColor:"var(--color-surface-container)", color:"var(--color-on-surface-variant)" }}>
                  <HelpCircle size={11} />
                </span>
              </div>
              
              <div className="relative flex rounded-lg border overflow-hidden w-[160px] flex-shrink-0"
                style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
                <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                  style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["week", "month", "year"].indexOf(periodoTopDias) * 100}%)` }} />
                {(["week","month","year"] as PeriodoComparativo[]).map((p) => (
                  <button key={p} onClick={() => setPeriodoTopDias(p)}
                    className="relative z-10 flex-1 py-1 text-[10px] uppercase font-semibold transition-colors duration-300 text-center outline-none"
                    style={{ color: periodoTopDias === p ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                    {{ week:"Sem", month:"Mês", year:"Ano" }[p]}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-3 flex-1 mt-2">
              {topDias.length === 0 ? (
                 <span className="text-xs text-center mt-4" style={{ color:"var(--color-on-surface-variant)" }}>Nenhum dado no período.</span>
              ) : (
                topDias.map((v, i) => {
                  const max = Math.max(...topDias.map(x => x.faturamento))
                  const pct = max > 0 ? (v.faturamento / max) * 100 : 0
                  return (
                    <div key={v.dia} className="flex flex-col gap-1">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium" style={{ color:"var(--color-on-surface)" }}>
                          {new Date(v.dia+"T00:00:00").toLocaleDateString("pt-BR",{day:"2-digit",month:"short"})}
                        </span>
                        <span className="text-xs font-semibold" style={{ color:"var(--color-on-surface)" }}>
                          {fmt(v.faturamento)}
                        </span>
                      </div>
                      <div className="h-1.5 rounded-full overflow-hidden" style={{ backgroundColor:"var(--color-surface-container)" }}>
                        <div className="h-full rounded-full transition-all"
                          style={{ width: `${pct}%`, backgroundColor: i === 0 ? "#3b82f6" : "var(--color-primary-fixed-dim)" }} />
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </div>
        </div>

        {/* === LINHA 2 DE GRÁFICOS === */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          
          {/* Mix de Vendas — 1/3 */}
          <div className="rounded-xl p-6 border flex flex-col gap-4"
            style={{ backgroundColor:"var(--color-surface-container-lowest)",
              borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wide"
                  style={{ color:"var(--color-on-surface-variant)" }}>
                  Mix de Vendas
                </span>
                <span title="Distribuição do faturamento por categoria de produto."
                  className="inline-flex items-center justify-center w-4 h-4 rounded-full cursor-help"
                  style={{ backgroundColor:"var(--color-surface-container)", color:"var(--color-on-surface-variant)" }}>
                  <HelpCircle size={11} />
                </span>
              </div>

              <div className="relative flex rounded-lg border overflow-hidden w-[160px] flex-shrink-0"
                style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
                <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                  style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["week", "month", "year"].indexOf(periodoMix) * 100}%)` }} />
                {(["week","month","year"] as PeriodoComparativo[]).map((p) => (
                  <button key={p} onClick={() => setPeriodoMix(p)}
                    className="relative z-10 flex-1 py-1 text-[10px] uppercase font-semibold transition-colors duration-300 text-center outline-none"
                    style={{ color: periodoMix === p ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                    {{ week:"Sem", month:"Mês", year:"Ano" }[p]}
                  </button>
                ))}
              </div>
            </div>

            <div className="relative h-[150px] w-full flex justify-center items-center">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={vendasCategoria} cx="50%" cy="50%" innerRadius={50} outerRadius={70} paddingAngle={3} dataKey="faturamento" stroke="none">
                    {vendasCategoria.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.fill} />)}
                  </Pie>
                  <Tooltip 
                    content={({ active, payload }) => {
                      if (!active || !payload?.length) return null;
                      const data = payload[0].payload;
                      return (
                        <div className="rounded-lg px-3 py-2 flex flex-col gap-1"
                          style={{ backgroundColor: "#213145", color: "#eaf1ff", fontSize: "12px" }}>
                          <span className="opacity-80">{data.categoria}</span>
                          <span className="font-semibold">{fmt(data.faturamento)}</span>
                        </div>
                      );
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
              
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="text-[10px] uppercase font-semibold opacity-50" style={{ color:"var(--color-on-surface)" }}>Total</span>
                <span className="text-sm font-bold" style={{ color:"var(--color-on-surface)" }}>
                  {(() => {
                    const total = vendasCategoria.reduce((a,b)=>a+b.faturamento,0);
                    return `R$${(total/1000).toFixed(0)}k`
                  })()}
                </span>
              </div>
            </div>

            <div className="flex flex-col gap-2 flex-1 justify-end mt-2">
              {vendasCategoria.map(cat => {
                const total = vendasCategoria.reduce((a,b)=>a+b.faturamento,0);
                const pct = total > 0 ? ((cat.faturamento / total) * 100).toFixed(1).replace('.', ',') : "0,0";
                
                return (
                  <div key={cat.categoria} className="flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: cat.fill }} />
                      <span style={{ color:"var(--color-on-surface-variant)", fontWeight: 500 }}>{cat.categoria}</span>
                    </div>
                    <span className="font-semibold" style={{ color:"var(--color-on-surface)" }}>{pct}%</span>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Pico de Atendimento — 2/3 */}
          <div className="lg:col-span-2 rounded-xl p-6 border flex flex-col gap-4"
            style={{ backgroundColor:"var(--color-surface-container-lowest)",
              borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wide"
                  style={{ color:"var(--color-on-surface-variant)" }}>
                  Pico de Atendimento (Vendas por Hora)
                </span>
                <span title="Mostra os horários de maior movimento na loja."
                  className="inline-flex items-center justify-center w-4 h-4 rounded-full cursor-help"
                  style={{ backgroundColor:"var(--color-surface-container)", color:"var(--color-on-surface-variant)" }}>
                  <HelpCircle size={11} />
                </span>
              </div>

              <div className="relative flex rounded-lg border overflow-hidden w-[240px] flex-shrink-0"
                style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
                <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                  style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["week", "month", "year"].indexOf(periodoPico) * 100}%)` }} />
                {(["week","month","year"] as PeriodoComparativo[]).map((p) => (
                  <button key={p} onClick={() => setPeriodoPico(p)}
                    className="relative z-10 flex-1 py-1.5 text-xs font-semibold transition-colors duration-300 text-center outline-none"
                    style={{ color: periodoPico === p ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                    {{ week:"Semana", month:"Mês", year:"Ano" }[p]}
                  </button>
                ))}
              </div>
            </div>
            
            <div className="flex-1 w-full min-h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={vendasHora} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#eff4ff" vertical={false} />
                  <XAxis dataKey="hora" tick={{ fontSize:11, fill:"#45464d" }} axisLine={false} tickLine={false} />
                  
                  <YAxis 
                    tick={{ fontSize:11, fill:"#45464d" }} 
                    tickFormatter={v => `R$${(v/1000).toFixed(1).replace('.', ',').replace(',0', '')}k`} 
                    axisLine={false} 
                    tickLine={false} 
                    domain={[0, 'dataMax']} 
                    allowDataOverflow={true} 
                  />
                  
                  <Tooltip
                    cursor={{ fill: 'var(--color-surface-container-low)' }}
                    content={({ active, payload }) => {
                      if (!active || !payload?.length) return null;
                      return (
                        <div className="rounded-lg px-4 py-3 flex flex-col gap-1"
                          style={{ backgroundColor: "#213145", color: "#eaf1ff", fontSize: "13px" }}>
                          <p className="font-semibold text-xs opacity-60">{payload[0].payload.hora}</p>
                          <span className="font-semibold">{fmt(payload[0].value as number)}</span>
                        </div>
                      );
                    }}
                  />
                  
                  <Bar dataKey="faturamento" radius={[4, 4, 0, 0]} maxBarSize={60}>
                    {vendasHora.map((entry, index) => {
                      const isPeak = entry.faturamento === Math.max(...vendasHora.map(v => v.faturamento));
                      return <Cell key={`cell-${index}`} fill={isPeak ? "#3b82f6" : "#94a3b8"} opacity={isPeak ? 1 : 0.4} />
                    })}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

        </div>
      </div>
    </>
  )
}