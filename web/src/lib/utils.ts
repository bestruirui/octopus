import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}


export function formatCount(num: number | undefined): { value: string, unit: string } {
  if (num === undefined) return { value: '-', unit: '' };
  if (num >= 1000000) {
    return { value: (num / 1000000).toFixed(2), unit: 'M' };
  }
  if (num >= 1000) {
    return { value: (num / 1000).toFixed(2), unit: 'K' };
  }
  return { value: num.toFixed(2), unit: '' };
}
export function formatMoney(num: number | undefined): { value: string, unit: string } {
  if (num === undefined) return { value: '-', unit: '' };
  if (num >= 1000000) {
    return { value: (num / 1000000).toFixed(2), unit: 'M$' };
  }
  if (num >= 1000) {
    return { value: (num / 1000).toFixed(2), unit: 'K$' };
  }
  return { value: num.toFixed(2), unit: '$' };
}

export function formatTime(ms: number | undefined): { value: string, unit: string } {
  if (ms === undefined) return { value: '-', unit: '' };
  if (ms < 1000) return { value: ms.toString(), unit: 'ms' };
  const s = ms / 1000;
  if (s < 60) return { value: s.toFixed(2), unit: 's' };
  const m = s / 60;
  if (m < 60) return { value: m.toFixed(2), unit: 'm' };
  const h = m / 60;
  if (h < 24) return { value: h.toFixed(2), unit: 'h' };
  const d = h / 24;
  return { value: d.toFixed(2), unit: 'd' };
}