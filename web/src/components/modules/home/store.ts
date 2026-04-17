'use client';

import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type RankSortMode = 'cost' | 'count' | 'tokens' | 'key-usage';
export type ChartMetricType = 'cost' | 'count' | 'tokens' | 'success-rate';
export type ChartPeriod = '1' | '7' | '30';

const RANK_SORT_MODES: readonly RankSortMode[] = ['cost', 'count', 'tokens', 'key-usage'];
const CHART_METRIC_TYPES: readonly ChartMetricType[] = ['cost', 'count', 'tokens', 'success-rate'];
const CHART_PERIODS: readonly ChartPeriod[] = ['1', '7', '30'];

function normalizeRankSortMode(value: string | null | undefined): RankSortMode {
    return RANK_SORT_MODES.includes(value as RankSortMode) ? (value as RankSortMode) : 'cost';
}

function normalizeChartMetricType(value: string | null | undefined): ChartMetricType {
    return CHART_METRIC_TYPES.includes(value as ChartMetricType) ? (value as ChartMetricType) : 'cost';
}

function normalizeChartPeriod(value: string | null | undefined): ChartPeriod {
    return CHART_PERIODS.includes(value as ChartPeriod) ? (value as ChartPeriod) : '1';
}

interface HomeViewState {
    rankSortMode: RankSortMode;
    chartMetricType: ChartMetricType;
    chartPeriod: ChartPeriod;
    setRankSortMode: (value: RankSortMode) => void;
    setChartMetricType: (value: ChartMetricType) => void;
    setChartPeriod: (value: ChartPeriod) => void;
}

export const useHomeViewStore = create<HomeViewState>()(
    persist(
        (set) => ({
            rankSortMode: 'cost',
            chartMetricType: 'cost',
            chartPeriod: '1',
            setRankSortMode: (value) => set({ rankSortMode: normalizeRankSortMode(value) }),
            setChartMetricType: (value) => set({ chartMetricType: normalizeChartMetricType(value) }),
            setChartPeriod: (value) => set({ chartPeriod: normalizeChartPeriod(value) }),
        }),
        {
            name: 'home-view-options-storage',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                rankSortMode: state.rankSortMode,
                chartMetricType: state.chartMetricType,
                chartPeriod: state.chartPeriod,
            }),
            merge: (persistedState, currentState) => {
                const typed = (persistedState as Partial<HomeViewState> | null) ?? null;
                return {
                    ...currentState,
                    ...typed,
                    rankSortMode: normalizeRankSortMode(typed?.rankSortMode),
                    chartMetricType: normalizeChartMetricType(typed?.chartMetricType),
                    chartPeriod: normalizeChartPeriod(typed?.chartPeriod),
                };
            },
        }
    )
);
