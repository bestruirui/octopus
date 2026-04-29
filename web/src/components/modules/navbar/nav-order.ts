export const DEFAULT_NAV_ORDER = [
    'home',
    'channel',
    'group',
    'model',
    'analytics',
    'log',
    'alert',
    'ops',
    'setting',
    'user',
] as const;

export type NavOrderItem = (typeof DEFAULT_NAV_ORDER)[number];

export function normalizeNavOrder(
    input: Iterable<string> | null | undefined,
    defaults: readonly string[] = DEFAULT_NAV_ORDER,
): string[] {
    const defaultOrder = Array.from(defaults);
    const allowed = new Set(defaultOrder);
    const seen = new Set<string>();
    const normalized: string[] = [];

    if (input) {
        for (const item of input) {
            const normalizedItem = item.trim();
            if (!normalizedItem || !allowed.has(normalizedItem) || seen.has(normalizedItem)) {
                continue;
            }

            seen.add(normalizedItem);
            normalized.push(normalizedItem);
        }
    }

    for (const item of defaultOrder) {
        if (seen.has(item)) {
            continue;
        }

        normalized.push(item);
    }

    return normalized;
}

export function parseNavOrder(
    value: string | null | undefined,
    defaults: readonly string[] = DEFAULT_NAV_ORDER,
): string[] {
    if (!value) {
        return [...defaults];
    }

    try {
        const parsed = JSON.parse(value);
        if (!Array.isArray(parsed)) {
            return [...defaults];
        }

        const items = parsed.filter((item): item is string => typeof item === 'string');
        return normalizeNavOrder(items, defaults);
    } catch {
        return [...defaults];
    }
}

export function serializeNavOrder(
    input: Iterable<string> | null | undefined,
    defaults: readonly string[] = DEFAULT_NAV_ORDER,
): string {
    return JSON.stringify(normalizeNavOrder(input, defaults));
}
