"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { getUser, PAGES_BY_ROLE, clearSession } from "@/lib/auth"
import {
  LayoutDashboard, TrendingUp, Package, Users,
  ShoppingCart, AlertTriangle, DollarSign,
  FileText, UserCog, Settings, LogOut,
} from "lucide-react"

import Logo from "@/components/ui/Logo"

const ALL_NAV_ITEMS = [
  { key: "home",          label: "Dashboard",      icon: LayoutDashboard, href: "/home" },
  { key: "vendas",        label: "Vendas",          icon: TrendingUp,      href: "/vendas" },
  { key: "estoque",       label: "Estoque",         icon: Package,         href: "/estoque" },
  { key: "clientes",      label: "Clientes",        icon: Users,           href: "/clientes" },
  { key: "produtos",      label: "Produtos",        icon: ShoppingCart,    href: "/produtos" },
  { key: "anomalias",     label: "Anomalias",       icon: AlertTriangle,   href: "/anomalias" },
  { key: "preco",         label: "Preço & Margem",  icon: DollarSign,      href: "/preco" },
  { key: "relatorios",    label: "Relatórios",      icon: FileText,        href: "/relatorios" },
  { key: "usuarios",      label: "Usuários",        icon: UserCog,         href: "/usuarios" },
  { key: "configuracoes", label: "Configurações",   icon: Settings,        href: "/configuracoes" },
]

export default function Sidebar() {
  const pathname = usePathname()
  const user     = getUser()
  const role     = user?.role ?? "viewer"
  const allowed  = PAGES_BY_ROLE[role] ?? []
  const navItems = ALL_NAV_ITEMS.filter(item => allowed.includes(item.key))

  return (
    <nav
      className="h-screen w-64 fixed left-0 top-0 flex flex-col py-8 px-2 z-20 border-r"
      style={{
        backgroundColor: "var(--color-surface-bright)",
        borderColor:     "var(--color-outline-variant)",
      }}
    >
      {/* Logo */}
      <div className="mb-6 px-2">
        <Logo size="md" showText={true} />
      </div>

      {/* Itens de navegação */}
      <ul className="flex flex-col gap-1 flex-1">
        {navItems.map((item) => {
          const Icon     = item.icon
          const isActive = pathname.startsWith(item.href)

          return (
            <li key={item.key}>
              <Link
                href={item.href}
                className="flex items-center gap-4 px-2 py-2 rounded-lg transition-colors"
                style={{
                  backgroundColor: isActive ? "var(--color-surface-container-low)" : "transparent",
                  color:           isActive ? "var(--color-secondary)"              : "var(--color-on-surface-variant)",
                  fontWeight:      isActive ? "600"                                 : "400",
                  borderRight:     isActive ? "3px solid var(--color-secondary)"    : "3px solid transparent",
                }}
                onMouseEnter={(e) => {
                  if (!isActive) {
                    e.currentTarget.style.backgroundColor = "var(--color-surface-container)"
                    e.currentTarget.style.color           = "var(--color-on-surface)"
                  }
                }}
                onMouseLeave={(e) => {
                  if (!isActive) {
                    e.currentTarget.style.backgroundColor = "transparent"
                    e.currentTarget.style.color           = "var(--color-on-surface-variant)"
                  }
                }}
              >
                <Icon size={20} />
                <span className="text-sm">{item.label}</span>
              </Link>
            </li>
          )
        })}
      </ul>

      {/* Usuário logado */}
      {user && (
        <div
          className="border-t pt-4 mt-4"
          style={{ borderColor: "var(--color-outline-variant)" }}
        >
          <div className="px-2 mb-2">
            <p className="text-sm font-semibold truncate" style={{ color: "var(--color-on-surface)" }}>
              {user.name}
            </p>
            <p className="text-xs truncate" style={{ color: "var(--color-on-surface-variant)" }}>
              {user.email}
            </p>
          </div>
          <button
            onClick={() => {
              clearSession()
              window.location.href = "/login"
            }}
            className="flex items-center gap-4 px-2 py-2 rounded-lg transition-colors w-full text-sm"
            style={{ color: "var(--color-on-surface-variant)" }}
            onMouseEnter={(e) => {
              e.currentTarget.style.backgroundColor = "var(--color-surface-container)"
              e.currentTarget.style.color           = "var(--color-error)"
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.backgroundColor = "transparent"
              e.currentTarget.style.color           = "var(--color-on-surface-variant)"
            }}
          >
            <LogOut size={18} />
            <span>Sair</span>
          </button>
        </div>
      )}
    </nav>
  )
}