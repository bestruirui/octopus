import { create } from 'zustand';
import type { ToolbarPage } from './view-options-store';

interface SearchState {
    searchTerms: Partial<Record<ToolbarPage, string>>;
    getSearchTerm: (page: ToolbarPage) => string;
    setSearchTerm: (page: ToolbarPage, term: string) => void;
}

export const useSearchStore = create<SearchState>((set, get) => ({
    searchTerms: {},
    getSearchTerm: (page) => get().searchTerms[page] || '',
    setSearchTerm: (page, term) => set((state) => ({
        searchTerms: { ...state.searchTerms, [page]: term }
    })),
}));
