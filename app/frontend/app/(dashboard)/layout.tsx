import Sidebar from "@/components/layout/Sidebar"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <div className="flex-1 flex flex-col md:ml-64 h-screen overflow-hidden">
        <main className="flex-1 overflow-y-auto p-margin">
          {children}
        </main>
      </div>
    </div>
  )
}