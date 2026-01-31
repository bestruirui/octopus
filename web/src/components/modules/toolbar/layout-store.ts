import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { NavItem } from '@/components/modules/navbar';

/**
 * 布局类型
 * - grid: 网格布局（默认，如渠道页面的卡片网格）
 * - single-column: 单列布局（如日志页面）
 */
export type LayoutType = 'grid' | 'single-column';

interface LayoutState {
    layouts: Partial<Record<NavItem, LayoutType>>;
    getLayout: (page: NavItem) => LayoutType;
    setLayout: (page: NavItem, layout: LayoutType) => void;
}

export const useLayoutStore = create<LayoutState>()(
    persist(
        (set, get) => ({
            layouts: {},
            getLayout: (page) => get().layouts[page] || 'grid',
            setLayout: (page, layout) => set((state) => ({
                layouts: { ...state.layouts, [page]: layout }
            })),
        }),
        {
            name: 'layout-storage',
        }
    )
);
