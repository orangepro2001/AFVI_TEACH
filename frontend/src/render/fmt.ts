// printf-style number formatting for the %d / %.Nf / %.Ng formats the Go side passes.
export function fmtNum(v: number, fmt?: string): string {
  if (typeof v !== 'number' || !isFinite(v)) return '—'
  if (!fmt) return String(v)
  const m = /%(?:\.(\d+))?([dfgs])/.exec(fmt)
  if (!m) return String(v)
  let s: string
  if (m[2] === 'd') s = String(Math.round(v))
  else if (m[2] === 'g') s = String(Number(v.toPrecision(m[1] ? +m[1] : 6)))
  else s = v.toFixed(m[1] ? +m[1] : 6)
  return fmt.replace(m[0], s)
}
