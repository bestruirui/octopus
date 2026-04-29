'use client';

import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';
import {
    type GenerateAIRouteProgress,
    type GroupTestProgress,
    useGenerateAIRouteProgress,
    useGroupList,
    useGroupTestProgress,
} from './group';
import {
    clearStoredAIRouteTask,
    clearStoredGroupTestTask,
    readStoredAIRouteTask,
    readStoredGroupTestTask,
    type StoredAIRouteTask,
    type StoredGroupTestTask,
} from '@/components/modules/group/task-storage';

export type AnalyticsRange = '1d' | '7d' | '30d' | '90d' | 'ytd' | 'all';

export interface AnalyticsMetrics {
    request_count: number;
    total_tokens: number;
    input_tokens: number;
    output_tokens: number;
    total_cost: number;
    success_rate: number;
}

export interface AnalyticsOverview extends AnalyticsMetrics {
    provider_count: number;
    api_key_count: number;
    model_count: number;
    fallback_rate: number;
}

export interface AnalyticsProviderBreakdownItem extends AnalyticsMetrics {
    channel_id: number;
    channel_name: string;
    enabled: boolean;
}

export interface AnalyticsModelBreakdownItem extends AnalyticsMetrics {
    model_name: string;
}

export interface AnalyticsAPIKeyBreakdownItem extends AnalyticsMetrics {
    api_key_id?: number;
    name: string;
}

export interface AnalyticsUtilization {
    provider_breakdown: AnalyticsProviderBreakdownItem[];
    model_breakdown: AnalyticsModelBreakdownItem[];
    apikey_breakdown: AnalyticsAPIKeyBreakdownItem[];
}

export interface AnalyticsGroupHealthItem {
    group_id: number;
    group_name: string;
    endpoint_type: string;
    item_count: number;
    enabled_item_count: number;
    disabled_item_count: number;
    failure_count: number;
    last_failure_at: number;
    health_score: number;
    status: 'healthy' | 'warning' | 'degraded' | 'down' | 'empty';
}

export interface AnalyticsEvaluationRuntime {
    groupCount: number;
    hasGroups: boolean;
    aiRouteTask: StoredAIRouteTask | null;
    groupTestTask: StoredGroupTestTask | null;
    aiRouteProgress: GenerateAIRouteProgress | null;
    groupTestProgress: GroupTestProgress | null;
    aiRouteError: unknown;
    groupTestError: unknown;
    isLoading: boolean;
}

export interface SemanticCacheEvaluationSummary {
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
    evaluated_requests: number;
    cache_hit_responses: number;
    cache_miss_requests: number;
    bypassed_requests: number;
    stored_responses: number;
}

export interface AnalyticsEvaluationSummary {
    semantic_cache: SemanticCacheEvaluationSummary;
}

function getErrorStatusCode(error: unknown) {
    if (error && typeof error === 'object' && 'code' in error && typeof error.code === 'number') {
        return error.code;
    }
    return undefined;
}

export function useAnalyticsEvaluationRuntime(): AnalyticsEvaluationRuntime {
    const { data: groups = [], isLoading: isGroupsLoading } = useGroupList();
    const [aiRouteTask, setAiRouteTask] = useState<StoredAIRouteTask | null>(() => readStoredAIRouteTask());
    const [groupTestTask, setGroupTestTask] = useState<StoredGroupTestTask | null>(() => readStoredGroupTestTask());

    useEffect(() => {
        const syncFromStorage = () => {
            setAiRouteTask(readStoredAIRouteTask());
            setGroupTestTask(readStoredGroupTestTask());
        };

        window.addEventListener('focus', syncFromStorage);
        return () => window.removeEventListener('focus', syncFromStorage);
    }, []);

    const aiRouteProgressQuery = useGenerateAIRouteProgress(aiRouteTask?.id ?? null);
    const groupTestProgressQuery = useGroupTestProgress(groupTestTask?.id ?? null);

    useEffect(() => {
        if (!aiRouteTask?.id || getErrorStatusCode(aiRouteProgressQuery.error) !== 404) {
            return;
        }

        clearStoredAIRouteTask(aiRouteTask.id);
        queueMicrotask(() => setAiRouteTask((current) => (current?.id === aiRouteTask.id ? null : current)));
    }, [aiRouteProgressQuery.error, aiRouteTask]);

    useEffect(() => {
        if (!groupTestTask?.id || getErrorStatusCode(groupTestProgressQuery.error) !== 404) {
            return;
        }

        clearStoredGroupTestTask(groupTestTask.id);
        queueMicrotask(() => setGroupTestTask((current) => (current?.id === groupTestTask.id ? null : current)));
    }, [groupTestProgressQuery.error, groupTestTask]);

    const aiRouteError =
        aiRouteTask?.id && getErrorStatusCode(aiRouteProgressQuery.error) !== 404
            ? aiRouteProgressQuery.error
            : null;
    const groupTestError =
        groupTestTask?.id && getErrorStatusCode(groupTestProgressQuery.error) !== 404
            ? groupTestProgressQuery.error
            : null;

    return {
        groupCount: groups.length,
        hasGroups: groups.length > 0,
        aiRouteTask,
        groupTestTask,
        aiRouteProgress: aiRouteProgressQuery.data ?? null,
        groupTestProgress: groupTestProgressQuery.data ?? null,
        aiRouteError,
        groupTestError,
        isLoading: isGroupsLoading || aiRouteProgressQuery.isLoading || groupTestProgressQuery.isLoading,
    };
}

export function useAnalyticsOverview(range: AnalyticsRange) {
    return useQuery({
        queryKey: ['analytics', 'overview', range],
        queryFn: async () => apiClient.get<AnalyticsOverview>('/api/v1/analytics/overview', { range }),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useAnalyticsUtilization(range: AnalyticsRange) {
    return useQuery({
        queryKey: ['analytics', 'utilization', range],
        queryFn: async () => apiClient.get<AnalyticsUtilization>('/api/v1/analytics/utilization', { range }),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useAnalyticsGroupHealth() {
    return useQuery({
        queryKey: ['analytics', 'group-health'],
        queryFn: async () => apiClient.get<AnalyticsGroupHealthItem[]>('/api/v1/analytics/group-health'),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useAnalyticsEvaluationSummary() {
    return useQuery({
        queryKey: ['analytics', 'evaluation'],
        queryFn: async () => apiClient.get<AnalyticsEvaluationSummary>('/api/v1/analytics/evaluation'),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}
