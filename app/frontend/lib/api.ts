import axios from "axios"
import Cookies from "js-cookie"

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost/api/v1"

const api = axios.create({
  baseURL: API_URL,
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
})

// Injeta o token JWT em todas as requisições
api.interceptors.request.use((config) => {
  const token = Cookies.get("lume_token")
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Redireciona para login se token expirar
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      Cookies.remove("lume_token")
      Cookies.remove("lume_user")
      window.location.href = "/login"
    }
    return Promise.reject(error)
  }
)

// ── Insights ─────────────────────────────────────────────────
export interface Insight {
  tipo:       string
  prioridade: number
  titulo:     string
  mensagem:   string
  acao:       string
  href:       string
  categoria:  string
  icone:      "success" | "warning" | "danger"
  gerado_em:  string
}

export default api