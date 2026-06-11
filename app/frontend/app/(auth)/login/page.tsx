"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Eye, EyeOff, Loader2 } from "lucide-react"
import api from "@/lib/api"
import { saveSession } from "@/lib/auth"
import Logo from "@/components/ui/Logo"

export default function LoginPage() {
  const router = useRouter()

  const [email,    setEmail]    = useState("")
  const [password, setPassword] = useState("")
  const [showPass, setShowPass] = useState(false)
  const [remember, setRemember] = useState(false)
  const [loading,  setLoading]  = useState(false)
  const [error,    setError]    = useState("")

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")

    if (!email || !password) {
      setError("Preencha e-mail e senha.")
      return
    }

    setLoading(true)
    try {
      const { data } = await api.post("/auth/login", { email, password })

      saveSession({
        token: data.token,
        user: {
          id:         data.user_id ?? "",
          name:       data.name,
          email:      email,
          role:       data.role,
          client_key: data.client_key,
        },
      })

      router.push("/home")
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      setError(error.response?.data?.error ?? "E-mail ou senha incorretos.")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="w-full max-w-[420px] border rounded-xl p-8 flex flex-col gap-6"
      style={{
        backgroundColor: "var(--color-surface-container-lowest)",
        borderColor:     "var(--color-outline-variant)",
        boxShadow:       "0px 4px 20px rgba(15, 23, 42, 0.08)",
      }}
    >
      {/* Logo e título */}
      <div className="flex flex-col items-center text-center gap-4">
        <Logo size="lg" showText={false} />

        <div>
          <h1
            className="text-2xl font-bold"
            style={{
              fontFamily: "var(--font-display)",
              color:      "var(--color-on-surface)",
            }}
          >
            Bem-vindo de volta
          </h1>
          <p className="text-sm mt-1" style={{ color: "var(--color-on-surface-variant)" }}>
            Entre com suas credenciais para acessar o painel.
          </p>
        </div>
      </div>

      {/* Formulário */}
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">

        {/* E-mail */}
        <div className="flex flex-col gap-1">
          <label
            htmlFor="email"
            className="text-xs font-semibold tracking-wide uppercase"
            style={{ color: "var(--color-on-surface)" }}
          >
            E-mail corporativo
          </label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="seu@email.com"
            autoComplete="email"
            className="w-full rounded-lg px-4 py-2.5 text-sm border transition-all outline-none"
            style={{
              backgroundColor: "var(--color-surface-container-lowest)",
              borderColor:     "var(--color-outline-variant)",
              color:           "var(--color-on-surface)",
            }}
            onFocus={(e) => {
              e.target.style.borderColor = "var(--color-secondary)"
              e.target.style.boxShadow   = "0 0 0 3px rgba(0, 81, 213, 0.1)"
            }}
            onBlur={(e) => {
              e.target.style.borderColor = "var(--color-outline-variant)"
              e.target.style.boxShadow   = "none"
            }}
          />
        </div>

        {/* Senha */}
        <div className="flex flex-col gap-1">
          <label
            htmlFor="password"
            className="text-xs font-semibold tracking-wide uppercase"
            style={{ color: "var(--color-on-surface)" }}
          >
            Senha
          </label>
          <div className="relative">
            <input
              id="password"
              type={showPass ? "text" : "password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="current-password"
              className="w-full rounded-lg px-4 py-2.5 pr-10 text-sm border transition-all outline-none"
              style={{
                backgroundColor: "var(--color-surface-container-lowest)",
                borderColor:     "var(--color-outline-variant)",
                color:           "var(--color-on-surface)",
              }}
              onFocus={(e) => {
                e.target.style.borderColor = "var(--color-secondary)"
                e.target.style.boxShadow   = "0 0 0 3px rgba(0, 81, 213, 0.1)"
              }}
              onBlur={(e) => {
                e.target.style.borderColor = "var(--color-outline-variant)"
                e.target.style.boxShadow   = "none"
              }}
            />
            <button
              type="button"
              onClick={() => setShowPass(!showPass)}
              className="absolute right-3 top-1/2 -translate-y-1/2 transition-colors"
              style={{ color: "var(--color-on-surface-variant)" }}
            >
              {showPass ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
        </div>

        {/* Lembrar + Esqueci */}
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
              className="w-4 h-4 rounded"
              style={{ accentColor: "var(--color-secondary)" }}
            />
            <span className="text-sm" style={{ color: "var(--color-on-surface-variant)" }}>
              Lembrar de mim
            </span>
          </label>
          <Link
            href="/esqueci-senha"
            className="text-xs font-semibold tracking-wide transition-colors"
            style={{ color: "var(--color-secondary)" }}
          >
            Esqueceu a senha?
          </Link>
        </div>

        {/* Erro */}
        {error && (
          <div
            className="text-sm px-4 py-3 rounded-lg"
            style={{
              backgroundColor: "var(--color-error-container)",
              color:           "var(--color-on-error-container)",
            }}
          >
            {error}
          </div>
        )}

        {/* Botão */}
        <button
          type="submit"
          disabled={loading}
          className="w-full py-3 rounded-lg text-sm font-semibold tracking-wide transition-all flex items-center justify-center gap-2 mt-1"
          style={{
            backgroundColor: loading ? "var(--color-on-surface-variant)" : "var(--color-inverse-surface)",
            color:           "var(--color-inverse-on-surface)",
            cursor:          loading ? "not-allowed" : "pointer",
          }}
        >
          {loading && <Loader2 size={16} className="animate-spin" />}
          {loading ? "Entrando..." : "Entrar"}
        </button>
      </form>

      {/* Rodapé */}
      <div
        className="pt-4 border-t text-center"
        style={{ borderColor: "var(--color-outline-variant)" }}
      >
        <p className="text-sm" style={{ color: "var(--color-on-surface-variant)" }}>
          Não tem uma conta?{" "}
          <a
            href="mailto:contato@lume.com.br"
            className="font-semibold transition-colors"
            style={{ color: "var(--color-secondary)" }}
          >
            Solicite acesso
          </a>
        </p>
      </div>
    </div>
  )
}