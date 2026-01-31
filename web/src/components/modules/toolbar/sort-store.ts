import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { NavItem } from '@/components/modules/navbar';

/**
 * 排序字段类型
 * - id: 按 ID 排序（默认）
 * - name: 按名称排序
 */
export type SortField = 'id' | 'name';

/**
 * 排序方向
 * - asc: 正序
 * - desc: 倒序
 */
export type SortOrder = 'asc' | 'desc';

export interface SortConfig {
    field: SortField;
    order: SortOrder;
}

interface SortState {
    sortConfigs: Partial<Record<NavItem, SortConfig>>;
    getSortConfig: (page: NavItem) => SortConfig;
    setSortConfig: (page: NavItem, config: SortConfig) => void;
    toggleSortOrder: (page: NavItem) => void;
}

const DEFAULT_SORT_CONFIG: SortConfig = { field: 'id', order: 'asc' };

export const useSortStore = create<SortState>()(
    persist(
        (set, get) => ({
            sortConfigs: {},
            getSortConfig: (page) => get().sortConfigs[page] || DEFAULT_SORT_CONFIG,
            setSortConfig: (page, config) => set((state) => ({
                sortConfigs: { ...state.sortConfigs, [page]: config }
            })),
            toggleSortOrder: (page) => {
                const current = get().getSortConfig(page);
                set((state) => ({
                    sortConfigs: {
                        ...state.sortConfigs,
                        [page]: { ...current, order: current.order === 'asc' ? 'desc' : 'asc' }
                    }
                }));
            },
        }),
        {
            name: 'sort-storage',
        }
    )
);
