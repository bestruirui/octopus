'use client';

import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type RankSortMode = 'cost' | 'count' | 'tokens';
export type ChartMetricType = 'cost' | 'count' | 'tokens';
export type ChartPeriod = '1' | '7' | '30';

export type DateRange = {
    from: string | null; // YYYYMMDD format
    to: string | null;   // YYYYMMDD format
};

interface HomeViewState {
    rankSortMode: RankSortMode;
    chartMetricType: ChartMetricType;
    chartPeriod: ChartPeriod;
    dateRange: DateRange;
    setRankSortMode: (value: RankSortMode) => void;
    setChartMetricType: (value: ChartMetricType) => void;
    setChartPeriod: (value: ChartPeriod) => void;
    setDateRange: (value: DateRange) => void;
}

export const useHomeViewStore = create<HomeViewState>()(
    persist(
        (set) => ({
            rankSortMode: 'cost',
            chartMetricType: 'cost',
            chartPeriod: '1',
            dateRange: { from: null, to: null },
            setRankSortMode: (value) => set({ rankSortMode: value }),
            setChartMetricType: (value) => set({ chartMetricType: value }),
            setChartPeriod: (value) => set({ chartPeriod: value }),
            setDateRange: (value) => set({ dateRange: value }),
        }),
        {
            name: 'home-view-options-storage',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                rankSortMode: state.rankSortMode,
                chartMetricType: state.chartMetricType,
                chartPeriod: state.chartPeriod,
            }),
        }
    )
);
