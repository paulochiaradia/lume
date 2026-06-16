"use client"

import { useState, useEffect } from "react"
import Header from "@/components/layout/Header"
import api from "@/lib/api"
import {
  TrendingUp, TrendingDown, ShoppingCart, Tag, Percent, 
  HelpCircle, Zap, AlertCircle, ChevronRight, Users, CalendarDays, Loader2
} from "lucide-react"
import {
  ComposedChart, Area, Line, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Cell, PieChart, Pie
} from "recharts"

// ── Tipos ────────────────────────────────────────────────────
type PeriodoComparativo = "week" | "month" | "year"

// Nova tipagem baseada na resposta da nossa API em Go
type TrendValue = {
  atual: number
  anterior: number
  variacao: number
}

type VendasKPIsTrend = {
  faturamento: TrendValue
  ticket_medio: TrendValue
  total_transacoes: TrendValue
  perc_desconto: TrendValue
}

// ── Mock Data (Manteremos para os blocos que ainda vamos integrar) ────────────────
const MOCK_INSIGHTS = [
  { tipo: "info", titulo: "Sazonalidade", mensagem: "Seus melhores dias de vendas são terça e quarta — considere reforçar o estoque da frente de loja nesses dias.", icone: "success" },
  { tipo: "warning", titulo: "Atenção (Vendedores)", mensagem: "O vendedor VEN02 tem um UPA (Itens por Venda) de 1.2, muito abaixo da média da equipe (3.4).", icone: "warning" },
  { tipo: "info", titulo: "Tendência de Categoria", mensagem: "A categoria 'Ferramentas Manuais' cresceu 18% neste período em relação ao anterior.", icone: "success" }
]

const MOCK_DAILY = [
  { dia: "01/12", faturamento: 12000, mediaMovel: 11000 },
  { dia: "02/12", faturamento: 15000, mediaMovel: 11500 },
  { dia: "03/12", faturamento: 18000, mediaMovel: 12500 },
  { dia: "04/12", faturamento: 14000, mediaMovel: 13000 },
  { dia: "05/12", faturamento: 22000, mediaMovel: 14500 },
  { dia: "06/12", faturamento: 25000, mediaMovel: 16000 },
  { dia: "07/12", faturamento: 19000, mediaMovel: 17850 },
]

const MOCK_MIX = [
  { categoria: "Construção", faturamento: 45000, fill: "#3b82f6" },
  { categoria: "Acabamento", faturamento: 30000, fill: "#10b981" },
  { categoria: "Ferramentas", faturamento: 15000, fill: "#f59e0b" },
  { categoria: "Elétrica", faturamento: 10000, fill: "#8b5cf6" },
]

const MOCK_SELLERS = [
  { id: "VEN01", nome: "Carlos Silva", vendas: 45, faturamento: 35000, ticket: 777, upa: 3.5, desconto: 4.2 },
  { id: "VEN02", nome: "Ana Costa", vendas: 62, faturamento: 28000, ticket: 451, upa: 1.2, desconto: 8.5 },
  { id: "VEN03", nome: "Roberto Luiz", vendas: 38, faturamento: 42000, ticket: 1105, upa: 4.1, desconto: 2.1 },
]

const DIAS_SEMANA = ["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"]
const HORARIOS = ["08h", "09h", "10h", "11h", "12h", "13h", "14h", "15h", "16h", "17h", "18h"]

const INTENSIDADES = [
  [10, 15, 20, 25, 30, 45], [20, 35, 45, 55, 60, 80], [30, 60, 75, 85, 90, 95], 
  [40, 70, 80, 90, 95, 85], [25, 40, 50, 45, 60, 70], [35, 50, 60, 55, 75, 80], 
  [50, 75, 85, 80, 90, 60], [65, 85, 95, 90, 100, 50], [55, 70, 80, 75, 85, 40], 
  [45, 55, 65, 60, 70, 30], [20, 30, 40, 35, 45, 15], 
]

const MOCK_HEATMAP = HORARIOS.map((h, hIdx) => 
  DIAS_SEMANA.map((d, dIdx) => ({ hora: h, dia: d, intensidade: INTENSIDADES[hIdx][dIdx] }))
)

