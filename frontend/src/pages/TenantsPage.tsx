import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createTenant, listTenants } from "@/lib/api"
import type { Tenant } from "@/lib/api"
import { useForm } from "react-hook-form"

type FormValues = { name: string }

export default function TenantsPage() {
  const qc = useQueryClient()

  const tenantsQ = useQuery({
    queryKey: ["tenants"],
    queryFn: listTenants,
  })

  const { register, handleSubmit, reset } = useForm<FormValues>({
    defaultValues: { name: "" },
  })

  const createM = useMutation({
    mutationFn: (payload: Partial<Tenant>) => createTenant(payload),
    onSuccess: async () => {
      reset()
      await qc.invalidateQueries({ queryKey: ["tenants"] })
    },
  })

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Tenants</h1>
          <p className="mt-1 text-sm text-white/60">
            Аккаунты/команды, к которым привязаны функции (services) и биллинг.
          </p>
        </div>
        <div className="text-xs text-white/50">
          {tenantsQ.data ? `${tenantsQ.data.length} total` : ""}
        </div>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="mb-3 text-sm text-white/70">Create tenant</div>
        <form
          className="flex flex-col gap-3 md:flex-row"
          onSubmit={handleSubmit((v) => createM.mutate({ name: v.name }))}
        >
          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
            placeholder="Tenant name (e.g. Acme Inc.)"
            {...register("name", { required: true })}
          />
          <button
            type="submit"
            className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black transition hover:bg-zinc-200 disabled:opacity-50"
            disabled={createM.isPending}
          >
            Create
          </button>
        </form>

        {createM.isError ? (
          <p className="mt-3 text-sm text-red-400">
            Error: {(createM.error as any)?.message ?? "failed"}
          </p>
        ) : null}
      </section>

      <section className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {tenantsQ.isLoading ? (
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-white/60">
            Loading...
          </div>
        ) : tenantsQ.isError ? (
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-red-400">
            Error: {(tenantsQ.error as any)?.message ?? "failed"}
          </div>
        ) : tenantsQ.data!.length === 0 ? (
          <div className="rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-white/60">
            No tenants yet.
          </div>
        ) : (
          tenantsQ.data!.map((t) => (
            <div
              key={t.id}
              className="rounded-2xl border border-white/10 bg-black/20 p-4 transition hover:border-white/20"
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-sm font-medium">{t.name}</div>
                  <div className="mt-1 text-xs text-white/50 break-all">{t.id}</div>
                </div>
                <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-xs text-white/70">
                  Tenant
                </span>
              </div>
            </div>
          ))
        )}
      </section>
    </div>
  )
}
