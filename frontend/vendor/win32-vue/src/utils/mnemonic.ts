/**
 * Win32 mnemonic parsing.
 *
 * Win32 labels mark the accelerator key with '&' (e.g. "&Save" shows "Save"
 * with an underlined "S"). "&&" renders a literal ampersand.
 */

export interface MnemonicChar {
  ch: string
  mnemonic: boolean
}

export function splitMnemonic(input: string): MnemonicChar[] {
  const out: MnemonicChar[] = []
  const src = input ?? ''
  for (let i = 0; i < src.length; i++) {
    const ch = src[i]
    if (ch === '&') {
      const next = src[i + 1]
      if (next === '&') {
        out.push({ ch: '&', mnemonic: false })
        i++
        continue
      }
      if (next !== undefined) {
        out.push({ ch: next, mnemonic: true })
        i++
        continue
      }
      out.push({ ch: '&', mnemonic: false })
      continue
    }
    out.push({ ch, mnemonic: false })
  }
  return out
}

/** Plain text with mnemonics stripped — used for aria-labels and titles. */
export function stripMnemonic(input: string): string {
  return splitMnemonic(input)
    .map((c) => c.ch)
    .join('')
}
