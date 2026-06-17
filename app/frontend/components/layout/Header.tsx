"use client"

import { useEffect, useState } from "react"
import { Search, Bell } from "lucide-react"
import { getUser } from "@/lib/auth"

interface HeaderProps {
  title: string
}

export default function Header({ title }: HeaderProps) {
  const [user, setUser] = useState<ReturnType<typeof getUser>>(null)
  const [search, setSearch] = useState("")

  useEffect(() => {
    setUser(getUser())
  }, [])

  const roleLabels: Record<string, string> = {
    admin:   "Administrador",
    gerente: "Gerente",
    compras: "Compras",
    viewer:  "Visualizador",
  }

return (
  <header
    className="flex justify-between items-center h-16 flex-shrink-0 border-b mb-6 -mx-8 px-8 sticky top-0 z-10"
    style={{
      backgroundColor: "var(--color-surface-container-lowest)",
      borderColor:     "var(--color-outline-variant)",
    }}
  >
      {/* Título */}
      <h1
        className="text-2xl font-bold tracking-tight"
        style={{
          fontFamily: "var(--font-display)",
          color:      "var(--color-on-surface)",
        }}
      >
        {title}
      </h1>

      {/* Ações */}
      <div className="flex items-center gap-4">

        {/* Busca */}
        <div className="relative">
          <Search
            size={15}
            className="absolute left-2.5 top-1/2 -translate-y-1/2"
            style={{ color: "var(--color-on-surface-variant)" }}
          />
          <input
            type="text"
            placeholder="Buscar..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8 pr-4 py-1.5 rounded text-sm w-56 transition-colors outline-none border"
            style={{
              backgroundColor: "var(--color-surface-container-low)",
              borderColor:     "var(--color-outline-variant)",
              color:           "var(--color-on-surface)",
            }}
          />
        </div>

        {/* Notificações */}
        <button
          className="p-1.5 rounded-lg transition-colors"
          style={{ color: "var(--color-on-surface-variant)" }}
        >
          <Bell size={20} />
        </button>

        {/* Perfil */}
        {user && (
          <div
            className="flex items-center gap-2 pl-4 border-l"
            style={{ borderColor: "var(--color-outline-variant)" }}
          >
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold"
              style={{
                backgroundColor: "var(--color-secondary-container)",
                color:           "var(--color-on-secondary-container)",
              }}
            >
              {user.name.charAt(0).toUpperCase()}
            </div>
            <div className="hidden lg:block">
              <p
                className="text-sm font-semibold leading-tight"
                style={{ color: "var(--color-on-surface)" }}
              >
                {user.name.split(" ")[0]}
              </p>
              <p
                className="text-xs leading-tight"
                style={{ color: "var(--color-on-surface-variant)" }}
              >
                {roleLabels[user.role] ?? user.role}
              </p>
            </div>
          </div>
        )}
      </div>
    </header>
  )
}