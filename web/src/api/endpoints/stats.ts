import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';

/**
 * 统计数据
 */
export interface StatsDaily {
    date: string; // ISO 8601 格式
    input_token: number;
    output_token: number;
    request_count: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
}
export interface StatsTotal {
    id: number;
    input_token: number;
    output_token: number;
    request_count: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
}
/**
 * 获取今日统计数据 Hook
 */
export function useStatsToday() {
    return useQuery({
        queryKey: ['stats', 'today'],
        queryFn: async () => {
            return apiClient.get<StatsDaily>('/api/v1/stats/today');
        },
        refetchInterval: 30000,
    });
}

/**
 * 获取每日统计数据 Hook
 */
export function useStatsDaily() {
    return useQuery({
        queryKey: ['stats', 'daily'],
        queryFn: async () => {
            return apiClient.get<StatsDaily[]>('/api/v1/stats/daily');
        },
        refetchInterval: 3600000, // 1 小时
    });
}
/**
 * 获取总统计数据 Hook
 */
export function useStatsTotal() {
    return useQuery({
        queryKey: ['stats', 'total'],
        queryFn: async () => {
            return apiClient.get<StatsTotal>('/api/v1/stats/total');
        },
        refetchInterval: 10000,// 10 秒
    });
}