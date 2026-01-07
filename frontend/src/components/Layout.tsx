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
          "block rounded-lg px-3 py-2 text-sm",
          isActive ? "bg-white/10 text-white" : "text-white/70 hover:bg-white/5",
        ].join(" ")
      }
    >
      {label}
    </NavLink>
  )
}

export default function Layout() {
  return (
    <div className="min-h-full bg-zinc-950 text-zinc-50">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-6 md:grid-cols-[240px_1fr]">
        <aside className="rounded-2xl border border-white/10 bg-zinc-900/40 p-4">
          <Link to="/" className="block px-2 pb-3 text-lg font-semibold">
            FaaS Billing
          </Link>
          <div className="space-y-1">
            <NavItem to="/tenants" label="Tenants" />
            <NavItem to="/services" label="Services" />
            <NavItem to="/billing" label="Billing" />
          </div>

          <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-3 text-xs text-white/70">
            Backend: <span className="text-white/90">{import.meta.env.VITE_API_BASE_URL}</span>
          </div>
        </aside>

        <main className="rounded-2xl border border-white/10 bg-zinc-900/30 p-5">
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
