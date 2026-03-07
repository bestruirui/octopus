import { create } from 'zustand';

export type ToolbarLayout = 'grid' | 'list';
export type ToolbarSortOrder = 'asc' | 'desc';
export const TOOLBAR_PAGES = ['channel', 'group', 'model'] as const;
export type ToolbarPage = (typeof TOOLBAR_PAGES)[number];
export type ChannelFilter = 'all' | 'enabled' | 'disabled';
export type GroupFilter = 'all' | 'with-members' | 'empty';
export type ModelFilter = 'all' | 'priced' | 'free';

interface ToolbarViewOptionsState {
    layouts: Partial<Record<ToolbarPage, ToolbarLayout>>;
    sortOrders: Partial<Record<ToolbarPage, ToolbarSortOrder>>;
    channelFilter: ChannelFilter;
    groupFilter: GroupFilter;
    modelFilter: ModelFilter;

    getLayout: (item: ToolbarPage) => ToolbarLayout;
    setLayout: (item: ToolbarPage, value: ToolbarLayout) => void;

    getSortOrder: (item: ToolbarPage) => ToolbarSortOrder;
    setSortOrder: (item: ToolbarPage, value: ToolbarSortOrder) => void;

    setChannelFilter: (value: ChannelFilter) => void;
    setGroupFilter: (value: GroupFilter) => void;
    setModelFilter: (value: ModelFilter) => void;
}

export const useToolbarViewOptionsStore = create<ToolbarViewOptionsState>((set, get) => ({
    layouts: {},
    sortOrders: {},
    channelFilter: 'all',
    groupFilter: 'all',
    modelFilter: 'all',

    getLayout: (item) => get().layouts[item] || 'grid',
    setLayout: (item, value) => {
        set((state) => ({ layouts: { ...state.layouts, [item]: value } }));
    },

    getSortOrder: (item) => get().sortOrders[item] || 'asc',
    setSortOrder: (item, value) => {
        set((state) => ({ sortOrders: { ...state.sortOrders, [item]: value } }));
    },

    setChannelFilter: (value) => set({ channelFilter: value }),
    setGroupFilter: (value) => set({ groupFilter: value }),
    setModelFilter: (value) => set({ modelFilter: value }),
}));
