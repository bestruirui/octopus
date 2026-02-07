import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';
import { useRef } from 'react';

export interface GroupCircuitBreakerItemState {
    group_id: number;
    group_name: string;
    channel_id: number;
    channel_name: string;
    model_name: string;
    breaker_key: string;
    state: 'CLOSED' | 'OPEN' | 'HALF_OPEN';
    consecutive_failures: number;
    trip_count: number;
    last_failure_at?: string;
    last_failure_reason?: string;
    last_trip_at?: string;
    open_until?: string;
    open_remaining_second?: number;
    probe_in_flight: boolean;
}

export interface GroupCircuitBreakerStatesResponse {
    group_id: number;
    group_name: string;
    items: GroupCircuitBreakerItemState[];
}

export interface CircuitBreakerResetResponse {
    affected_breakers: number;
    breaker_key?: string;
    breaker_keys?: string[];
}

export function useGroupCircuitBreakerStates(groupId: number, enabled: boolean) {
    const lastHotAtRef = useRef(0);
    return useQuery({
        queryKey: ['circuit-breaker', 'group-states', groupId],
        queryFn: async () => {
            const data = await apiClient.get<GroupCircuitBreakerStatesResponse>(`/api/v1/circuit-breaker/group/${groupId}/states`);
            const hasHotState = (data.items ?? []).some(item => item.state === 'OPEN' || item.state === 'HALF_OPEN');
            if (hasHotState) lastHotAtRef.current = Date.now();
            return data;
        },
        enabled: enabled && groupId > 0,
        refetchInterval: () => Date.now() < lastHotAtRef.current + 120000 ? 5000 : 15000,
        refetchOnMount: 'always',
    });
}

export function useResetCircuitBreakerItem() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (payload: { channel_id: number; model_name: string }) =>
            apiClient.post<CircuitBreakerResetResponse>('/api/v1/circuit-breaker/item/reset', payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['circuit-breaker'] });
        },
        onError: (error) => logger.error('重置模型熔断状态失败:', error),
    });
}
