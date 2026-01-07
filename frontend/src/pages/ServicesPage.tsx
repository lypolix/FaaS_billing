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
      // “Загрузка файла” пока локальная: запомним имя артефакта по serviceId
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
      <header>
        <h1 className="text-xl font-semibold">Services (Functions)</h1>
        <p className="mt-1 text-sm text-white/60">
          Сейчас “функция” = Service в бэке. Файл zip не отправляется, потому что у бэка нет upload endpoint.
        </p>
      </header>

      <section className="rounded-2xl border border-white/10 bg-black/20 p-4">
        <div className="mb-3 text-sm text-white/70">Upload function (stub)</div>
        <form
          className="grid grid-cols-1 gap-3 md:grid-cols-2"
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
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
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
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20"
            placeholder="Function name"
            {...register("name", { required: true })}
          />

          <input
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20 md:col-span-2"
            placeholder="Description (optional)"
            {...register("description")}
          />

          <input
            type="file"
            accept=".zip,.tar,.tar.gz"
            className="w-full rounded-xl border border-white/10 bg-zinc-950 px-3 py-2 text-sm outline-none focus:border-white/20 md:col-span-2"
            {...register("file")}
          />

          <div className="md:col-span-2 flex gap-2">
            <button
              type="submit"
              className="rounded-xl bg-white px-4 py-2 text-sm font-medium text-black hover:bg-zinc-200 disabled:opacity-50"
              disabled={createM.isPending}
            >
              Create service
            </button>
          </div>
        </form>

        {createM.isError ? (
          <p className="mt-3 text-sm text-red-400">
            Error: {(createM.error as any)?.message ?? "failed"}
          </p>
        ) : null}
      </section>

      <section className="rounded-2xl border border-white/10 bg-black/20">
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
          <div className="text-sm text-white/70">Services list</div>
          <div className="text-xs text-white/50">
            Filter: {selectedTenantId ? selectedTenantId : "all"}
          </div>
        </div>

        <div className="p-2">
          {servicesQ.isLoading ? (
            <div className="p-4 text-sm text-white/60">Loading...</div>
          ) : servicesQ.isError ? (
            <div className="p-4 text-sm text-red-400">
              Error: {(servicesQ.error as any)?.message ?? "failed"}
            </div>
          ) : (
            <div className="divide-y divide-white/10">
              {filtered.map((s) => (
                <div key={s.id} className="px-3 py-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium">{s.name}</div>
                      <div className="text-xs text-white/50">tenant: {s.tenant_id}</div>
                      <div className="text-xs text-white/50">id: {s.id}</div>
                      {localFiles[s.id] ? (
                        <div className="mt-1 text-xs text-emerald-300/80">
                          local artifact: {localFiles[s.id]}
                        </div>
                      ) : null}
                    </div>
                  </div>
                  {s.description ? (
                    <div className="mt-2 text-sm text-white/70">{s.description}</div>
                  ) : null}
                </div>
              ))}
              {filtered.length === 0 ? (
                <div className="p-4 text-sm text-white/60">No services for this filter.</div>
              ) : null}
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
