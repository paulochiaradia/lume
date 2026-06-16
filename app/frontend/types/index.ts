// ── Auth ─────────────────────────────────────────────────────
export interface User {
  id: string
  name: string
  email: string
  role: "admin" | "gerente" | "compras" | "viewer"
  client_key: string
  must_change_password?: boolean
}

export interface AuthSession {
  token: string
  user: User
}

// ── API ──────────────────────────────────────────────────────
export interface ApiResponse<T> {
  data?: T
  error?: string
}

// ── KPIs ─────────────────────────────────────────────────────
export interface HomeKPIs {
  faturamento: number
  ticket_medio: number
  total_vendas: number
  itens_por_venda: number
}

export interface VendaDia {
  dia: string
  vendas: number
  faturamento: number
}

// ── Produtos ─────────────────────────────────────────────────
export interface ProdutoABC {
  produto_key: string
  nome: string
  categoria: string
  faturamento: number
  classe: "A" | "B" | "C"
}

// ── Clientes ─────────────────────────────────────────────────
export interface ClienteRFM {
  cliente_id: string
  recencia: number
  frequencia: number
  valor_total: number
  score_r: number
  score_f: number
  score_m: number
  score_rfm: string
  score_total: number
  segmento: string
}

export interface ResumoSegmento {
  segmento: string
  clientes: number
  valor_medio: number
  valor_total: number
  recencia_media: number
}

// ── Estoque ──────────────────────────────────────────────────
export interface EstoqueItem {
  produto_key: string
  nome: string
  categoria: string
  quantidade: number
  quantidade_min: number
  preco_venda: number
  preco_custo: number
  alerta: boolean
}

// ── Usuários ─────────────────────────────────────────────────
export interface UsuarioLoja {
  id: string
  name: string
  email: string
  role: string
  active: boolean
}

export interface VendaHora {
  hora: string;
  faturamento: number;
}

export interface MixCategoria {
  categoria: string;
  faturamento: number;
  fill?: string; // Colocamos a interrogação (?) porque essa cor nós injetamos no frontend, a API não manda
}