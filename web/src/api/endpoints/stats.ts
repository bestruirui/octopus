import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

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

export interface StatsDailyData {
    dateStr: string;
    isFuture: boolean;
    raw: StatsDaily | null;
    formatted: {
        request_count: { value: string; unit: string };
        wait_time: { value: string; unit: string };
        input_token: { value: string; unit: string };
        input_cost: { value: string; unit: string };
        output_token: { value: string; unit: string };
        output_cost: { value: string; unit: string };
    } | null;
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
        select: (data) => ({
            raw: data,
            formatted: data.map(item => ({
                date: item.date,
                input_token: formatCount(item.input_token),
                output_token: formatCount(item.output_token),
                request_count: formatCount(item.request_count),
                input_cost: formatMoney(item.input_cost),
                output_cost: formatMoney(item.output_cost),
                wait_time: formatTime(item.wait_time),
            }))
        }),
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