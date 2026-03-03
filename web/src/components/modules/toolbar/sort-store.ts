import { create } from 'zustand';
import type { NavItem } from '@/components/modules/navbar';

export type SortField = 'id' | 'name';
export type SortOrder = 'asc' | 'desc';

interface SortState {
    sortFields: Partial<Record<NavItem, SortField>>;
    sortOrders: Partial<Record<NavItem, SortOrder>>;
    getSortField: (page: NavItem) => SortField;
    getSortOrder: (page: NavItem) => SortOrder;
    setSortField: (page: NavItem, field: SortField) => void;
    setSortOrder: (page: NavItem, order: SortOrder) => void;
    toggleSortOrder: (page: NavItem) => void;
}

export const useSortStore = create<SortState>((set, get) => ({
    sortFields: {},
    sortOrders: {},
    getSortField: (page) => get().sortFields[page] ?? 'id',
    getSortOrder: (page) => get().sortOrders[page] ?? 'asc',
    setSortField: (page, field) =>
        set((state) => ({
            sortFields: { ...state.sortFields, [page]: field },
        })),
    setSortOrder: (page, order) =>
        set((state) => ({
            sortOrders: { ...state.sortOrders, [page]: order },
        })),
    toggleSortOrder: (page) =>
        set((state) => ({
            sortOrders: {
                ...state.sortOrders,
                [page]: (state.sortOrders[page] ?? 'asc') === 'asc' ? 'desc' : 'asc',
            },
        })),
}));