// ── Helpers de Formatação ────────────────────────────────────
const fmt = (v: number) => new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(v)
const fmtPct = (v: number) => `${v > 0 ? '+' : ''}${v.toFixed(1).replace('.', ',')}%`

// ── Componente: KPI Card com Tendência ───────────────────────
function KpiTrendCard({ label, value, trend, inverseTrend = false, icon, accent }: {
  label: string; value: string; trend: number; inverseTrend?: boolean; icon: React.ReactNode; accent?: string
}) {
  const isPositive = trend > 0;
  const isGood = inverseTrend ? !isPositive : isPositive;
  
  const trendColor = isGood ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400";
  const trendBg = isGood ? "bg-emerald-100 dark:bg-emerald-500/10" : "bg-rose-100 dark:bg-rose-500/10";
  const TrendIcon = isPositive ? TrendingUp : TrendingDown;

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
      <div className="flex items-end justify-between">
        <p className="text-2xl font-bold tracking-tight" style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}>
          {value}
        </p>
        <div className={`flex items-center gap-1 px-2 py-1 rounded-md text-[11px] font-bold ${trendBg} ${trendColor}`}>
          <TrendIcon size={12} strokeWidth={3} />
          {fmtPct(trend)}
        </div>
      </div>
    </div>
  )
}

