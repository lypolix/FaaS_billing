import { useMemo, useState } from "react"
import { useMutation, useQuery } from "@tanstack/react-query"
import { calculateBill, generateBill, getUsageAggregates, listServices, listTenants } from "@/lib/api"
import { endOfDay, startOfDay, toRFC3339 } from "@/lib/time"
import { isBillPaid, markBillPaid } from "@/lib/fakePayments"

export default function BillingPage() {
  const tenantsQ = useQuery({ queryKey: ["tenants"], queryFn: listTenants })
  const servicesQ = useQuery({ queryKey: ["services"], queryFn: () => listServices() })

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
    mutationFn: () => calculateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const genM = useMutation({
    mutationFn: () => generateBill({ tenant_id: tenantId, start_time: start, end_time: end }),
    onSuccess: (data) => setLastBill(data),
  })

  const billKey = useMemo(() => (tenantId ? `${tenantId}:${start}:${end}` : ""), [tenantId, start, end])
  const paid = billKey ? isBillPaid(billKey) : false

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Billing</h1>
          <p className="mt-1 text-sm text-white/60">
            Рассчитай стоимость за период и сгенерируй счёт. Оплата пока фейковая (local).
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs">
          <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-white/70">Tenant-based</span>
          <span className={paid ? "rounded-full border border-emerald-400/20 bg-emerald-400/10 px-2 py-1 text-emerald-200"
                                : "rounded-full border border-yellow-400/20 bg-yellow-400/10 px-2 py-1 text-yellow-200"}>
            {billKey ? (paid ? "PAID (fake)" : "UNPAID") : "Select tenant"}
          </span>
        </div>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <select
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
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
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
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
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
            value={start}
            onChange={(e) => setStart(e.target.value)}
            placeholder="start_time RFC3339"
          />
          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
            value={end}
            onChange={(e) => setEnd(e.target.value)}
            placeholder="end_time RFC3339"
          />

          <div className="md:col-span-2 flex flex-wrap gap-2">
            <button
              className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black transition hover:bg-zinc-200 disabled:opacity-50"
              onClick={() => calcM.mutate()}
              disabled={!tenantId || calcM.isPending}
            >
              Calculate
            </button>
            <button
              className="rounded-xl border border-white/15 bg-transparent px-4 py-2 text-sm font-medium text-white transition hover:bg-white/5 disabled:opacity-50"
              onClick={() => genM.mutate()}
              disabled={!tenantId || genM.isPending}
            >
              Generate bill
            </button>

            <button
              className="ml-auto rounded-xl bg-emerald-400 px-4 py-2 text-sm font-medium text-black transition hover:bg-emerald-300 disabled:opacity-50"
              onClick={() => billKey && markBillPaid(billKey)}
              disabled={!billKey}
            >
              Pay (fake)
            </button>
          </div>

          {billKey ? (
            <div className="md:col-span-2 text-xs text-white/60">
              Bill key: <span className="text-white/80 break-all">{billKey}</span>
            </div>
          ) : null}
        </div>
      </section>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
          <div className="text-sm font-medium">Usage aggregates</div>
          <div className="mt-1 text-xs text-white/50">
            Фактические агрегаты за период (фильтр tenant/service).
          </div>

          <div className="mt-4 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="text-xs text-white/50">
                <tr className="border-b border-white/10">
                  <th className="py-2 pr-4">window</th>
                  <th className="py-2 pr-4">service</th>
                  <th className="py-2 pr-4">inv</th>
                  <th className="py-2 pr-4">ms</th>
                  <th className="py-2 pr-4">cost</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {aggsQ.isFetching ? (
                  <tr>
                    <td className="py-3 text-white/60" colSpan={5}>
                      Loading...
                    </td>
                  </tr>
                ) : aggsQ.isError ? (
                  <tr>
                    <td className="py-3 text-red-400" colSpan={5}>
                      Error: {(aggsQ.error as any)?.message ?? "failed"}
                    </td>
                  </tr>
                ) : (aggsQ.data?.data ?? []).length === 0 ? (
                  <tr>
                    <td className="py-3 text-white/60" colSpan={5}>
                      No aggregates.
                    </td>
                  </tr>
                ) : (
                  (aggsQ.data?.data ?? []).slice(0, 12).map((a, idx) => (
                    <tr key={idx} className="hover:bg-white/[0.03]">
                      <td className="py-2 pr-4 text-xs text-white/70">
                        {a.window_start.split(".")[0]} → {a.window_end.split(".")[0]}
                      </td>
                      <td className="py-2 pr-4 text-xs text-white/60">{a.service_id}</td>
                      <td className="py-2 pr-4">{a.invocations ?? "-"}</td>
                      <td className="py-2 pr-4">{a.duration_ms ?? "-"}</td>
                      <td className="py-2 pr-4">{a.cost ?? "-"}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        <div className="rounded-2xl border border-white/10 bg-black/20 p-4">
          <div className="text-sm font-medium">Last bill response</div>
          <div className="mt-1 text-xs text-white/50">
            Ответ от /billing/calculate или /billing/generate.
          </div>

          <pre className="mt-4 max-h-[420px] overflow-auto rounded-xl border border-white/10 bg-zinc-950/60 p-3 text-xs text-white/80">
{lastBill ? JSON.stringify(lastBill, null, 2) : "No bill yet. Click Calculate or Generate bill."}
          </pre>
        </div>
      </section>
    </div>
  )
}
