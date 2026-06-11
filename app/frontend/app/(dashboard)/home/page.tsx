import Header from "@/components/layout/Header"

export default function HomePage() {
  return (
    <>
      <Header title="Dashboard" />
      <div className="p-8 flex flex-col gap-4">
        <div className="grid grid-cols-4 gap-4">
          <div style={{backgroundColor: 'var(--color-secondary)'}} className="text-white rounded-lg p-4">
            <p className="text-sm font-semibold">Faturamento</p>
            <p className="text-2xl font-bold mt-1">R$ 23.330</p>
          </div>
          <div style={{backgroundColor: 'var(--color-inverse-surface)'}} className="text-white rounded-lg p-4">
            <p className="text-sm font-semibold">Ticket Médio</p>
            <p className="text-2xl font-bold mt-1">R$ 2.333</p>
          </div>
          <div style={{backgroundColor: 'var(--color-surface-container-low)'}} className="rounded-lg p-4">
            <p className="text-sm font-semibold" style={{color: 'var(--color-on-surface-variant)'}}>Total de Vendas</p>
            <p className="text-2xl font-bold mt-1" style={{color: 'var(--color-on-surface)'}}>10</p>
          </div>
          <div style={{backgroundColor: 'var(--color-surface-container-low)'}} className="rounded-lg p-4">
            <p className="text-sm font-semibold" style={{color: 'var(--color-on-surface-variant)'}}>Descontos</p>
            <p className="text-2xl font-bold mt-1" style={{color: 'var(--color-on-surface)'}}>R$ 1.050</p>
          </div>
        </div>
        <div style={{backgroundColor: 'var(--color-surface-container-lowest)', borderColor: 'var(--color-outline-variant)'}} className="border rounded-lg p-6">
          <p style={{color: 'var(--color-secondary)'}} className="font-semibold">Cor secondary: #0051d5</p>
          <p style={{color: 'var(--color-on-surface-variant)'}} className="text-sm mt-1">Cor on-surface-variant: #45464d</p>
          <p style={{fontFamily: 'var(--font-display)'}} className="text-xl font-bold mt-2">Hanken Grotesk via --font-display</p>
          <p style={{fontFamily: 'var(--font-body)'}} className="mt-1">Inter via --font-body</p>
        </div>
      </div>
    </>
  )
}