// ── Página Principal ─────────────────────────────────────────
export default function VendasPage() {
  const [periodo, setPeriodo] = useState<PeriodoComparativo>("month")
  
  // Estado real dos KPIs vindo da API
  const [kpis, setKpis] = useState<VendasKPIsTrend | null>(null)
  const [loading, setLoading] = useState(true)

  // Dispara a busca toda vez que o botão do filtro (Semana/Mês/Ano) for clicado
  useEffect(() => {
    async function fetchKpis() {
      setLoading(true)
      try {
        const response = await api.get(`/vendas/kpis?periodo=${periodo}`)
        setKpis(response.data)
      } catch (err) {
        console.error("Erro ao buscar KPIs de Vendas:", err)
      } finally {
        setLoading(false)
      }
    }
    fetchKpis()
  }, [periodo])

  return (
    <>
      <Header title="Análise de Vendas" />
      <div className="flex flex-col gap-6">

        {/* ── BLOCO 1: KPIs com Tendência ────────────────────────── */}
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>
              Métricas Principais (vs Período Anterior)
            </h2>
            <div className="relative flex rounded-lg border overflow-hidden w-[240px] flex-shrink-0"
              style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
              <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["week", "month", "year"].indexOf(periodo) * 100}%)` }} />
              {(["week", "month", "year"] as PeriodoComparativo[]).map((p) => (
                <button key={p} onClick={() => setPeriodo(p)}
                  className="relative z-10 flex-1 py-1.5 text-xs font-semibold transition-colors duration-300 text-center outline-none"
                  style={{ color: periodo === p ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                  {{ week: "Semana", month: "Mês", year: "Ano" }[p]}
                </button>
              ))}
            </div>
          </div>

          {/* Renderiza um Loader ou os Cards Reais */}
          {loading || !kpis ? (
            <div className="h-[120px] rounded-xl flex items-center justify-center border" style={{ backgroundColor: "var(--color-surface-container-lowest)", borderColor: "var(--color-outline-variant)" }}>
               <Loader2 size={24} className="animate-spin text-blue-500" />
            </div>
          ) : (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <KpiTrendCard 
                label="Faturamento" 
                value={fmt(kpis.faturamento.atual)} 
                trend={kpis.faturamento.variacao} 
                icon={<TrendingUp size={16} color="var(--color-secondary)" />} 
                accent="var(--color-primary-fixed)" 
              />
              <KpiTrendCard 
                label="Ticket Médio" 
                value={fmt(kpis.ticket_medio.atual)} 
                trend={kpis.ticket_medio.variacao} 
                icon={<Tag size={16} color="var(--color-secondary)" />} 
                accent="var(--color-primary-fixed)" 
              />
              <KpiTrendCard 
                label="Total de Vendas" 
                value={String(kpis.total_transacoes.atual)} 
                trend={kpis.total_transacoes.variacao} 
                icon={<ShoppingCart size={16} color="var(--color-on-surface-variant)" />} 
                accent="var(--color-surface-container)" 
              />
              <KpiTrendCard 
                label="% Desconto Médio" 
                value={`${kpis.perc_desconto.atual.toFixed(1).replace('.', ',')}%`} 
                trend={kpis.perc_desconto.variacao} 
                inverseTrend={true} 
                icon={<Percent size={16} color="var(--color-on-surface-variant)" />} 
                accent="var(--color-surface-container)" 
              />
            </div>
          )}
        </div>

        {/* ── BLOCO 6: Insights ──────────────────────────────────── */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {MOCK_INSIGHTS.map((insight, idx) => {
            const cfg = {
              success: { bg:"var(--color-primary-fixed)", border:"var(--color-primary-fixed-dim)", iconBg:"var(--color-secondary)", lc:"var(--color-on-primary-fixed-variant)", icon:<Zap size={15} color="#fff" /> },
              warning: { bg:"#fffbeb", border:"#fde68a", iconBg:"#f59e0b", lc:"#92400e", icon:<AlertCircle size={15} color="#fff" /> },
            }[insight.icone] || { bg:"var(--color-surface-container)", border:"var(--color-outline)", iconBg:"var(--color-on-surface-variant)", lc:"var(--color-on-surface)", icon:<Zap size={15} color="#fff" /> }

            return (
              <div key={idx} className="rounded-xl border p-5 flex flex-col gap-3" style={{ backgroundColor: cfg.bg, borderColor: cfg.border }}>
                <div className="flex items-center gap-2">
                  <div className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ backgroundColor: cfg.iconBg }}>{cfg.icon}</div>
                  <span className="text-xs font-semibold uppercase tracking-wide" style={{ color: cfg.lc }}>{insight.titulo}</span>
                </div>
                <p className="text-sm leading-relaxed flex-1" style={{ color: "var(--color-on-surface)" }}>{insight.mensagem}</p>
              </div>
            )
          })}
        </div>

        {/* ── BLOCO 2 e 5: Linha Diária e Mix ────────────────────── */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          
          <div className="lg:col-span-2 rounded-xl p-6 border flex flex-col gap-4" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                Faturamento e Média Móvel (7d)
              </span>
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-blue-500"/><span className="text-[10px] uppercase font-semibold text-slate-500">Diário</span></div>
                <div className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-amber-500"/><span className="text-[10px] uppercase font-semibold text-slate-500">Média 7d</span></div>
              </div>
            </div>
            
            <div className="flex-1 min-h-[220px]">
              <ResponsiveContainer width="100%" height="100%">
                <ComposedChart data={MOCK_DAILY} margin={{ top: 5, right: 0, left: 0, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorFaturamento" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2}/>
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--color-surface-container-high)" />
                  <XAxis dataKey="dia" axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: "var(--color-on-surface-variant)" }} />
                  <YAxis axisLine={false} tickLine={false} tickFormatter={(v) => `R$${v/1000}k`} tick={{ fontSize: 11, fill: "var(--color-on-surface-variant)" }} />
                  <Tooltip />
                  <Area type="monotone" dataKey="faturamento" fillOpacity={1} fill="url(#colorFaturamento)" stroke="#3b82f6" strokeWidth={2} />
                  <Line type="monotone" dataKey="mediaMovel" stroke="#f59e0b" strokeWidth={3} dot={false} activeDot={{ r: 6 }} />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          </div>

          <div className="rounded-xl p-6 border flex flex-col gap-4" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
            <span className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
              Mix de Vendas (Categoria)
            </span>
            <div className="flex-1 relative flex justify-center items-center min-h-[160px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={MOCK_MIX} cx="50%" cy="50%" innerRadius={55} outerRadius={75} paddingAngle={4} dataKey="faturamento" stroke="none">
                    {MOCK_MIX.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.fill} />)}
                  </Pie>
                  <Tooltip formatter={(value: number) => fmt(value)} />
                </PieChart>
              </ResponsiveContainer>
              <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                <span className="text-xl font-bold" style={{ color:"var(--color-on-surface)" }}>100K</span>
              </div>
            </div>
            <div className="flex flex-col gap-2">
              {MOCK_MIX.map(cat => (
                <div key={cat.categoria} className="flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: cat.fill }} />
                    <span style={{ color:"var(--color-on-surface-variant)", fontWeight: 500 }}>{cat.categoria}</span>
                  </div>
                  <span className="font-semibold" style={{ color:"var(--color-on-surface)" }}>{fmt(cat.faturamento)}</span>
                </div>
              ))}
            </div>
          </div>

        </div>

        {/* ── BLOCO 3: Performance por Vendedor (Tabela) ─────────── */}
        <div className="rounded-xl border overflow-hidden" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
            <div className="flex items-center gap-2">
              <Users size={16} color="var(--color-on-surface-variant)" />
              <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>Ranking de Vendedores</h3>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)" }}>
                <tr>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Vendedor</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Qtd. Vendas</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Faturamento</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Ticket Médio</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">UPA</th>
                  <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">% Desconto</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--color-outline-variant)]">
                {MOCK_SELLERS.map((seller, idx) => (
                  <tr key={seller.id} className="transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20">
                    <td className="px-5 py-3">
                      <div className="flex items-center gap-3">
                        <span className="flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold" 
                              style={{ backgroundColor: idx === 0 ? "#fef08a" : "var(--color-surface-container-high)", color: idx === 0 ? "#854d0e" : "inherit" }}>
                          {idx + 1}
                        </span>
                        <div className="flex flex-col">
                          <span className="font-semibold" style={{ color: "var(--color-on-surface)" }}>{seller.nome}</span>
                          <span className="text-[10px]" style={{ color: "var(--color-on-surface-variant)" }}>{seller.id}</span>
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-3 text-right font-medium">{seller.vendas}</td>
                    <td className="px-5 py-3 text-right font-semibold text-blue-600 dark:text-blue-400">{fmt(seller.faturamento)}</td>
                    <td className="px-5 py-3 text-right">{fmt(seller.ticket)}</td>
                    <td className="px-5 py-3 text-right">
                      <span className={`px-2 py-1 rounded-md text-xs font-medium ${seller.upa < 2 ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                        {seller.upa.toFixed(1)}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-right">{seller.desconto.toFixed(1).replace('.', ',')}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* ── BLOCO 4: Heatmap (Hora x Dia) ──────────────────────── */}
        <div className="rounded-xl p-6 border flex flex-col gap-5" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <CalendarDays size={16} color="var(--color-on-surface-variant)" />
              <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>
                Concentração de Vendas (Heatmap Comercial)
              </h3>
            </div>
            <div className="flex items-center gap-2 text-[10px] text-slate-500 uppercase font-semibold">
              <span>Menos</span>
              <div className="flex gap-1">
                {[10, 30, 60, 90].map(op => <div key={op} className="w-3 h-3 rounded-sm bg-blue-500" style={{ opacity: op/100 }}/>)}
              </div>
              <span>Mais</span>
            </div>
          </div>

          <div className="flex flex-col gap-1 overflow-x-auto pb-2">
            <div className="flex items-center">
              <div className="w-12 flex-shrink-0" />
              {DIAS_SEMANA.map(dia => (
                <div key={dia} className="flex-1 text-center text-xs font-semibold py-2" style={{ color:"var(--color-on-surface-variant)" }}>
                  {dia}
                </div>
              ))}
            </div>
            
            {MOCK_HEATMAP.map((row, idx) => (
              <div key={idx} className="flex items-center gap-1 min-w-[500px]">
                <div className="w-12 flex-shrink-0 text-[10px] font-semibold text-right pr-3" style={{ color:"var(--color-on-surface-variant)" }}>
                  {row[0].hora}
                </div>
                {row.map((cell, cIdx) => (
                  <div key={cIdx} 
                       title={`${cell.dia} às ${cell.hora}: Intensidade ${cell.intensidade}%`}
                       className="flex-1 h-8 rounded-md transition-all hover:scale-105 cursor-crosshair" 
                       style={{ 
                         backgroundColor: "#3b82f6", 
                         opacity: Math.max(0.05, cell.intensidade / 100)
                       }} 
                  />
                ))}
              </div>
            ))}
          </div>
        </div>

      </div>
    </>
  )
}