"use client"

import { useState, useEffect } from "react"
import Header from "@/components/layout/Header"
import { 
  Package, DollarSign, AlertTriangle, Activity, 
  Loader2, AlertCircle, ShoppingCart, Archive 
} from "lucide-react"

// ── Tipos ────────────────────────────────────────────────────
type AbaAtiva = "inteligencia" | "deadstock" | "geral"

interface EstoqueKPIs {
  total_skus: number
  valor_total_estoque: number
  itens_em_alerta: number
  taxa_ruptura: number
}

interface EstoqueAlerta {
  produto_key: string
  nome: string
  quantidade: number
  quantidade_min: number
}

interface EstoqueReposicao {
  id: string
  nome: string
  categoria: string
  classe_abc: string
  estoque_atual: number
  demanda_prevista: number
  dias_ate_ruptura: number
  quantidade_sugerida: number
  urgencia: number
}

interface DeadStock {
  Nome: string
  Quantidade: number
  PrecoCusto: number
  CapitalParado: number
}

interface EstoqueCompleto {
  produto_key: string
  nome: string
  categoria: string
  quantidade: number
  quantidade_min: number
  preco_venda: number
  preco_custo: number
  alerta: boolean
}

// ── Helpers de Formatação ────────────────────────────────────
const fmt = (v: number) => new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" }).format(v)
const fmtCompact = (v: number) => new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 }).format(v)

// ── Componente: KPI Card (Estilo Vendas) ─────────────────────
function KpiCard({ label, value, icon, accent, highlightClass = "" }: {
  label: string; value: string | number; icon: React.ReactNode; accent?: string; highlightClass?: string;
}) {
  return (
    <div className={`rounded-xl p-5 flex flex-col gap-4 border ${highlightClass}`}
      style={{ 
        backgroundColor: highlightClass ? undefined : "var(--color-surface-container-lowest)", 
        borderColor: highlightClass ? undefined : "var(--color-outline-variant)", 
        boxShadow: "0px 1px 4px rgba(15,23,42,0.06)" 
      }}>
      <div className="flex items-center justify-between">
        <span className={`text-xs font-semibold uppercase tracking-wide ${highlightClass ? "" : "text-[var(--color-on-surface-variant)]"}`}>
          {label}
        </span>
        <div className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" 
          style={{ backgroundColor: accent ?? "var(--color-surface-container-low)" }}>
          {icon}
        </div>
      </div>
      <div className="flex items-end justify-between">
        <p className={`text-2xl font-bold tracking-tight ${highlightClass ? "" : "text-[var(--color-on-surface)]"}`} 
           style={{ fontFamily: "var(--font-display)" }}>
          {value}
        </p>
      </div>
    </div>
  )
}

