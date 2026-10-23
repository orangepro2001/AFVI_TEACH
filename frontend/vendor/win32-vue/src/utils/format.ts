/** Numeric formatting helpers shared by the industrial widgets. */

/**
 * Choose a sensible number of decimals for a dial readout based on the span.
 * A 0-120 bar gauge shows one decimal, a 0-5 V gauge shows two.
 */
export function formatValue(value: number, span: number): string {
  if (!Number.isFinite(value)) return '--'
  if (span <= 5) return value.toFixed(2)
  if (span <= 50) return value.toFixed(1)
  return value.toFixed(0)
}
