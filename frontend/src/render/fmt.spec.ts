import { describe, expect, it } from 'vitest'
import { fmtNum } from './fmt'

describe('fmtNum', () => {
  it('formats the printf shapes the Go side sends', () => {
    expect(fmtNum(1.2345, '%.2f')).toBe('1.23')
    expect(fmtNum(7.6, '%d')).toBe('8')
    expect(fmtNum(0.5, 'GV %.1f')).toBe('GV 0.5')
    expect(fmtNum(1234.5678, '%.6g')).toBe('1234.57')
  })
  it('renders non-finite values as an em dash and no format as plain text', () => {
    expect(fmtNum(NaN, '%.2f')).toBe('—')
    expect(fmtNum(3)).toBe('3')
  })
})
