import { Link, NavLink, Route, Routes } from "react-router-dom"
import TenantsPage from "@/pages/TenantsPage"
import ServicesPage from "@/pages/ServicesPage"
import BillingPage from "@/pages/BillingPage"

function NavItem({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        [
          "flex items-center gap-2 rounded-xl px-3 py-2 text-sm transition",
          isActive
            ? "bg-white/10 text-white ring-1 ring-white/15"
            : "text-white/70 hover:bg-white/5 hover:text-white",
        ].join(" ")
      }
    >
      <span className="h-2 w-2 rounded-full bg-white/30" />
      {label}
    </NavLink>
  )
}

export default function Layout() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-50">
      {/* background */}
      <div className="pointer-events-none fixed inset-0">
        <div className="absolute -top-40 left-1/2 h-[520px] w-[800px] -translate-x-1/2 rounded-full bg-gradient-to-r from-indigo-500/15 via-fuchsia-500/10 to-cyan-500/10 blur-3xl" />
        <div className="absolute -bottom-40 right-0 h-[420px] w-[620px] rounded-full bg-gradient-to-r from-emerald-500/10 to-sky-500/10 blur-3xl" />
      </div>

      <div className="relative mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-6 md:grid-cols-[260px_1fr]">
        <aside className="h-fit rounded-2xl border border-white/10 bg-zinc-900/30 p-4 backdrop-blur">
          <Link to="/" className="block rounded-xl px-2 py-2">
            <div className="text-lg font-semibold tracking-tight">FaaS Billing</div>
            <div className="text-xs text-white/50">Dashboard</div>
          </Link>

          <div className="mt-4 space-y-1">
            <NavItem to="/tenants" label="Tenants" />
            <NavItem to="/services" label="Services" />
            <NavItem to="/billing" label="Billing" />
          </div>

          <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-3 text-xs text-white/70">
            Backend:
            <div className="mt-1 break-all text-white/90">
              {import.meta.env.VITE_API_BASE_URL}
            </div>
          </div>
        </aside>

        <main className="rounded-2xl border border-white/10 bg-zinc-900/20 p-5 backdrop-blur">
          <Routes>
            <Route path="/" element={<TenantsPage />} />
            <Route path="/tenants" element={<TenantsPage />} />
            <Route path="/services" element={<ServicesPage />} />
            <Route path="/billing" element={<BillingPage />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
