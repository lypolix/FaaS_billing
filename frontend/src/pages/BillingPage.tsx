import { useMemo, useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import {
  calculateBill,
  generateBill,
  getUsageAggregates,
  listServices,
  listTenants,
} from "@/lib/api"
import { endOfDay, startOfDay, toRFC3339 } from "@/lib/time"
import { isBillPaid, markBillPaid } from "@/lib/fakePayments"

function StatCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <div className="card">
      <div className="muted">{label}</div>
      <div className="mt-2 text-2xl font-semibold tracking-tight">{value}</div>
      {hint ? <div className="mt-1 text-xs text-white/50">{hint}</div> : null}
    </div>
  )
}

export default function BillingPage() {
  const tenantsQ = useQuery({ queryKey: ["tenants"], queryFn: listTenants })
  const servicesQ = useQuery({
    queryKey: ["services"],
    queryFn: () => listServices(),
  })

  const [tenantId, setTenantId] = useState("")
  const [serviceId, setServiceId] = useState("")
  const [start, setStart] = useState(() => toRFC3339(startOfDay()))
  const [end, setEnd] = useState(() => toRFC3339(endOfDay()))
  const [lastBill, setLastBill] = useState<any>(null)

  const aggsQ = useQuery({
    queryKey: ["usage-aggregates", tenantId, serviceId, start, end],
    queryFn: () =>
      getUsageAggregates({
        tenant_id: tenantId || undefined,
        service_id: serviceId || undefined,
        start_time: start,
        end_time: end,
      }),
    enabled: Boolean(tenantId),
  })

  const calcM = useMutation({
    mutationFn: () =>
      calculateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const genM = useMutation({
    mutationFn: () =>
      generateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const billKey = useMemo(
    () => (tenantId ? `${tenantId}:${start}:${end}` : ""),
    [tenantId, start, end],
  )
  const paid = billKey ? isBillPaid(billKey) : false

  const aggs = aggsQ.data?.data ?? []

  const totals = useMemo(() => {
    const inv = aggs.reduce((acc, a) => acc + (a.invocations ?? 0), 0)
    const ms = aggs.reduce((acc, a) => acc + (a.duration_ms ?? 0), 0)
    const cost = aggs.reduce((acc, a) => acc + (a.cost ?? 0), 0)
    return { inv, ms, cost }
  }, [aggs])

  const selectedTenantName = useMemo(() => {
    const t = (tenantsQ.data ?? []).find((x) => x.id === tenantId)
    return t?.name ?? ""
  }, [tenantsQ.data, tenantId])

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div className="card !p-0 overflow-hidden">
          <div className="p-5">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="title">Billing</h1>
              <span className="chip">Tenant-based</span>
              <span className={paid ? "chip-ok" : "chip-warn"}>
                {billKey ? (paid ? "PAID (fake)" : "UNPAID") : "Select tenant"}
              </span>
            </div>
            <p className="mt-2 subtitle">
              Выбери tenant и период — посчитай стоимость и сгенерируй счёт.
              “Оплата” пока фейковая (localStorage).
            </p>

            {tenantId ? (
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <span className="chip">
                  Tenant: <span className="text-white/90">{selectedTenantName || tenantId}</span>
                </span>
                {serviceId ? (
                  <span className="chip">
                    Service filter: <span className="text-white/90">{serviceId}</span>
                  </span>
                ) : (
                  <span className="chip">All services</span>
                )}
              </div>
            ) : null}
          </div>

          <div className="border-t border-white/10 bg-black/20 px-5 py-3">
            <div className="muted">
              Bill key:{" "}
              <span className="text-white/70 break-all">
                {billKey || "—"}
              </span>
            </div>
          </div>
        </div>

        <div className="flex gap-2">
          <button
            className="btn-ghost"
            onClick={() => {
              setStart(toRFC3339(startOfDay()))
              setEnd(toRFC3339(endOfDay()))
            }}
          >
            Today
          </button>
        </div>
      </header>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard
          label="Invocations"
          value={tenantId ? String(totals.inv) : "—"}
          hint="Sum over usage aggregates"
        />
        <StatCard
          label="Duration (ms)"
          value={tenantId ? String(totals.ms) : "—"}
          hint="Sum over usage aggregates"
        />
        <StatCard
          label="Cost (agg sum)"
          value={tenantId ? String(totals.cost) : "—"}
          hint="If your aggregates contain cost"
        />
      </section>

      <section className="card">
        <div className="card-header">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <div className="text-sm font-medium">Billing query</div>
              <div className="muted">RFC3339 time range, tenant required.</div>
            </div>
            <div className="flex items-center gap-2">
              <button
                className="btn-primary"
                onClick={() => calcM.mutate()}
                disabled={!tenantId || calcM.isPending}
              >
                Calculate
              </button>
              <button
                className="btn-ghost"
                onClick={() => genM.mutate()}
                disabled={!tenantId || genM.isPending}
              >
                Generate bill
              </button>
              <button
                className="btn-success"
                onClick={() => billKey && markBillPaid(billKey)}
                disabled={!billKey}
                title="Fake payment: stored locally"
              >
                Pay (fake)
              </button>
            </div>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
          <div>
            <div className="muted mb-2">Tenant</div>
            <select
              className="select"
              value={tenantId}
              onChange={(e) => setTenantId(e.target.value)}
            >
              <option value="">Select tenant</option>
              {(tenantsQ.data ?? []).map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <div className="muted mb-2">Service (optional)</div>
            <select
              className="select"
              value={serviceId}
              onChange={(e) => setServiceId(e.target.value)}
              disabled={!tenantId}
            >
              <option value="">All services</option>
              {(servicesQ.data ?? [])
                .filter((s) => (tenantId ? s.tenant_id === tenantId : true))
                .map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
            </select>
          </div>

          <div>
            <div className="muted mb-2">Start time</div>
            <input
              className="input"
              value={start}
              onChange={(e) => setStart(e.target.value)}
              placeholder="start_time RFC3339"
            />
          </div>

          <div>
            <div className="muted mb-2">End time</div>
            <input
              className="input"
              value={end}
              onChange={(e) => setEnd(e.target.value)}
              placeholder="end_time RFC3339"
            />
          </div>
        </div>

        {(calcM.isError || genM.isError) ? (
          <div className="mt-4 rounded-xl border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-200">
            Error:{" "}
            {(calcM.error as any)?.message ??
              (genM.error as any)?.message ??
              "failed"}
          </div>
        ) : null}
      </section>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="card">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-medium">Usage aggregates</div>
              <div className="muted">Данные из /usage-aggregates (top 12).</div>
            </div>
            {aggsQ.isFetching ? <span className="chip">Loading…</span> : <span className="chip">Live</span>}
          </div>

          <div className="mt-4 overflow-hidden rounded-xl border border-white/10">
            <div className="overflow-x-auto">
              <table className="table">
                <thead className="thead bg-white/[0.03]">
                  <tr>
                    <th className="pl-4">Window</th>
                    <th>Service</th>
                    <th>Inv</th>
                    <th>ms</th>
                    <th className="pr-4">Cost</th>
                  </tr>
                </thead>
                <tbody>
                  {aggsQ.isError ? (
                    <tr className="row">
                      <td className="cell pl-4 text-red-300" colSpan={5}>
                        Error: {(aggsQ.error as any)?.message ?? "failed"}
                      </td>
                    </tr>
                  ) : !tenantId ? (
                    <tr className="row">
                      <td className="cell pl-4 text-white/60" colSpan={5}>
                        Select a tenant to load aggregates.
                      </td>
                    </tr>
                  ) : aggs.length === 0 ? (
                    <tr className="row">
                      <td className="cell pl-4 text-white/60" colSpan={5}>
                        No aggregates for this period/filter.
                      </td>
                    </tr>
                  ) : (
                    aggs.slice(0, 12).map((a, idx) => (
                      <tr key={idx} className="row">
                        <td className="cell pl-4 text-xs text-white/75">
                          {(a.window_start ?? "").split(".")[0]} →{" "}
                          {(a.window_end ?? "").split(".")[0]}
                        </td>
                        <td className="cell text-xs text-white/55">
                          {a.service_id}
                        </td>
                        <td className="cell">{a.invocations ?? "-"}</td>
                        <td className="cell">{a.duration_ms ?? "-"}</td>
                        <td className="cell pr-4">{a.cost ?? "-"}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-sm font-medium">Last bill response</div>
              <div className="muted">
                Ответ от /billing/calculate или /billing/generate.
              </div>
            </div>
            <span className="chip">JSON</span>
          </div>

          <pre className="mt-4 max-h-[420px] overflow-auto rounded-xl border border-white/10 bg-zinc-950/50 p-3 text-xs text-white/85">
{lastBill ? JSON.stringify(lastBill, null, 2) : "No bill yet. Click Calculate or Generate bill."}
          </pre>
        </div>
      </section>
    </div>
  )
}
