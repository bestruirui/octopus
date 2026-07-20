import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

/**
 * 统计数据
 */
interface StatsMetrics {
    input_token: number;
    output_token: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
    request_success: number;
    request_failed: number;
}

export interface StatsMetricsFormatted {
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    input_cost: ReturnType<typeof formatMoney>;
    output_cost: ReturnType<typeof formatMoney>;
    wait_time: ReturnType<typeof formatTime>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;

    request_count: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
}

export interface StatsChannel extends StatsMetrics {
    channel_id: number;
}

export interface StatsDaily extends StatsMetrics {
    date: string;
}
export interface StatsDailyFormatted extends StatsMetricsFormatted {
    date: string;
}

export interface StatsTotal extends StatsMetrics {
    id: number;
}
export type StatsTotalFormatted = StatsMetricsFormatted;

export interface StatsHourly extends StatsMetrics {
    hour: number;
    date: string;
}
export interface StatsHourlyFormatted extends StatsMetricsFormatted {
    hour: number;
    date: string;
}
/**
 * API Key 统计数据
 */
export interface StatsAPIKey extends StatsMetrics {
    api_key_id: number;
}

export interface StatsAPIKeyFormatted extends StatsMetricsFormatted {
    api_key_id: number;
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
        refetchOnMount: 'always',
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
        select: (data) => data.map((item): StatsDailyFormatted => ({
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            total_token: formatCount(item.input_token + item.output_token),
            input_cost: formatMoney(item.input_cost),
            output_cost: formatMoney(item.output_cost),
            total_cost: formatMoney(item.input_cost + item.output_cost),
            wait_time: formatTime(item.wait_time),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
            date: item.date,
        })),
        refetchInterval: 3600000, // 1 小时
        refetchOnMount: 'always',
    });
}
/**
 * 获取总统计数据 Hook
 */
export function useStatsHourly() {
    return useQuery({
        queryKey: ['stats', 'hourly'],
        queryFn: async () => {
            return apiClient.get<StatsHourly[]>('/api/v1/stats/hourly');
        },
        select: (data) => data.map((item): StatsHourlyFormatted => ({
            hour: item.hour,
            date: item.date,
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            total_token: formatCount(item.input_token + item.output_token),
            input_cost: formatMoney(item.input_cost),
            output_cost: formatMoney(item.output_cost),
            total_cost: formatMoney(item.input_cost + item.output_cost),
            wait_time: formatTime(item.wait_time),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
        })),
        refetchInterval: 10000,// 10 秒
        refetchOnMount: 'always',
    });
}

export function useStatsTotal() {
    return useQuery({
        queryKey: ['stats', 'total'],
        queryFn: async () => {
            return apiClient.get<StatsTotal>('/api/v1/stats/total');
        },
        select: (data) => ({
            input_token: formatCount(data.input_token),
            output_token: formatCount(data.output_token),
            total_token: formatCount(data.input_token + data.output_token),
            input_cost: formatMoney(data.input_cost),
            output_cost: formatMoney(data.output_cost),
            total_cost: formatMoney(data.input_cost + data.output_cost),
            wait_time: formatTime(data.wait_time),
            request_success: formatCount(data.request_success),
            request_failed: formatCount(data.request_failed),
            request_count: formatCount(data.request_success + data.request_failed),
        }),
        refetchInterval: 10000,// 10 秒
        refetchOnMount: 'always',
    });
}



/**
 * 获取 API Key 统计数据列表 Hook
 */
export function useStatsAPIKey() {
    return useQuery({
        queryKey: ['stats', 'apikey'],
        queryFn: async () => {
            return apiClient.get<StatsAPIKey[]>('/api/v1/stats/apikey');
        },
        select: (data) => data.map((item): StatsAPIKeyFormatted => ({
            api_key_id: item.api_key_id,
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            total_token: formatCount(item.input_token + item.output_token),
            input_cost: formatMoney(item.input_cost),
            output_cost: formatMoney(item.output_cost),
            total_cost: formatMoney(item.input_cost + item.output_cost),
            wait_time: formatTime(item.wait_time),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
        })),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/** 渠道健康状态 */
export type ChannelHealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'unknown';

export interface ChannelHealth {
    channel_id: number;
    channel_name: string;
    status: ChannelHealthStatus;
    last_probe_time: number;
    last_probe_ok: boolean;
    last_probe_delay: number;
    last_probe_error?: string;
    fail_streak: number;
    circuit_open: boolean;
    circuit_remain_sec?: number;
    base_url?: string;
}

export interface ChannelRealtimeItem {
    channel_id: number;
    channel_name: string;
    enabled: boolean;
    stats: StatsMetrics;
    health: ChannelHealth;
    success_rate: number;
    avg_latency_ms: number;
    total_cost: number;
}

export interface RealtimeSummary {
    healthy_count: number;
    degraded_count: number;
    unhealthy_count: number;
    unknown_count: number;
    success_rate: number;
    avg_latency_ms: number;
    total_cost: number;
    request_count: number;
}

export interface RealtimeDashboard {
    generated_at: number;
    today: StatsDaily;
    hourly: StatsHourly[];
    total: StatsTotal;
    channels: ChannelRealtimeItem[];
    summary: RealtimeSummary;
}

/** 实时看板（含渠道健康 + 成功率 + 延迟 + 费用）
 *  refreshSec: 看板刷新间隔（秒），由前端设置 health_dashboard_refresh 控制
 */
export function useRealtimeDashboard(refreshSec = 15) {
    const intervalMs = Math.max(3, refreshSec) * 1000;
    return useQuery({
        queryKey: ['stats', 'realtime'],
        queryFn: async () => {
            return apiClient.get<RealtimeDashboard>('/api/v1/stats/realtime');
        },
        refetchInterval: intervalMs,
        refetchOnMount: 'always',
    });
}

/** 渠道健康列表 */
export function useChannelHealth(refreshSec = 15) {
    const intervalMs = Math.max(3, refreshSec) * 1000;
    return useQuery({
        queryKey: ['stats', 'health'],
        queryFn: async () => {
            return apiClient.get<ChannelHealth[]>('/api/v1/stats/health');
        },
        refetchInterval: intervalMs,
        refetchOnMount: 'always',
    });
}