// ── Página Principal ─────────────────────────────────────────
export default function EstoquePage() {
  const [abaAtiva, setAbaAtiva] = useState<AbaAtiva>("inteligencia")
  
  const [kpis, setKpis] = useState<EstoqueKPIs | null>(null)
  const [alertas, setAlertas] = useState<EstoqueAlerta[]>([])
  const [reposicao, setReposicao] = useState<EstoqueReposicao[]>([])
  const [deadStock, setDeadStock] = useState<DeadStock[]>([])
  const [estoqueCompleto, setEstoqueCompleto] = useState<EstoqueCompleto[]>([])
  
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchEstoqueData() {
      setLoading(true)
      const token = localStorage.getItem('token')
      
      if (!token) {
        console.error("Atenção: Nenhum token encontrado!")
        setLoading(false)
        return
      }

      try {
        const headers = { 
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }

        const [resKpis, resAlertas, resReposicao, resDeadStock, resCompleto] = await Promise.all([
          fetch('http://localhost/api/v1/estoque/kpis', { headers }),
          fetch('http://localhost/api/v1/estoque/alertas', { headers }),
          fetch('http://localhost/api/v1/estoque/reposicao', { headers }),
          fetch('http://localhost/api/v1/produtos/dead-stock', { headers }),
          fetch('http://localhost/api/v1/estoque/completo', { headers })
        ])

        if (resKpis.ok) setKpis(await resKpis.json())
        if (resAlertas.ok) setAlertas(await resAlertas.json())
        if (resReposicao.ok) setReposicao(await resReposicao.json())
        if (resDeadStock.ok) setDeadStock(await resDeadStock.json())
        if (resCompleto.ok) setEstoqueCompleto(await resCompleto.json())

      } catch (err) {
        console.error("Erro ao buscar dados de Estoque:", err)
      } finally {
        setLoading(false)
      }
    }
    
    fetchEstoqueData()
  }, [])

  // Função auxiliar para tags de urgência
  const renderBadgeUrgencia = (urgencia: number, dias: number) => {
    switch (urgencia) {
      case 1: return <span className="px-2 py-1 text-[10px] font-bold text-red-700 bg-red-100 rounded-md">Crítico ({dias}d)</span>;
      case 2: return <span className="px-2 py-1 text-[10px] font-bold text-amber-700 bg-amber-100 rounded-md">Alerta ({dias}d)</span>;
      case 3: return <span className="px-2 py-1 text-[10px] font-bold text-yellow-700 bg-yellow-100 rounded-md">Atenção ({dias}d)</span>;
      default: return <span className="px-2 py-1 text-[10px] font-bold text-emerald-700 bg-emerald-100 rounded-md">Seguro</span>;
    }
  }

  return (
    <>
      <Header title="Gestão de Estoque" />
      <div className="flex flex-col gap-6 p-4 lg:p-6">

        {/* ── BLOCO 1: KPIs Principais ────────────────────────────── */}
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wide" style={{ color: "var(--color-on-surface-variant)" }}>
              Visão Geral do Imobilizado
            </h2>
            
            {/* Seletor de Abas Estilo Vendas */}
            <div className="relative flex rounded-lg border overflow-hidden w-[360px] flex-shrink-0"
              style={{ backgroundColor: "var(--color-surface-container-low)", borderColor: "var(--color-outline-variant)" }}>
              <div className="absolute top-0 bottom-0 w-1/3 transition-transform duration-300 ease-out"
                style={{ backgroundColor: "var(--color-inverse-surface)", transform: `translateX(${["inteligencia", "deadstock", "geral"].indexOf(abaAtiva) * 100}%)` }} />
              {(["inteligencia", "deadstock", "geral"] as AbaAtiva[]).map((tab) => (
                <button key={tab} onClick={() => setAbaAtiva(tab)}
                  className="relative z-10 flex-1 py-1.5 text-xs font-semibold transition-colors duration-300 text-center outline-none"
                  style={{ color: abaAtiva === tab ? "var(--color-inverse-on-surface)" : "var(--color-on-surface-variant)" }}>
                  {{ inteligencia: "Predição", deadstock: "Dead Stock", geral: "Lista Geral" }[tab]}
                </button>
              ))}
            </div>
          </div>

          {loading || !kpis ? (
            <div className="h-[120px] rounded-xl flex items-center justify-center border" style={{ backgroundColor: "var(--color-surface-container-lowest)", borderColor: "var(--color-outline-variant)" }}>
               <Loader2 size={24} className="animate-spin text-blue-500" />
            </div>
          ) : (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <KpiCard 
                label="Total de SKUs" 
                value={kpis.total_skus} 
                icon={<Package size={16} color="var(--color-secondary)" />} 
                accent="var(--color-primary-fixed)" 
              />
              <KpiCard 
                label="Capital Imobilizado" 
                value={fmtCompact(kpis.valor_total_estoque)} 
                icon={<DollarSign size={16} color="var(--color-secondary)" />} 
                accent="var(--color-primary-fixed)" 
              />
              <KpiCard 
                label="Itens em Alerta (≤ Mín)" 
                value={kpis.itens_em_alerta} 
                icon={<AlertTriangle size={16} color="#ef4444" />} 
                accent="#fee2e2"
                highlightClass={kpis.itens_em_alerta > 0 ? "bg-red-50 border-red-200 text-red-900" : ""}
              />
              <KpiCard 
                label="Taxa de Ruptura (Zerado)" 
                value={`${kpis.taxa_ruptura}%`} 
                icon={<Activity size={16} color="var(--color-on-surface-variant)" />} 
                accent="var(--color-surface-container)" 
              />
            </div>
          )}
        </div>

        {/* ── CONTEÚDO DAS ABAS ──────────────────────────────────── */}
        {loading ? (
          <div className="h-[400px] flex items-center justify-center">
            <Loader2 size={32} className="animate-spin text-blue-500" />
          </div>
        ) : (
          <div className="flex flex-col gap-6">
            
            {/* ABA: INTELIGÊNCIA (Alertas + Fila) */}
            {abaAtiva === 'inteligencia' && (
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                
                {/* Alertas Críticos (Tabela Menor) */}
                <div className="rounded-xl border overflow-hidden flex flex-col" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
                  <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
                    <div className="flex items-center gap-2">
                      <AlertCircle size={16} className="text-red-500" />
                      <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>Alertas de Ruptura</h3>
                    </div>
                  </div>
                  <div className="overflow-x-auto max-h-[500px]">
                    <table className="w-full text-left text-sm">
                      <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)" }}>
                        <tr>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Produto</th>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Físico</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[var(--color-outline-variant)]">
                        {alertas.length === 0 ? (
                          <tr><td colSpan={2} className="px-5 py-8 text-center text-xs text-slate-500">Nenhum alerta no momento.</td></tr>
                        ) : (
                          alertas.map((alerta) => (
                            <tr key={alerta.produto_key} className="transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20">
                              <td className="px-5 py-3">
                                <div className="font-semibold text-[13px]" style={{ color: "var(--color-on-surface)" }}>{alerta.nome}</div>
                                <div className="text-[11px] text-slate-500">Mín: {alerta.quantidade_min}</div>
                              </td>
                              <td className="px-5 py-3 text-right">
                                <span className={`px-2 py-1 rounded-md text-[11px] font-bold ${alerta.quantidade <= 0 ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>
                                  {alerta.quantidade}
                                </span>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Fila Preditiva (Tabela Maior) */}
                <div className="lg:col-span-2 rounded-xl border overflow-hidden flex flex-col" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
                  <div className="p-5 border-b flex items-center justify-between" style={{ borderColor:"var(--color-outline-variant)" }}>
                    <div className="flex items-center gap-2">
                      <ShoppingCart size={16} color="var(--color-on-surface-variant)" />
                      <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>Fila Inteligente de Compras</h3>
                    </div>
                  </div>
                  <div className="overflow-x-auto max-h-[500px]">
                    <table className="w-full text-left text-sm">
                      <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)", position: "sticky", top: 0, zIndex: 10 }}>
                        <tr>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Produto / Classe</th>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Est. Físico</th>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Demanda (30d)</th>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-center">Risco</th>
                          <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Comprar</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[var(--color-outline-variant)]">
                        {reposicao.length === 0 ? (
                          <tr><td colSpan={5} className="px-5 py-8 text-center text-xs text-slate-500">Estoque Saudável. Nenhuma sugestão preditiva.</td></tr>
                        ) : (
                          reposicao.map((item) => (
                            <tr key={item.id} className="transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20">
                              <td className="px-5 py-3">
                                <div className="font-semibold text-[13px]" style={{ color: "var(--color-on-surface)" }}>{item.nome}</div>
                                <div className="flex items-center gap-2 mt-1">
                                  <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400">Classe {item.classe_abc}</span>
                                  <span className="text-[11px] text-slate-500">{item.categoria}</span>
                                </div>
                              </td>
                              <td className="px-5 py-3 text-right font-medium">{item.estoque_atual}</td>
                              <td className="px-5 py-3 text-right text-slate-500">{item.demanda_prevista}</td>
                              <td className="px-5 py-3 text-center">{renderBadgeUrgencia(item.urgencia, item.dias_ate_ruptura)}</td>
                              <td className="px-5 py-3 text-right font-bold text-blue-600 dark:text-blue-400">{item.quantidade_sugerida > 0 ? `+${item.quantidade_sugerida}` : '-'}</td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            )}

            {/* ABA: DEAD STOCK */}
            {abaAtiva === 'deadstock' && (
              <div className="rounded-xl border overflow-hidden flex flex-col" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
                <div className="p-5 border-b flex items-center justify-between" style={{ borderColor:"var(--color-outline-variant)" }}>
                  <div className="flex items-center gap-2">
                    <Archive size={16} color="var(--color-on-surface-variant)" />
                    <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>Capital Parado (Sem giro há 90+ dias)</h3>
                  </div>
                  <span className="text-sm font-bold text-red-600">
                    Total: {fmt(deadStock.reduce((acc, item) => acc + item.CapitalParado, 0))}
                  </span>
                </div>
                <div className="overflow-x-auto max-h-[600px]">
                  <table className="w-full text-left text-sm">
                    <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)", position: "sticky", top: 0, zIndex: 10 }}>
                      <tr>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Produto</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Saldo Físico</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Custo Unitário</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Capital Imobilizado</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[var(--color-outline-variant)]">
                      {deadStock.map((item, idx) => (
                        <tr key={idx} className="transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20">
                          <td className="px-5 py-3 font-semibold text-[13px]" style={{ color: "var(--color-on-surface)" }}>{item.Nome}</td>
                          <td className="px-5 py-3 text-right font-medium">{item.Quantidade} un</td>
                          <td className="px-5 py-3 text-right text-slate-500">{fmt(item.PrecoCusto)}</td>
                          <td className="px-5 py-3 text-right font-bold text-red-600">{fmt(item.CapitalParado)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* ABA: GERAL */}
            {abaAtiva === 'geral' && (
              <div className="rounded-xl border overflow-hidden flex flex-col" style={{ backgroundColor:"var(--color-surface-container-lowest)", borderColor:"var(--color-outline-variant)", boxShadow:"var(--shadow-sm)" }}>
                <div className="p-5 border-b" style={{ borderColor:"var(--color-outline-variant)" }}>
                  <div className="flex items-center gap-2">
                    <Package size={16} color="var(--color-on-surface-variant)" />
                    <h3 className="text-xs font-semibold uppercase tracking-wide" style={{ color:"var(--color-on-surface-variant)" }}>Lista Consolidada de Produtos</h3>
                  </div>
                </div>
                <div className="overflow-x-auto max-h-[600px]">
                  <table className="w-full text-left text-sm">
                    <thead style={{ backgroundColor: "var(--color-surface-container-low)", color: "var(--color-on-surface-variant)", position: "sticky", top: 0, zIndex: 10 }}>
                      <tr>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Cód / Produto</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide">Categoria</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Saldo</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Mínimo</th>
                        <th className="px-5 py-3 font-semibold text-xs uppercase tracking-wide text-right">Valor Venda</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[var(--color-outline-variant)]">
                      {estoqueCompleto.map((item) => (
                        <tr key={item.produto_key} className={`transition-colors hover:bg-slate-50/50 dark:hover:bg-slate-800/20 ${item.alerta ? 'bg-red-50/50 dark:bg-red-900/10' : ''}`}>
                          <td className="px-5 py-3">
                            <div className="font-semibold text-[13px]" style={{ color: "var(--color-on-surface)" }}>{item.nome}</div>
                            <div className="text-[11px] text-slate-500">Ref: {item.produto_key}</div>
                          </td>
                          <td className="px-5 py-3 text-slate-500">{item.categoria || '-'}</td>
                          <td className={`px-5 py-3 text-right font-medium ${item.alerta ? 'text-red-600' : 'text-slate-700 dark:text-slate-300'}`}>
                            {item.quantidade}
                          </td>
                          <td className="px-5 py-3 text-right text-slate-500">{item.quantidade_min}</td>
                          <td className="px-5 py-3 text-right font-medium text-blue-600 dark:text-blue-400">{fmt(item.preco_venda)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

          </div>
        )}
      </div>
    </> 
  )
}