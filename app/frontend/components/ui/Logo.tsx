import { Zap } from "lucide-react"

interface LogoProps {
  size?: "sm" | "md" | "lg"
  showText?: boolean
}

export default function Logo({ size = "md", showText = true }: LogoProps) {
  const sizes = {
    sm: { container: "w-7 h-7 rounded-lg", icon: 14, text: "text-sm",  sub: "text-[9px]"  },
    md: { container: "w-8 h-8 rounded-lg", icon: 16, text: "text-[15px]", sub: "text-[10px]" },
    lg: { container: "w-12 h-12 rounded-xl", icon: 22, text: "text-2xl", sub: "text-xs"    },
  }

  const s = sizes[size]

  return (
    <div className="flex items-center gap-2">
      {/* Troque o conteúdo abaixo pelo logo oficial quando estiver pronto */}
      <div
        className={`${s.container} flex items-center justify-center flex-shrink-0`}
        style={{ backgroundColor: "var(--color-inverse-surface)" }}
      >
        <Zap size={s.icon} color="var(--color-inverse-on-surface)" />
      </div>

      {showText && (
        <div>
          <span
            className={`${s.text} font-bold tracking-tight block leading-tight`}
            style={{ fontFamily: "var(--font-display)", color: "var(--color-on-surface)" }}
          >
            Lume
          </span>
          <p className={`${s.sub} leading-none`} style={{ color: "var(--color-on-surface-variant)" }}>
            Inteligência Comercial
          </p>
        </div>
      )}
    </div>
  )
}