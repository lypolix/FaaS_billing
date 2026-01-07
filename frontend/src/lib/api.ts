import axios from "axios"

export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1"

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { "Content-Type": "application/json" },
})

export type UUID = string

export type Tenant = {
  id: UUID
  name: string
  created_at?: string
  updated_at?: string
}

export type Service = {
  id: UUID
  tenant_id: UUID
  name: string
  description?: string
  created_at?: string
  updated_at?: string
}

export type UsageAggregate = {
  id?: UUID
  tenant_id: UUID
  service_id: UUID
  window_start: string
  window_end: string
  invocations?: number
  duration_ms?: number
  cost?: number
}

export type UsageAggregatesResponse = {
  data: UsageAggregate[]
}

export type CalculateBillRequest = {
  tenant_id: UUID
  start_time: string // RFC3339
  end_time: string // RFC3339
}

export async function health() {
  const { data } = await api.get("/health")
  return data as { status: string }
}

export async function listTenants() {
  const { data } = await api.get("/tenants")
  return data as Tenant[]
}

export async function createTenant(payload: Partial<Tenant>) {
  const { data } = await api.post("/tenants", payload)
  return data as Tenant
}

export async function listServices(params?: { tenant_id?: string }) {
  const { data } = await api.get("/services", { params })
  return data as Service[]
}

export async function createService(payload: Partial<Service>) {
  const { data } = await api.post("/services", payload)
  return data as Service
}

export async function getUsageAggregates(params?: {
  tenant_id?: string
  service_id?: string
  start_time?: string
  end_time?: string
}) {
  const { data } = await api.get("/usage-aggregates", { params })
  return data as UsageAggregatesResponse
}

export async function calculateBill(payload: CalculateBillRequest) {
  const { data } = await api.post("/billing/calculate", payload)
  return data
}

export async function generateBill(payload: CalculateBillRequest) {
  const { data } = await api.post("/billing/generate", payload)
  return data
}

export async function forecastCost(payload: unknown) {
  const { data } = await api.post("/forecast/cost", payload)
  return data
}
