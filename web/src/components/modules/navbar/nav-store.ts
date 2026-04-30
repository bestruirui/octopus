import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type NavItem =
    | 'home'
    | 'channel'
    | 'group'
    | 'model'
    | 'analytics'
    | 'log'
    | 'alert'
    | 'ops'
    | 'setting'
    | 'user';

export const DEFAULT_NAV_ORDER: NavItem[] = [
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
];
export const MIN_VISIBLE_NAV_ITEMS = 5;
export const FIXED_VISIBLE_NAV_ITEMS: NavItem[] = ['setting'];

function isNavItem(value: unknown): value is NavItem {
    return typeof value === 'string' && DEFAULT_NAV_ORDER.includes(value as NavItem);
}

function uniqueNavItems(items: readonly NavItem[]): NavItem[] {
    return Array.from(new Set(items));
}

function getOrderedVisibleItems(orderedItems: readonly NavItem[], visibleItems: readonly NavItem[]): NavItem[] {
    const visibleSet = new Set(visibleItems);
    return orderedItems.filter((item) => visibleSet.has(item));
}

function getFallbackActiveItem(orderedItems: readonly NavItem[], visibleItems: readonly NavItem[]): NavItem {
    return getOrderedVisibleItems(orderedItems, visibleItems)[0] ?? 'setting';
}

function getDirection(
    currentItem: NavItem,
    nextItem: NavItem,
    orderedItems: readonly NavItem[],
    visibleItems: readonly NavItem[],
): number {
    const navigationItems = getOrderedVisibleItems(orderedItems, visibleItems);
    const currentIndex = navigationItems.indexOf(currentItem);
    const nextIndex = navigationItems.indexOf(nextItem);

    if (currentIndex === -1 || nextIndex === -1 || currentIndex === nextIndex) {
        return 0;
    }

    return nextIndex > currentIndex ? 1 : -1;
}

export function normalizeNavOrder(value: unknown): NavItem[] {
    const items = Array.isArray(value) ? uniqueNavItems(value.filter(isNavItem)) : [];
    const missingItems = DEFAULT_NAV_ORDER.filter((item) => !items.includes(item));
    return [...items, ...missingItems];
}

export function normalizeVisibleNavItems(value: unknown, orderedItems: readonly NavItem[]): NavItem[] {
    const requested = Array.isArray(value)
        ? uniqueNavItems(value.filter(isNavItem))
        : [...DEFAULT_NAV_ORDER];
    const withFixedItems = uniqueNavItems([...requested, ...FIXED_VISIBLE_NAV_ITEMS]);
    const orderedVisibleItems = getOrderedVisibleItems(orderedItems, withFixedItems);

    if (orderedVisibleItems.length >= MIN_VISIBLE_NAV_ITEMS) {
        return orderedVisibleItems;
    }

    const missingItems = orderedItems.filter((item) => !orderedVisibleItems.includes(item));
    return [...orderedVisibleItems, ...missingItems.slice(0, MIN_VISIBLE_NAV_ITEMS - orderedVisibleItems.length)];
}

export function isFixedVisibleNavItem(item: NavItem): boolean {
    return FIXED_VISIBLE_NAV_ITEMS.includes(item);
}

interface NavState {
    activeItem: NavItem;
    prevItem: NavItem | null;
    direction: number;
    orderedItems: NavItem[];
    visibleItems: NavItem[];
    setActiveItem: (item: NavItem) => void;
    setNavOrder: (items: NavItem[]) => void;
    setOrderedItems: (items: NavItem[]) => void;
    setItemVisible: (item: NavItem, visible: boolean) => void;
    resetNavOrder: () => void;
    resetPreferences: () => void;
}

