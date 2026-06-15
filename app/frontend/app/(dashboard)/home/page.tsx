"use client"

import { useEffect, useState } from "react"
import Header from "@/components/layout/Header"
import { TrendingUp, ShoppingCart, Tag, AlertTriangle, Loader2 } from "lucide-react"
import api, { Insight } from "@/lib/api"
import { Zap, AlertCircle, TrendingDown, ChevronRight } from "lucide-react"
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, BarChart, Bar
} from "recharts"
import { HomeKPIs, VendaDia, EstoqueItem } from "@/types"

function fmt(value: number) {
  return new Intl.NumberFormat("pt-BR", {
    style: "currency", currency: "BRL", maximumFractionDigits: 0
  }).format(value)
}

interface KpiCardProps {
  label:      string
  value:      string
  icon:       React.ReactNode
  accent?:    string
  iconColor?: string
}
function KpiCard({ label, value, icon, accent, iconColor }: KpiCardProps & { iconColor?: string }) {
  return (
    <div
      className="rounded-xl p-5 flex flex-col gap-4 border"
      style={{
        backgroundColor: "var(--color-surface-container-lowest)",
        borderColor:     "var(--color-outline-variant)",
        boxShadow:       "0px 1px 4px rgba(15, 23, 42, 0.06)",
      }}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-wide"
          style={{ color: "var(--color-on-surface-variant)" }}>
          {label}
        </span>
        <div
          className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
          style={{ backgroundColor: accent ?? "var(--color-surface-container-low)" }}
        >
          {icon}
        </div>
      </div>
      <p
        className="text-2xl font-bold tracking-tight"
        style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}
      >
        {value}
      </p>
    </div>
  )
}

