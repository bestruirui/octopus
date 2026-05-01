import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type ToolbarLayout = 'grid' | 'list';
export type ToolbarSortOrder = 'asc' | 'desc';
export type ToolbarSortField = 'name' | 'created';
export type ToolbarCreatedSortablePage = 'channel' | 'group';
export const TOOLBAR_PAGES = ['channel', 'group', 'model'] as const;
export type ToolbarPage = (typeof TOOLBAR_PAGES)[number];
export type ChannelFilter = 'all' | 'enabled' | 'disabled';
export type GroupFilter = 'all' | 'with-members' | 'empty' | 'chat' | 'deepseek' | 'mimo' | 'responses' | 'messages' | 'embeddings' | 'rerank' | 'moderations' | 'image_generation' | 'audio_speech' | 'audio_transcription' | 'video_generation' | 'music_generation' | 'search';
export type ModelFilter = 'all' | 'priced' | 'free';
export type ModelSortMode = 'success-rate' | 'request-count';

export function normalizeGroupFilterValue(value?: string | null): GroupFilter {
    switch (value) {
        case 'moderation':
            return 'moderations';
        case 'responses':
        case 'messages':
            return 'chat';
        case 'all':
        case 'with-members':
        case 'empty':
        case 'chat':
        case 'deepseek':
        case 'mimo':
        case 'embeddings':
        case 'rerank':
        case 'moderations':
        case 'image_generation':
        case 'audio_speech':
        case 'audio_transcription':
        case 'video_generation':
        case 'music_generation':
        case 'search':
            return value;
        default:
            return 'all';
    }
}

function normalizePersistedToolbarState(
    state?: Partial<ToolbarViewOptionsState> | null,
): Partial<ToolbarViewOptionsState> {
    if (!state) {
        return {};
    }

    return {
        ...state,
        groupFilter: normalizeGroupFilterValue(state.groupFilter),
    };
}

interface ToolbarViewOptionsState {
    layouts: Partial<Record<ToolbarPage, ToolbarLayout>>;
    sortFields: Partial<Record<ToolbarCreatedSortablePage, ToolbarSortField>>;
    sortOrders: Partial<Record<ToolbarPage, ToolbarSortOrder>>;
    channelFilter: ChannelFilter;
    groupFilter: GroupFilter;
    modelFilter: ModelFilter;
    modelSortMode: ModelSortMode;

    getLayout: (item: ToolbarPage) => ToolbarLayout;
    setLayout: (item: ToolbarPage, value: ToolbarLayout) => void;

    getSortField: (item: ToolbarCreatedSortablePage) => ToolbarSortField;
    setSortConfig: (
        item: ToolbarCreatedSortablePage,
        field: ToolbarSortField,
        order: ToolbarSortOrder
    ) => void;

    getSortOrder: (item: ToolbarPage) => ToolbarSortOrder;
    setSortOrder: (item: ToolbarPage, value: ToolbarSortOrder) => void;

    setChannelFilter: (value: ChannelFilter) => void;
    setGroupFilter: (value: GroupFilter) => void;
    setModelFilter: (value: ModelFilter) => void;
    setModelSortMode: (value: ModelSortMode) => void;
}

export const useToolbarViewOptionsStore = create<ToolbarViewOptionsState>()(
    persist(
        (set, get) => ({
            layouts: {},
            sortFields: {},
            sortOrders: {},
            channelFilter: 'all',
            groupFilter: 'all',
            modelFilter: 'all',
            modelSortMode: 'success-rate',

            getLayout: (item) => get().layouts[item] || 'grid',
            setLayout: (item, value) => {
                set((state) => ({ layouts: { ...state.layouts, [item]: value } }));
            },

            getSortField: (item) => get().sortFields[item] || 'name',
            setSortConfig: (item, field, order) => {
                set((state) => ({
                    sortFields: { ...state.sortFields, [item]: field },
                    sortOrders: { ...state.sortOrders, [item]: order },
                }));
            },

            getSortOrder: (item) => (get().sortOrders[item] === 'desc' ? 'desc' : 'asc'),
            setSortOrder: (item, value) => {
                set((state) => ({ sortOrders: { ...state.sortOrders, [item]: value } }));
            },

            setChannelFilter: (value) => set({ channelFilter: value }),
            setGroupFilter: (value) => set({ groupFilter: normalizeGroupFilterValue(value) }),
            setModelFilter: (value) => set({ modelFilter: value }),
            setModelSortMode: (value) => set({ modelSortMode: value }),
        }),
        {
            name: 'toolbar-view-options-storage',
            partialize: (state) => ({
                layouts: state.layouts,
                sortFields: state.sortFields,
                sortOrders: state.sortOrders,
                channelFilter: state.channelFilter,
                groupFilter: state.groupFilter,
                modelFilter: state.modelFilter,
                modelSortMode: state.modelSortMode,
            }),
            merge: (persistedState, currentState) => ({
                ...currentState,
                ...normalizePersistedToolbarState(persistedState as Partial<ToolbarViewOptionsState> | null),
            }),
        }
    )
);
