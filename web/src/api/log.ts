import { useMutation, useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { apiRequest } from './client';

// RequestState 表示 Relay 请求的实时状态。
type RequestState = 'running' | 'committed' | 'success' | 'failed' | 'canceled';

// RelayAttempt 是详情流实时展示的一次渠道尝试。
interface RelayAttempt {
    attempt_index: number;
    channel_name: string;
    model_name: string;
    error: string;
}

// RelayLogOverview 是概览流中不含正文和尝试详情的日志。
export interface RelayLogOverview {
    id: number;
    state: RequestState;
    started_at: string;
    completed_at: string;
    duration: number;
    request_model: string;
    request_api_key_name?: string;
    actual_model: string;
    client_protocol: string;
    stream: boolean;
    final_channel_name: string;
    input_tokens: number;
    output_tokens: number;
    cache_read_tokens: number;
    cache_write_tokens: number;
    total_cost: number;
    error?: string;
}

function isFinished(state: RequestState) {
    return state === 'success' || state === 'failed' || state === 'canceled';
}

function sortLogs(logs: RelayLogOverview[]) {
    return [...logs].sort((a, b) => b.id - a.id);
}

// useClearLogs 清空已完成的内存日志。
export function useClearLogs() {
    return useMutation({
        mutationFn: () => apiRequest<null>('/api/v1/log/clear', { method: 'DELETE' }),
    });
}

// useStopAttempt 中止指定请求当前序号匹配的上游尝试。
export function useStopAttempt() {
    return useMutation({
        mutationFn: ({ requestId, attemptIndex }: { requestId: number; attemptIndex: number }) =>
            apiRequest<null>(`/api/v1/log/${requestId}/${attemptIndex}/stop`, { method: 'POST' }),
    });
}

// useLogs 订阅进程内日志概览，并按 RequestID 更新同一条记录。
export function useLogs() {
    const [logs, setLogs] = useState<RelayLogOverview[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        const source = new EventSource('/api/v1/log/overview/stream', { withCredentials: true });

        source.onopen = () => {
            setError(null);
            setIsLoading(false);
        };
        source.addEventListener('log', (event) => {
            try {
                const next = JSON.parse((event as MessageEvent<string>).data) as RelayLogOverview;
                setLogs((current) => sortLogs([next, ...current.filter((item) => item.id !== next.id)]));
                setIsLoading(false);
                setError(null);
            } catch {
                setError(new Error('Invalid log update'));
            }
        });
        source.onerror = () => {
            setIsLoading(false);
            setError(new Error('Log stream disconnected'));
        };

        return () => {
            source.close();
        };
    }, []);

    return { logs, isLoading, error };
}

// useLogRequestBody 在调用方启用时按需获取指定日志的请求体。
export function useLogRequestBody(id: number, startedAt: string, enabled: boolean) {
    return useQuery({
        queryKey: ['logs', id, startedAt, 'request-body'],
        queryFn: () => apiRequest<string>(`/api/v1/log/${id}/request-body`),
        enabled,
        staleTime: Infinity,
    });
}

// useLogResponseBody 在调用方启用时获取指定日志的最终响应体。
export function useLogResponseBody(id: number, startedAt: string, enabled: boolean) {
    return useQuery({
        queryKey: ['logs', id, startedAt, 'response-body'],
        queryFn: () => apiRequest<string>(`/api/v1/log/${id}/response-body`),
        enabled,
        staleTime: Infinity,
    });
}

// useLogDetailStream 为活动请求订阅单条尝试和响应提交增量。
export function useLogDetailStream(id: number, state: RequestState, enabled: boolean) {
    const [attempts, setAttempts] = useState<RelayAttempt[]>([]);
    const [runningAttempt, setRunningAttempt] = useState<RelayAttempt | null>(null);
    const [isCommitted, setIsCommitted] = useState(false);

    useEffect(() => {
        if (!enabled) {
            return;
        }

        setAttempts([]);
        setRunningAttempt(null);
        setIsCommitted(state === 'committed');
        if (isFinished(state)) return;

        const source = new EventSource(`/api/v1/log/${id}/stream`, { withCredentials: true });
        source.addEventListener('attempt.started', (event) => {
            try {
                const next = JSON.parse((event as MessageEvent<string>).data) as RelayAttempt;
                setRunningAttempt(next);
                setAttempts((current) => [...current.filter((attempt) => attempt.attempt_index !== next.attempt_index), next].slice(-50));
            } catch {
                return;
            }
        });
        source.addEventListener('attempt.finished', (event) => {
            try {
                const next = JSON.parse((event as MessageEvent<string>).data) as RelayAttempt;
                setRunningAttempt((current) => current?.attempt_index === next.attempt_index ? null : current);
                setAttempts((current) => {
                    return [...current.filter((attempt) => attempt.attempt_index !== next.attempt_index), next].slice(-50);
                });
            } catch {
                return;
            }
        });
        source.addEventListener('response.committed', () => {
            setRunningAttempt(null);
            setIsCommitted(true);
        });

        return () => {
            source.close();
        };
    }, [enabled, id, state]);

    return { attempts, runningAttempt, isCommitted };
}
