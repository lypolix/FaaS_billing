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

export default function BillingPage() {
  const tenantsQ = useQuery({ queryKey: ["tenants"], queryFn: listTenants })
  const servicesQ = useQuery({ queryKey: ["services"], queryFn: () => listServices() })

  const [tenantId, setTenantId] = useState("")
  const [serviceId, setServiceId] = useState<string>("")
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
    mutationFn: () => calculateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const genM = useMutation({
    mutationFn: () => generateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const billKey = useMemo(() => {
    if (!tenantId) return ""
    return `${tenantId}:${start}:${end}`
  }, [tenantId, start, end])

  const paid = billKey ? isBillPaid(billKey) : false

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-xl font-semibold">Billing</h1>
        <p className="mt-1 text-sm text-white/60">
          Расчёт идёт через /billing/calculate и /billing/generate; “оплата” пока локальная (fake).
        </p>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <select
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
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

          <select
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
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

          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
            value={start}
            onChange={(e) => setStart(e.target.value)}
            placeholder="start_time RFC3339"
          />
          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
            value={end}
            onChange={(e) => setEnd(e.target.value)}
            placeholder="end_time RFC3339"
          />

          <div className="md:col-span-2 flex flex-wrap gap-2">
            <button
              className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black hover:bg-zinc-200 disabled:opacity-50"
              onClick={() => calcM.mutate()}
              disabled={!tenantId || calcM.isPending}
            >
              Calculate
            </button>
            <button
              className="rounded-xl border border-white/15 bg-transparent px-4 py-2 text-sm font-medium text-white hover:bg-white/5 disabled:opacity-50"
              onClick={() => genM.mutate()}
              disabled={!tenantId || genM.isPending}
            >
              Generate bill
            </button>

            <button
              className="ml-auto rounded-xl bg-emerald-400 px-4 py-2 text-sm font-medium text-black hover:bg-emerald-300 disabled:opacity-50"
              onClick={() => {
                if (!billKey) return
                // amount можно попробовать достать из lastBill, если у тебя там есть total
                markBillPaid(billKey)
              }}
              disabled={!billKey}
            >
              Pay (fake)
            </button>
          </div>

          {billKey ? (
            <div className="md:col-span-2 text-xs text-white/60">
              Bill key: <span className="text-white/80">{billKey}</span> — status:{" "}
              <span className={paid ? "text-emerald-300" : "text-yellow-300"}>
                {paid ? "PAID (fake)" : "UNPAID"}
              </span>
            </div>
          ) : null}
        </div>
      </section>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="text-sm text-white/70">Usage aggregates</div>
        {aggsQ.isFetching ? (
          <div className="mt-3 text-sm text-white/60">Loading...</div>
        ) : aggsQ.isError ? (
          <div className="mt-3 text-sm text-red-400">
            Error: {(aggsQ.error as any)?.message ?? "failed"}
          </div>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="text-xs text-white/50">
                <tr className="border-b border-white/10">
                  <th className="py-2 pr-4">window</th>
                  <th className="py-2 pr-4">service</th>
                  <th className="py-2 pr-4">invocations</th>
                  <th className="py-2 pr-4">duration_ms</th>
                  <th className="py-2 pr-4">cost</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {(aggsQ.data?.data ?? []).map((a, idx) => (
                  <tr key={idx}>
                    <td className="py-2 pr-4 text-white/80">
                      {a.window_start} → {a.window_end}
                    </td>
                    <td className="py-2 pr-4 text-white/60">{a.service_id}</td>
                    <td className="py-2 pr-4">{a.invocations ?? "-"}</td>
                    <td className="py-2 pr-4">{a.duration_ms ?? "-"}</td>
                    <td className="py-2 pr-4">{a.cost ?? "-"}</td>
                  </tr>
                ))}
                {(aggsQ.data?.data ?? []).length === 0 ? (
                  <tr>
                    <td className="py-3 text-white/60" colSpan={5}>
                      No aggregates for this period/filter.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="text-sm text-white/70">Last bill response</div>
        <pre className="mt-3 max-h-80 overflow-auto rounded-xl border border-white/10 bg-zinc-950 p-3 text-xs text-white/80">
{lastBill ? JSON.stringify(lastBill, null, 2) : "No bill yet. Click Calculate or Generate bill."}
        </pre>
      </section>
    </div>
  )
}
