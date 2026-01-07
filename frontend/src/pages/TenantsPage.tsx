import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createTenant, listTenants } from "@/lib/api"
import type { Tenant } from "@/lib/api"

import { useForm } from "react-hook-form"

type FormValues = {
  name: string
}

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
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold">Tenants</h1>
          <p className="mt-1 text-sm text-white/60">
            Создай tenant, чтобы привязать сервисы и биллинг.
          </p>
        </div>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <form
          className="flex flex-col gap-3 md:flex-row"
          onSubmit={handleSubmit((v) => createM.mutate({ name: v.name }))}
        >
          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
            placeholder="Tenant name"
            {...register("name", { required: true })}
          />
          <button
            type="submit"
            className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black hover:bg-zinc-200 disabled:opacity-50"
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

      <section className="rounded-2xl border border-white/10 bg-black/20">
        <div className="border-b border-white/10 px-4 py-3 text-sm text-white/70">
          Tenants list
        </div>
        <div className="p-2">
          {tenantsQ.isLoading ? (
            <div className="p-4 text-sm text-white/60">Loading...</div>
          ) : tenantsQ.isError ? (
            <div className="p-4 text-sm text-red-400">
              Error: {(tenantsQ.error as any)?.message ?? "failed"}
            </div>
          ) : (
            <div className="divide-y divide-white/10">
              {tenantsQ.data!.map((t) => (
                <div key={t.id} className="flex items-center justify-between px-3 py-3">
                  <div>
                    <div className="text-sm font-medium">{t.name}</div>
                    <div className="text-xs text-white/50">{t.id}</div>
                  </div>
                </div>
              ))}
              {tenantsQ.data!.length === 0 ? (
                <div className="p-4 text-sm text-white/60">No tenants yet.</div>
              ) : null}
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