export const useNavStore = create<NavState>()(
    persist(
        (set, get) => ({
            activeItem: 'home',
            prevItem: null,
            direction: 0,
            orderedItems: [...DEFAULT_NAV_ORDER],
            visibleItems: [...DEFAULT_NAV_ORDER],
            setActiveItem: (item) => {
                const { activeItem, orderedItems, visibleItems } = get();
                if (item === activeItem || !visibleItems.includes(item)) {
                    return;
                }

                set({
                    activeItem: item,
                    prevItem: activeItem,
                    direction: getDirection(activeItem, item, orderedItems, visibleItems),
                });
            },
            setOrderedItems: (items) => {
                set((state) => {
                    const orderedItems = normalizeNavOrder(items);
                    const visibleItems = normalizeVisibleNavItems(state.visibleItems, orderedItems);
                    const activeItem = visibleItems.includes(state.activeItem)
                        ? state.activeItem
                        : getFallbackActiveItem(orderedItems, visibleItems);

                    return {
                        orderedItems,
                        visibleItems,
                        activeItem,
                        prevItem: activeItem === state.activeItem ? state.prevItem : state.activeItem,
                        direction: activeItem === state.activeItem
                            ? state.direction
                            : getDirection(state.activeItem, activeItem, orderedItems, visibleItems),
                    };
                });
            },
            setNavOrder: (items) => {
                get().setOrderedItems(items);
            },
            setItemVisible: (item, visible) => {
                set((state) => {
                    if (visible === state.visibleItems.includes(item)) {
                        return state;
                    }
                    if (!visible && isFixedVisibleNavItem(item)) {
                        return state;
                    }
                    if (!visible && state.visibleItems.length <= MIN_VISIBLE_NAV_ITEMS) {
                        return state;
                    }

                    const requestedVisibleItems = visible
                        ? [...state.visibleItems, item]
                        : state.visibleItems.filter((visibleItem) => visibleItem !== item);
                    const visibleItems = normalizeVisibleNavItems(requestedVisibleItems, state.orderedItems);
                    const activeItem = visibleItems.includes(state.activeItem)
                        ? state.activeItem
                        : getFallbackActiveItem(state.orderedItems, visibleItems);

                    return {
                        visibleItems,
                        activeItem,
                        prevItem: activeItem === state.activeItem ? state.prevItem : state.activeItem,
                        direction: activeItem === state.activeItem
                            ? state.direction
                            : getDirection(state.activeItem, activeItem, state.orderedItems, visibleItems),
                    };
                });
            },
            resetNavOrder: () => {
                get().resetPreferences();
            },
            resetPreferences: () => {
                set((state) => {
                    const orderedItems = [...DEFAULT_NAV_ORDER];
                    const visibleItems = [...DEFAULT_NAV_ORDER];
                    const activeItem = visibleItems.includes(state.activeItem)
                        ? state.activeItem
                        : getFallbackActiveItem(orderedItems, visibleItems);

                    return {
                        orderedItems,
                        visibleItems,
                        activeItem,
                        prevItem: activeItem === state.activeItem ? state.prevItem : state.activeItem,
                        direction: activeItem === state.activeItem
                            ? state.direction
                            : getDirection(state.activeItem, activeItem, orderedItems, visibleItems),
                    };
                });
            },
        }),
        {
            name: 'nav-storage',
            storage: createJSONStorage(() => localStorage),
            merge: (persistedState, currentState) => {
                const typed = (persistedState as Partial<NavState> | null) ?? null;
                const orderedItems = normalizeNavOrder(typed?.orderedItems);
                const visibleItems = normalizeVisibleNavItems(typed?.visibleItems, orderedItems);
                const activeItem = isNavItem(typed?.activeItem) && visibleItems.includes(typed.activeItem)
                    ? typed.activeItem
                    : getFallbackActiveItem(orderedItems, visibleItems);
                const prevItem = isNavItem(typed?.prevItem) ? typed.prevItem : null;

                return {
                    ...currentState,
                    ...typed,
                    activeItem,
                    prevItem,
                    direction: typeof typed?.direction === 'number' ? typed.direction : 0,
                    orderedItems,
                    visibleItems,
                };
            },
        }
    )
);
