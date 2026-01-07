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
          "flex items-center justify-between rounded-xl px-3 py-2 text-sm transition",
          isActive
            ? "bg-white/10 text-white ring-1 ring-white/15"
            : "text-white/70 hover:bg-white/5 hover:text-white",
        ].join(" ")
      }
    >
      <span className="flex items-center gap-2">
        <span className="h-2 w-2 rounded-full bg-white/30" />
        {label}
      </span>
      <span className="text-xs text-white/30">→</span>
    </NavLink>
  )
}

export default function Layout() {
  return (
    <div className="app-bg bg-aurora">
      <div className="mx-auto grid max-w-6xl grid-cols-1 gap-6 px-4 py-6 md:grid-cols-[270px_1fr]">
        <aside className="glass-strong p-4">
          <Link to="/" className="block rounded-xl px-2 py-2">
            <div className="text-lg font-semibold tracking-tight">FaaS Billing</div>
            <div className="muted">Tenants • Services • Billing</div>
          </Link>

          <div className="mt-4 space-y-1">
            <NavItem to="/tenants" label="Tenants" />
            <NavItem to="/services" label="Services" />
            <NavItem to="/billing" label="Billing" />
          </div>

          <div className="mt-6 card !p-3">
            <div className="muted">Backend</div>
            <div className="mt-1 break-all text-xs text-white/80">
              {import.meta.env.VITE_API_BASE_URL}
            </div>
          </div>

          <div className="mt-4 muted">
            Tip: включи dark theme в браузере — UI заточен под dark.
          </div>
        </aside>

        <main className="glass-strong p-5">
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
