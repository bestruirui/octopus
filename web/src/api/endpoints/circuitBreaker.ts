import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

/**
 * 熔断器状态（渠道 + 模型粒度）
 */
export interface CircuitBreaker {
    id?: number;
    channel_id: number;
    channel_name: string;
    model_name: string;
    /** 0=正常 1=熔断中 2=半开 */
    state: number;
    trip_count: number;
    consecutive_failures: number;
    /** 手动禁用（管理员显式关闭） */
    manual_disabled: boolean;
    /** 上次报错时间（Unix 秒） */
    last_error_time: number;
    /** 上次报错信息 */
    last_error: string;
}

/**
 * 获取熔断器状态列表 Hook
 */
export function useCircuitBreakerList() {
    return useQuery({
        queryKey: ['circuit-breaker', 'list'],
        queryFn: async () => {
            return apiClient.get<CircuitBreaker[]>('/api/v1/circuit-breaker/list');
        },
        refetchInterval: 15000,
        refetchOnMount: 'always',
    });
}

/**
 * 手动启用/禁用某个渠道+模型的熔断状态
 */
export function useCircuitBreakerManual() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: { channel_id: number; model_name: string; disabled: boolean }) => {
            return apiClient.post<null>('/api/v1/circuit-breaker/manual', data);
        },
        onSuccess: () => {
            logger.log('熔断器状态更新成功');
            queryClient.invalidateQueries({ queryKey: ['circuit-breaker', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'channel'] });
        },
        onError: (error) => {
            logger.error('熔断器状态更新失败:', error);
        },
    });
}

/**
 * 取消熔断：清除熔断状态与手动禁用标记
 */
export function useCircuitBreakerReset() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: { channel_id: number; model_name: string }) => {
            return apiClient.post<null>('/api/v1/circuit-breaker/reset', data);
        },
        onSuccess: () => {
            logger.log('熔断状态已重置');
            queryClient.invalidateQueries({ queryKey: ['circuit-breaker', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'channel'] });
        },
        onError: (error) => {
            logger.error('熔断状态重置失败:', error);
        },
    });
}
