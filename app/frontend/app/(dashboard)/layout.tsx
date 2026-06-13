import Sidebar from "@/components/layout/Sidebar"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex h-screen overflow-hidden" style={{ backgroundColor: "var(--color-background)" }}>
      <Sidebar />
      <div
        className="flex flex-col h-screen overflow-hidden"
        style={{ marginLeft: "256px", width: "calc(100% - 256px)" }}
      >
        <main className="flex-1 overflow-y-auto px-8 pb-8">
          {children}
        </main>
      </div>
    </div>
  )
}