export default function HomePage() {
  const [kpis,    setKpis]    = useState<HomeKPIs | null>(null)
  const [vendas,  setVendas]  = useState<VendaDia[]>([])
  const [alertas, setAlertas] = useState<EstoqueItem[]>([])
  const [insights, setInsights] = useState<Insight[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      try {
        const [kpisRes, vendasRes, alertasRes, insightsRes] = await Promise.all([
          api.get("/home/kpis"),
          api.get("/vendas/por-dia"),
          api.get("/estoque/alertas"),
          api.get("/insights"),
        ])
        setKpis(kpisRes.data)
        setVendas(vendasRes.data ?? [])
        setAlertas(alertasRes.data ?? [])
        setInsights(insightsRes.data ?? [])
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

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

        {/* KPI Cards */}
        {kpis && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard
              label="Faturamento Total"
              value={fmt(kpis.faturamento)}
              icon={<TrendingUp size={16} color="var(--color-secondary)" />}
              accent="var(--color-primary-fixed)"
            />
            <KpiCard
              label="Ticket Médio"
              value={fmt(kpis.ticket_medio)}
              icon={<Tag size={16} color="var(--color-secondary)" />}
              accent="var(--color-primary-fixed)"
            />
            <KpiCard
              label="Total de Vendas"
              value={String(kpis.total_vendas)}
              icon={<ShoppingCart size={16} color="var(--color-on-surface-variant)" />}
              accent="var(--color-surface-container)"
            />
            <KpiCard
              label="Total de Descontos"
              value={fmt(kpis.total_desconto)}
              icon={<AlertTriangle size={16} color="var(--color-error)" />}
              accent="var(--color-error-container)"
            />
          </div>
        )}
        {/* Insights dinamicos */}
        {insights.length > 0 && (
          <div className="flex flex-col gap-3">
            <h2
              className="text-xs font-semibold uppercase tracking-wide"
              style={{ color: "var(--color-on-surface-variant)" }}
            >
              Insights do Sistema
            </h2>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
              {insights.slice(0, 3).map((insight) => {
                const config = {
                  success: {
                    bg: "var(--color-primary-fixed)",
                    border: "var(--color-primary-fixed-dim)",
                    iconBg: "var(--color-secondary)",
                    labelColor: "var(--color-on-primary-fixed-variant)",
                    icon: <Zap size={15} color="#ffffff" />,
                  },
                  warning: {
                    bg: "#fffbeb",
                    border: "#fde68a",
                    iconBg: "#f59e0b",
                    labelColor: "#92400e",
                    icon: <AlertCircle size={15} color="#ffffff" />,
                  },
                  danger: {
                    bg: "var(--color-error-container)",
                    border: "#fca5a5",
                    iconBg: "var(--color-error)",
                    labelColor: "var(--color-on-error-container)",
                    icon: <TrendingDown size={15} color="#ffffff" />,
                  },
                }

                const c = config[insight.icone] ?? config.success

                return (
                  <div
                    key={insight.tipo}
                    className="rounded-xl border p-5 flex flex-col gap-3"
                    style={{ backgroundColor: c.bg, borderColor: c.border }}
                  >
                    <div className="flex items-center gap-2">
                      <div
                        className="w-7 h-7 rounded-lg flex items-center justify-center"
                        style={{ backgroundColor: c.iconBg }}
                      >
                        {c.icon}
                      </div>
                      <span
                        className="text-xs font-semibold uppercase tracking-wide"
                        style={{ color: c.labelColor }}
                      >
                        {insight.titulo}
                      </span>
                    </div>

                    <p
                      className="text-sm leading-relaxed flex-1"
                      style={{ color: "var(--color-on-surface)" }}
                    >
                      {insight.mensagem}
                    </p>

                    {insight.acao && insight.href && (
                      <a
                        href={insight.href}
                        className="flex items-center gap-1 text-xs font-semibold mt-auto"
                        style={{ color: c.labelColor }}
                      >
                        {insight.acao} <ChevronRight size={13} />
                      </a>
                    )}
                  </div>
                )
              })}
            </div>

          </div>
        )}
        
        {/* Gráficos */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">

          {/* Faturamento por dia — ocupa 2/3 */}
          <div
            className="lg:col-span-2 rounded-xl p-6 border"
            style={{
              backgroundColor: "var(--color-surface-container-lowest)",
              borderColor:     "var(--color-outline-variant)",
              boxShadow:       "var(--shadow-sm)",
            }}
          >
            <h2
              className="text-base font-semibold mb-4"
              style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}
            >
              Faturamento por Dia
            </h2>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={vendas} margin={{ top: 4, right: 4, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="gradFat" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%"  stopColor="#0051d5" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#0051d5" stopOpacity={0}    />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#eff4ff" />
                <XAxis
                  dataKey="dia"
                  tick={{ fontSize: 11, fill: "#45464d" }}
                  tickFormatter={(v) => v.slice(5)}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "#45464d" }}
                  tickFormatter={(v) => `R$${(v/1000).toFixed(0)}k`}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip
                  formatter={(v: number) => [fmt(v), "Faturamento"]}
                  contentStyle={{
                    backgroundColor: "#213145",
                    border:          "none",
                    borderRadius:    "8px",
                    color:           "#eaf1ff",
                    fontSize:        "13px",
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="faturamento"
                  stroke="#0051d5"
                  strokeWidth={2}
                  fill="url(#gradFat)"
                  dot={{ fill: "#ffffff", stroke: "#0051d5", strokeWidth: 2, r: 3 }}
                  activeDot={{ r: 5 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Vendas por dia — ocupa 1/3 */}
          <div
            className="rounded-xl p-6 border"
            style={{
              backgroundColor: "var(--color-surface-container-lowest)",
              borderColor:     "var(--color-outline-variant)",
              boxShadow:       "var(--shadow-sm)",
            }}
          >
            <h2
              className="text-base font-semibold mb-4"
              style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}
            >
              Vendas por Dia
            </h2>
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={vendas} margin={{ top: 4, right: 4, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#eff4ff" />
                <XAxis
                  dataKey="dia"
                  tick={{ fontSize: 11, fill: "#45464d" }}
                  tickFormatter={(v) => v.slice(5)}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: "#45464d" }}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "#213145",
                    border:          "none",
                    borderRadius:    "8px",
                    color:           "#eaf1ff",
                    fontSize:        "13px",
                  }}
                />
                <Bar
                  dataKey="vendas"
                  fill="#213145"
                  radius={[4, 4, 0, 0]}
                  maxBarSize={32}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>

        </div>

        {/* Resumo por dia */}
        <div
          className="rounded-xl border overflow-hidden"
          style={{
            backgroundColor: "var(--color-surface-container-lowest)",
            borderColor:     "var(--color-outline-variant)",
            boxShadow:       "var(--shadow-sm)",
          }}
        >
          <div className="px-6 py-4 border-b" style={{ borderColor: "var(--color-outline-variant)" }}>
            <h2
              className="text-base font-semibold"
              style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}
            >
              Resumo por Dia
            </h2>
          </div>
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-outline-variant)" }}>
                <th className="text-left px-6 py-3 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-on-surface-variant)" }}>Data</th>
                <th className="text-right px-6 py-3 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-on-surface-variant)" }}>Vendas</th>
                <th className="text-right px-6 py-3 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-on-surface-variant)" }}>Faturamento</th>
                <th className="text-right px-6 py-3 text-xs font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-on-surface-variant)" }}>Ticket Médio</th>
              </tr>
            </thead>
            <tbody>
              {vendas.map((v, i) => (
                <tr
                  key={v.dia}
                  style={{
                    borderBottom: i < vendas.length - 1 ? "1px solid var(--color-surface-container)" : "none",
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = "var(--color-surface-container-low)"
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "transparent"
                  }}
                >
                  <td className="px-6 py-3 font-medium" style={{ color: "var(--color-on-surface)" }}>
                    {new Date(v.dia + "T00:00:00").toLocaleDateString("pt-BR")}
                  </td>
                  <td className="px-6 py-3 text-right" style={{ color: "var(--color-on-surface-variant)" }}>
                    {v.vendas}
                  </td>
                  <td className="px-6 py-3 text-right font-semibold" style={{ color: "var(--color-on-surface)" }}>
                    {fmt(v.faturamento)}
                  </td>
                  <td className="px-6 py-3 text-right" style={{ color: "var(--color-on-surface-variant)" }}>
                    {fmt(v.faturamento / v.vendas)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

      </div>
    </>
  )
}