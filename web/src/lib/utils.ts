import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}


const toTwoDecimal = (value: number): number => Number(value.toFixed(2));

function formatNumber(num: number | undefined, compare: number[], units: string[]): { value: number, unit: string } {
  if (num === undefined) return { value: 0, unit: units[0] };
  else if (num >= compare[0]) return { value: toTwoDecimal(num / compare[0]), unit: units[1] };
  else if (num >= compare[1]) return { value: toTwoDecimal(num / compare[1]), unit: units[2] };
  else if (num >= compare[2]) return { value: toTwoDecimal(num / compare[2]), unit: units[3] };
  else if (num >= compare[3]) return { value: toTwoDecimal(num / compare[3]), unit: units[4] };
  else return { value: toTwoDecimal(num), unit: units[5] };
}

export function formatCount(num: number | undefined): { raw: number, formatted: { value: number, unit: string } } {
  return {
    raw: num ?? 0,
    formatted: formatNumber(num, [1000000000, 1000000, 1000, 1], ['', 'B', 'M', 'K', '', '']),
  };
}
export function formatMoney(num: number | undefined): { raw: number, formatted: { value: number, unit: string } } {
  return {
    raw: num ?? 0,
    formatted: formatNumber(num, [1000000000, 1000000, 1000, 1], ['$', 'B$', 'M$', 'K$', '$', '$']),
  };
}

export function formatTime(ms: number | undefined): { raw: number, formatted: { value: number, unit: string } } {
  return {
    raw: ms ?? 0,
    formatted: formatNumber(ms, [86400000, 3600000, 60000, 1000], ['', 'd', 'h', 'm', 's', 'ms']),
  };
}