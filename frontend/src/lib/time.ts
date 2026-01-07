export function toRFC3339(d: Date) {
    return d.toISOString()
  }
  
  export function startOfDay(d = new Date()) {
    const x = new Date(d)
    x.setHours(0, 0, 0, 0)
    return x
  }
  
  export function endOfDay(d = new Date()) {
    const x = new Date(d)
    x.setHours(23, 59, 59, 999)
    return x
  }
  