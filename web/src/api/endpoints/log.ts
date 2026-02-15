import type { InfiniteData } from '@tanstack/react-query';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient, API_BASE_URL } from '../client';
import { logger } from '@/lib/logger';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

/**
 * 尝试状态
 */
export type AttemptStatus = 'success' | 'failed' | 'circuit_break' | 'skipped';

/**
 * 单次渠道尝试信息
 */
export interface ChannelAttempt {
    channel_id: number;
    channel_key_id?: number;
    channel_name: string;
    model_name: string;
    attempt_num: number;
    status: AttemptStatus;
    duration: number;
    sticky?: boolean;
    msg?: string;
}

/**
 * 日志数据
 */
export interface RelayLog {
    id: number;
    time: number;                // 时间戳
    request_model_name: string;  // 请求模型名称
    channel: number;             // 实际使用的渠道ID
    channel_name: string;        // 渠道名称
    actual_model_name: string;   // 实际使用模型名称
    input_tokens: number;        // 输入Token
    output_tokens: number;       // 输出Token
    ftut: number;                // 首字时间(毫秒)
    use_time: number;            // 总用时(毫秒)
    cost: number;                // 消耗费用
    request_content: string;     // 请求内容
    response_content: string;    // 响应内容
    error: string;               // 错误信息
    attempts?: ChannelAttempt[]; // 所有尝试记录
    total_attempts?: number;     // 总尝试次数
}

export interface LogFilters {
    group?: string;
    model?: string;
    retried?: 'all' | 'yes' | 'no';
    channel?: string;
}

/**
 * 日志列表查询参数
 */
export interface LogListParams {
    page?: number;
    page_size?: number;
    start_time?: number;
    end_time?: number;
    group?: string;
    model?: string;
    retried?: boolean;
    channel?: string;
}

/**
 * 清空日志 Hook
 * 
 * @example
 * const clearLogs = useClearLogs();
 * 
 * clearLogs.mutate();
 */
export function useClearLogs() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async () => {
            return apiClient.delete<null>('/api/v1/log/clear');
        },
        onSuccess: () => {
            logger.log('日志清空成功');
            queryClient.invalidateQueries({ queryKey: ['logs'] });
        },
        onError: (error) => {
            logger.error('日志清空失败:', error);
        },
    });
}

function hasActiveFilter(filters: { group: string; model: string; retried: 'all' | 'yes' | 'no'; channel: string }): boolean {
    return Boolean(filters.group || filters.model || filters.channel || filters.retried !== 'all');
}

function matchLogFilter(log: RelayLog, filters: { group: string; model: string; retried: 'all' | 'yes' | 'no'; channel: string }): boolean {
    if (filters.group && log.request_model_name !== filters.group) {
        return false;
    }
    if (filters.model && log.actual_model_name !== filters.model) {
        return false;
    }

    if (filters.retried !== 'all') {
        const retried = (log.total_attempts ?? log.attempts?.length ?? 0) > 1;
        if (filters.retried === 'yes' && !retried) {
            return false;
        }
        if (filters.retried === 'no' && retried) {
            return false;
        }
    }

    if (filters.channel) {
        const channelMatched =
            log.channel_name === filters.channel ||
            !!log.attempts?.some((attempt) => attempt.channel_name === filters.channel);
        if (!channelMatched) {
            return false;
        }
    }

    return true;
}

const logsInfiniteQueryKey = (
    pageSize: number,
    filters: { group: string; model: string; retried: 'all' | 'yes' | 'no'; channel: string }
) => ['logs', 'infinite', pageSize, filters.group, filters.model, filters.retried, filters.channel] as const;

/**
 * 日志管理 Hook
 * 整合初始加载、SSE 实时推送、滚动加载更多
 * 
 * @example
 * const { logs, isConnected, hasMore, isLoadingMore, loadMore, clear } = useLogs();
 * 
 * // logs 自动包含历史日志和实时日志，按时间倒序
 * logs.forEach(log => console.log(log.request_model_name));
 * 
 * // 滚动到底部时加载更多
 * if (hasMore && !isLoadingMore) loadMore();
 */
