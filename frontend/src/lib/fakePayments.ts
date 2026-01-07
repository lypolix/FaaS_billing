const KEY = "faas_billing_fake_payments_v1"

export type FakePayment = {
  billKey: string
  paidAt: string
  amount?: number
}

export function markBillPaid(billKey: string, amount?: number) {
  const existing = readPayments()
  const next: FakePayment[] = [
    ...existing.filter((p) => p.billKey !== billKey),
    { billKey, paidAt: new Date().toISOString(), amount },
  ]
  localStorage.setItem(KEY, JSON.stringify(next))
}

export function isBillPaid(billKey: string) {
  return readPayments().some((p) => p.billKey === billKey)
}

export function readPayments(): FakePayment[] {
  try {
    const raw = localStorage.getItem(KEY)
    return raw ? (JSON.parse(raw) as FakePayment[]) : []
  } catch {
    return []
  }
}
