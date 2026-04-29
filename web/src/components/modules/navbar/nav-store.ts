import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { RouteId } from '@/route/config'
import { DEFAULT_NAV_ORDER, normalizeNavOrder } from './nav-order'

export type NavItem = RouteId

const DEFAULT_ACTIVE_ITEM: NavItem = DEFAULT_NAV_ORDER[0]
const VALID_NAV_ITEMS = new Set<NavItem>(DEFAULT_NAV_ORDER)

function normalizeStoreNavOrder(input: Iterable<string> | null | undefined): NavItem[] {
    return normalizeNavOrder(input, DEFAULT_NAV_ORDER) as NavItem[]
}

function isSameNavOrder(left: readonly NavItem[], right: readonly NavItem[]): boolean {
    return left.length === right.length && left.every((item, index) => item === right[index])
}

function getDirection(navOrder: readonly NavItem[], activeItem: NavItem, nextItem: NavItem): number {
    if (activeItem === nextItem) {
        return 0
    }

    const currentIndex = navOrder.indexOf(activeItem)
    const nextIndex = navOrder.indexOf(nextItem)
    if (currentIndex === -1 || nextIndex === -1) {
        return 0
    }

    return nextIndex > currentIndex ? 1 : -1
}

interface NavState {
    activeItem: NavItem
    prevItem: NavItem | null
    direction: number
    navOrder: NavItem[]
    setActiveItem: (item: NavItem) => void
    setNavOrder: (order: Iterable<string> | null | undefined) => void
    resetNavOrder: () => void
}

export const useNavStore = create<NavState>()(
    persist(
        (set, get) => ({
            activeItem: DEFAULT_ACTIVE_ITEM,
            prevItem: null,
            direction: 0,
            navOrder: [...DEFAULT_NAV_ORDER],
            setActiveItem: (item) => {
                const { activeItem, navOrder, prevItem } = get()
                const direction = getDirection(navOrder, activeItem, item)

                set({
                    activeItem: item,
                    prevItem: activeItem === item ? prevItem : activeItem,
                    direction
                })
            },
            setNavOrder: (order) => {
                const nextOrder = normalizeStoreNavOrder(order)
                const { navOrder } = get()
                if (isSameNavOrder(navOrder, nextOrder)) {
                    return
                }

                set({ navOrder: nextOrder })
            },
            resetNavOrder: () => {
                const defaultOrder = [...DEFAULT_NAV_ORDER] as NavItem[]
                const { navOrder } = get()
                if (isSameNavOrder(navOrder, defaultOrder)) {
                    return
                }

                set({ navOrder: defaultOrder })
            },
        }),
        {
            name: 'nav-storage',
            partialize: (state) => ({
                activeItem: state.activeItem,
            }),
            merge: (persistedState, currentState) => {
                const typed = (persistedState as Partial<NavState> | null) ?? null
                const persistedActiveItem = typed?.activeItem

                return {
                    ...currentState,
                    activeItem: persistedActiveItem && VALID_NAV_ITEMS.has(persistedActiveItem)
                        ? persistedActiveItem
                        : currentState.activeItem,
                }
            },
        }
    )
)
