import Cookies from "js-cookie"
import { User, AuthSession } from "@/types"

const TOKEN_KEY = "lume_token"
const USER_KEY  = "lume_user"

export function saveSession(session: AuthSession) {
  Cookies.set(TOKEN_KEY, session.token, { expires: 1 }) // 1 dia
  Cookies.set(USER_KEY, JSON.stringify(session.user), { expires: 1 })
}

export function getToken(): string | undefined {
  return Cookies.get(TOKEN_KEY)
}

export function getUser(): User | null {
  const raw = Cookies.get(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function clearSession() {
  Cookies.remove(TOKEN_KEY)
  Cookies.remove(USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

// Páginas permitidas por role
export const PAGES_BY_ROLE: Record<string, string[]> = {
  admin:   ["home", "vendas", "estoque", "clientes", "produtos", "anomalias", "preco", "relatorios", "usuarios", "configuracoes"],
  gerente: ["home", "vendas", "anomalias", "relatorios"],
  compras: ["home", "estoque", "relatorios"],
  viewer:  ["home", "vendas"],
}

export function canAccess(role: string, page: string): boolean {
  return PAGES_BY_ROLE[role]?.includes(page) ?? false
}