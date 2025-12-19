import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type NavItem = 'home' | 'channel' | 'group' | 'model' | 'log' | 'setting'

const NAV_ORDER: NavItem[] = ['home', 'channel', 'group', 'model', 'log', 'setting']

interface NavState {
    activeItem: NavItem
    prevItem: NavItem | null
    direction: number
    setActiveItem: (item: NavItem) => void
}

export const useNavStore = create<NavState>()(
    persist(
        (set, get) => ({
            activeItem: 'home',
            prevItem: null,
            direction: 0,
            setActiveItem: (item) => {
                const { activeItem } = get()
                const currentIndex = NAV_ORDER.indexOf(activeItem)
                const newIndex = NAV_ORDER.indexOf(item)
                const direction = newIndex > currentIndex ? 1 : -1

                set({
                    activeItem: item,
                    prevItem: activeItem,
                    direction
                })
            },
        }),
        {
            name: 'nav-storage',
        }
    )
)