export function useLogs(options: { pageSize?: number; filters?: LogFilters } = {}) {
    const { pageSize = 20, filters } = options;

    const groupFilter = (filters?.group ?? '').trim();
    const modelFilter = (filters?.model ?? '').trim();
    const retriedFilter = filters?.retried ?? 'all';
    const channelFilter = (filters?.channel ?? '').trim();

    const currentFilters = useMemo(() => ({
        group: groupFilter,
        model: modelFilter,
        retried: retriedFilter,
        channel: channelFilter,
    }), [groupFilter, modelFilter, retriedFilter, channelFilter]);

    const queryKey = useMemo(() => logsInfiniteQueryKey(pageSize, currentFilters), [pageSize, currentFilters]);
    const filterActive = hasActiveFilter(currentFilters);

    const [isConnected, setIsConnected] = useState(false);
    const [error, setError] = useState<Error | null>(null);
    const eventSourceRef = useRef<EventSource | null>(null);

    const queryClient = useQueryClient();

    const logsQuery = useInfiniteQuery({
        queryKey,
        initialPageParam: 1,
        queryFn: async ({ pageParam }) => {
            const params = new URLSearchParams();
            params.set('page', String(pageParam));
            params.set('page_size', String(pageSize));
            if (currentFilters.group) {
                params.set('group', currentFilters.group);
            }
            if (currentFilters.model) {
                params.set('model', currentFilters.model);
            }
            if (currentFilters.channel) {
                params.set('channel', currentFilters.channel);
            }
            if (currentFilters.retried !== 'all') {
                params.set('retried', currentFilters.retried === 'yes' ? 'true' : 'false');
            }
            const result = await apiClient.get<RelayLog[] | null>(`/api/v1/log/list?${params.toString()}`);
            return result ?? [];
        },
        getNextPageParam: (lastPage, allPages) => {
            if (!lastPage || lastPage.length < pageSize) return undefined;
            return allPages.length + 1;
        },
        staleTime: Infinity,
        refetchOnMount: 'always',
    });

    const logs = useMemo(() => {
        const pages = logsQuery.data?.pages ?? [];
        const seen = new Set<number>();
        const merged: RelayLog[] = [];

        for (const page of pages) {
            for (const log of page) {
                if (seen.has(log.id)) continue;
                seen.add(log.id);
                merged.push(log);
            }
        }

        merged.sort((a, b) => b.time - a.time);
        return merged;
    }, [logsQuery.data]);

    const loadMore = useCallback(async () => {
        if (!logsQuery.hasNextPage) return;
        if (logsQuery.isFetchingNextPage) return;

        try {
            await logsQuery.fetchNextPage();
        } catch (e) {
            logger.error('加载更多日志失败:', e);
        }
    }, [logsQuery]);

    useEffect(() => {
        let cancelled = false;

        const connect = async () => {
            try {
                const { token } = await apiClient.get<{ token: string }>('/api/v1/log/stream-token');
                if (cancelled) return;

                const eventSource = new EventSource(`${API_BASE_URL}/api/v1/log/stream?token=${token}`);
                eventSourceRef.current = eventSource;

                eventSource.onopen = () => {
                    setIsConnected(true);
                    setError(null);
                };

                eventSource.onmessage = (event) => {
                    try {
                        const log: RelayLog = JSON.parse(event.data);
                        if (filterActive && !matchLogFilter(log, currentFilters)) {
                            return;
                        }
                        queryClient.setQueryData(
                            queryKey,
                            (old: InfiniteData<RelayLog[], number> | undefined) => {
                                if (!old) {
                                    return { pages: [[log]], pageParams: [1] };
                                }

                                const exists = old.pages.some((p) => p?.some((x) => x.id === log.id));
                                if (exists) return old;

                                const firstPage = old.pages[0] ?? [];
                                return { ...old, pages: [[log, ...firstPage], ...old.pages.slice(1)] };
                            }
                        );
                    } catch (e) {
                        logger.error('解析日志数据失败:', e);
                    }
                };

                eventSource.onerror = () => {
                    setIsConnected(false);
                    setError(new Error('SSE 连接断开'));
                    eventSource.close();
                    eventSourceRef.current = null;
                };
            } catch (e) {
                if (cancelled) return;
                setError(e instanceof Error ? e : new Error('获取 stream token 失败'));
                logger.error('获取 stream token 失败:', e);
            }
        };

        connect();

        return () => {
            cancelled = true;
            eventSourceRef.current?.close();
            eventSourceRef.current = null;
            setIsConnected(false);
        };
    }, [currentFilters, filterActive, queryClient, queryKey]);

    const clear = useCallback(() => {
        queryClient.removeQueries({ queryKey });
    }, [queryClient, queryKey]);

    return {
        logs,
        isConnected,
        error,
        hasMore: !!logsQuery.hasNextPage,
        isLoading: logsQuery.isLoading,
        isLoadingMore: logsQuery.isFetchingNextPage,
        loadMore,
        clear,
    };
}
