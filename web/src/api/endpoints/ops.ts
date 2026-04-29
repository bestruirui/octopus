'use client';

import { useCallback, useMemo, useState } from 'react';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';

export interface OpsCacheStatus {
    enabled: boolean;
    runtime_enabled: boolean;
    ttl_seconds: number;
    threshold: number;
    max_entries: number;
    current_entries: number;
    hits: number;
    misses: number;
    hit_rate: number;
    usage_rate: number;
}

export interface OpsQuotaKeyItem {
    api_key_id: number;
    name: string;
    enabled: boolean;
    expired: boolean;
    status: 'open' | 'limited' | 'exhausted' | 'expired' | 'disabled';
    supported_model_count: number;
    has_per_model_quota: boolean;
    rate_limit_rpm: number;
    rate_limit_tpm: number;
    max_cost: number;
    request_count: number;
    total_cost: number;
}

export interface OpsQuotaSummary {
    total_key_count: number;
    enabled_key_count: number;
    available_key_count: number;
    expired_key_count: number;
    limited_key_count: number;
    unlimited_key_count: number;
    exhausted_key_count: number;
    per_model_quota_key_count: number;
    active_usage_key_count: number;
    total_rpm: number;
    total_tpm: number;
    total_max_cost: number;
    keys: OpsQuotaKeyItem[];
}

export interface OpsHealthGroupItem {
    group_id: number;
    group_name: string;
    endpoint_type: string;
    status: 'healthy' | 'warning' | 'degraded' | 'down' | 'empty';
    failure_count: number;
    health_score: number;
}

export interface OpsHealthStatus {
    database_ok: boolean;
    cache_ok: boolean;
    task_runtime_ok: boolean;
    recent_error_count: number;
    healthy_group_count: number;
    warning_group_count: number;
    degraded_group_count: number;
    down_group_count: number;
    empty_group_count: number;
    failing_groups: OpsHealthGroupItem[];
    checked_at: number;
}

export interface OpsAIRouteServiceSummary {
    name: string;
    base_url: string;
    model: string;
    enabled: boolean;
}

export interface OpsSystemSummary {
    version: string;
    commit: string;
    build_time: string;
    repo: string;
    database_type: string;
    public_api_base_url: string;
    proxy_url: string;
    relay_log_keep_enabled: boolean;
    relay_log_keep_days: number;
    stats_save_interval_minutes: number;
    sync_llm_interval_hours: number;
    model_info_update_interval_hours: number;
    import_enabled: boolean;
    export_enabled: boolean;
    ai_route_group_id: number;
    ai_route_timeout_seconds: number;
    ai_route_parallelism: number;
    ai_route_legacy_mode: boolean;
    ai_route_service_count: number;
    ai_route_enabled_service_count: number;
    ai_route_services: OpsAIRouteServiceSummary[];
    channel_count: number;
    group_count: number;
    api_key_count: number;
}

export interface AuditLogEntry {
    id: number;
    user_id: number;
    username: string;
    action: string;
    method: string;
    path: string;
    status_code: number;
    target: string;
    created_at: number;
}

type OpsQuotaSummaryServer = Omit<OpsQuotaSummary, 'keys'> & {
    keys: OpsQuotaKeyItem[] | null;
};

type OpsHealthStatusServer = Omit<OpsHealthStatus, 'failing_groups'> & {
    failing_groups: OpsHealthGroupItem[] | null;
};

type OpsSystemSummaryServer = Omit<OpsSystemSummary, 'ai_route_services'> & {
    ai_route_services: OpsAIRouteServiceSummary[] | null;
};

const auditLogsInfiniteQueryKey = (pageSize: number) => ['audit', 'infinite', pageSize] as const;

export const DEFAULT_AUDIT_PAGE_SIZE = 12;

export function useOpsCacheStatus() {
    return useQuery({
        queryKey: ['ops', 'cache'],
        queryFn: async () => apiClient.get<OpsCacheStatus>('/api/v1/ops/cache'),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useOpsQuotaSummary() {
    return useQuery({
        queryKey: ['ops', 'quota'],
        queryFn: async () => apiClient.get<OpsQuotaSummaryServer>('/api/v1/ops/quota'),
        select: (data): OpsQuotaSummary => ({
            ...data,
            keys: data.keys ?? [],
        }),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useOpsHealthStatus() {
    return useQuery({
        queryKey: ['ops', 'health'],
        queryFn: async () => apiClient.get<OpsHealthStatusServer>('/api/v1/ops/health'),
        select: (data): OpsHealthStatus => ({
            ...data,
            failing_groups: data.failing_groups ?? [],
        }),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useOpsSystemSummary() {
    return useQuery({
        queryKey: ['ops', 'system'],
        queryFn: async () => apiClient.get<OpsSystemSummaryServer>('/api/v1/ops/system'),
        select: (data): OpsSystemSummary => ({
            ...data,
            ai_route_services: data.ai_route_services ?? [],
        }),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useAuditLogs(options: { pageSize?: number } = {}) {
    const { pageSize = DEFAULT_AUDIT_PAGE_SIZE } = options;

    const auditLogsQuery = useInfiniteQuery({
        queryKey: auditLogsInfiniteQueryKey(pageSize),
        initialPageParam: 1,
        queryFn: async ({ pageParam }) => {
            const params = new URLSearchParams();
            params.set('page', String(pageParam));
            params.set('page_size', String(pageSize));
            const result = await apiClient.get<AuditLogEntry[] | null>(`/api/v1/audit/list?${params.toString()}`);
            return result ?? [];
        },
        getNextPageParam: (lastPage, allPages) => {
            if (!lastPage || lastPage.length < pageSize) return undefined;
            return allPages.length + 1;
        },
        staleTime: 30000,
        refetchOnMount: 'always',
    });

    const logs = useMemo(() => {
        const pages = auditLogsQuery.data?.pages ?? [];
        const seen = new Set<number>();
        const merged: AuditLogEntry[] = [];

        for (const page of pages) {
            for (const item of page) {
                if (seen.has(item.id)) continue;
                seen.add(item.id);
                merged.push(item);
            }
        }

        merged.sort((left, right) => {
            if (left.created_at !== right.created_at) {
                return right.created_at - left.created_at;
            }
            return right.id - left.id;
        });
        return merged;
    }, [auditLogsQuery.data]);

    const loadMore = useCallback(async () => {
        if (!auditLogsQuery.hasNextPage || auditLogsQuery.isFetchingNextPage) {
            return;
        }
        await auditLogsQuery.fetchNextPage();
    }, [auditLogsQuery]);

    return {
        logs,
        error: auditLogsQuery.error,
        hasMore: !!auditLogsQuery.hasNextPage,
        isLoading: auditLogsQuery.isLoading,
        isLoadingMore: auditLogsQuery.isFetchingNextPage,
        loadMore,
    };
}

export function useAuditLogDetail() {
    const [detail, setDetail] = useState<AuditLogEntry | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    const fetchDetail = useCallback(async (id: number) => {
        setIsLoading(true);
        try {
            const result = await apiClient.get<AuditLogEntry | null>(`/api/v1/audit/detail?id=${id}`);
            setDetail(result);
        } catch {
            setDetail(null);
        } finally {
            setIsLoading(false);
        }
    }, []);

    const reset = useCallback(() => {
        setDetail(null);
    }, []);

    return { detail, isLoading, fetchDetail, reset };
}
