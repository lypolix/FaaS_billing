import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createService, listServices, listTenants } from "@/lib/api"
import type { Service } from "@/lib/api"
import { useForm } from "react-hook-form"

type UploadForm = {
  tenant_id: string
  name: string
  description?: string
  file?: FileList
}

export default function ServicesPage() {
  const qc = useQueryClient()
  const [localFiles, setLocalFiles] = useState<Record<string, string>>({})

  const tenantsQ = useQuery({ queryKey: ["tenants"], queryFn: listTenants })
  const servicesQ = useQuery({ queryKey: ["services"], queryFn: () => listServices() })

  const { register, handleSubmit, watch, reset } = useForm<UploadForm>({
    defaultValues: { tenant_id: "", name: "", description: "" },
  })

  const selectedTenantId = watch("tenant_id")

  const filtered = useMemo(() => {
    if (!servicesQ.data) return []
    if (!selectedTenantId) return servicesQ.data
    return servicesQ.data.filter((s) => s.tenant_id === selectedTenantId)
  }, [servicesQ.data, selectedTenantId])

  const createM = useMutation({
    mutationFn: (payload: Partial<Service>) => createService(payload),
    onSuccess: async (svc) => {
      reset({ tenant_id: selectedTenantId, name: "", description: "" })
      await qc.invalidateQueries({ queryKey: ["services"] })
      if ((window as any).__lastUploadedFileName) {
        setLocalFiles((prev) => ({
          ...prev,
          [svc.id]: (window as any).__lastUploadedFileName as string,
        }))
        ;(window as any).__lastUploadedFileName = undefined
      }
    },
  })

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Services</h1>
          <p className="mt-1 text-sm text-white/60">
            Сейчас это “функции” в биллинге. Когда добавишь upload endpoint — сюда подключим реальную загрузку zip.
          </p>
        </div>
        <div className="text-xs text-white/50">
          {servicesQ.data ? `${servicesQ.data.length} total` : ""}
        </div>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium">Upload function</div>
            <div className="mt-1 text-xs text-white/50">
              Создаёт Service в бэке, файл пока хранится локально в UI (stub).
            </div>
          </div>
          <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-xs text-white/70">
            Beta
          </span>
        </div>

        <form
          className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2"
          onSubmit={handleSubmit((v) => {
            const f = v.file?.[0]
            if (f) (window as any).__lastUploadedFileName = f.name
            createM.mutate({
              tenant_id: v.tenant_id,
              name: v.name,
              description: v.description,
            })
          })}
        >
          <select
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
            {...register("tenant_id", { required: true })}
          >
            <option value="">Select tenant</option>
            {(tenantsQ.data ?? []).map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>

          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10"
            placeholder="Function name"
            {...register("name", { required: true })}
          />

          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10 md:col-span-2"
            placeholder="Description (optional)"
            {...register("description")}
          />

          <input
            type="file"
            accept=".zip,.tar,.tar.gz"
            className="w-full rounded-xl border border-white/10 bg-zinc-950/60 px-3 py-2 text-sm outline-none transition focus:border-white/20 focus:ring-2 focus:ring-white/10 md:col-span-2"
            {...register("file")}
          />

          <div className="md:col-span-2 flex gap-2">
            <button
              type="submit"
              className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black transition hover:bg-zinc-200 disabled:opacity-50"
              disabled={createM.isPending}
            >
              Create service
            </button>
            <div className="text-xs text-white/50 self-center">
              {selectedTenantId ? `tenant: ${selectedTenantId}` : "pick a tenant"}
            </div>
          </div>
        </form>
      </section>

      <section className="rounded-2xl border border-white/10 bg-black/20">
        <div className="flex flex-col gap-2 border-b border-white/10 px-4 py-3 md:flex-row md:items-center md:justify-between">
          <div className="text-sm font-medium">Services list</div>
          <div className="text-xs text-white/50">
            Showing: {selectedTenantId ? "selected tenant" : "all"} • {filtered.length} items
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs text-white/50">
              <tr className="border-b border-white/10">
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Tenant</th>
                <th className="px-4 py-3">Artifact</th>
                <th className="px-4 py-3">ID</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {servicesQ.isLoading ? (
                <tr>
                  <td className="px-4 py-4 text-white/60" colSpan={4}>
                    Loading...
                  </td>
                </tr>
              ) : servicesQ.isError ? (
                <tr>
                  <td className="px-4 py-4 text-red-400" colSpan={4}>
                    Error: {(servicesQ.error as any)?.message ?? "failed"}
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td className="px-4 py-4 text-white/60" colSpan={4}>
                    No services yet.
                  </td>
                </tr>
              ) : (
                filtered.map((s) => (
                  <tr key={s.id} className="hover:bg-white/[0.03]">
                    <td className="px-4 py-3">
                      <div className="font-medium">{s.name}</div>
                      {s.description ? (
                        <div className="mt-1 text-xs text-white/50">{s.description}</div>
                      ) : null}
                    </td>
                    <td className="px-4 py-3 text-white/70">{s.tenant_id}</td>
                    <td className="px-4 py-3">
                      {localFiles[s.id] ? (
                        <span className="rounded-full border border-emerald-400/20 bg-emerald-400/10 px-2 py-1 text-xs text-emerald-200">
                          {localFiles[s.id]}
                        </span>
                      ) : (
                        <span className="rounded-full border border-white/10 bg-white/5 px-2 py-1 text-xs text-white/60">
                          none
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs text-white/50">{s.id}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
