import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import { apiClient, API_BASE_URL } from '../client';
import { logger } from '@/lib/logger';
import { useAuthStore } from './user';

/**
 * 分组项信息
 */
export interface GroupItem {
    id?: number;
    group_id?: number;
    channel_id: number;
    model_name: string;
    priority: number;
    weight: number;
}

/**
 * 分组模式
 */
export enum GroupMode {
    RoundRobin = 1,
    Random = 2,
    Failover = 3,
    Weighted = 4,
    Auto = 5,
}

/**
 * 分组信息
 */
export interface Group {
    id?: number;
    name: string;
    endpoint_type: string;
    mode: GroupMode;
    match_regex: string;
    first_token_time_out?: number;
    session_keep_time?: number;
    items?: GroupItem[];
}

export interface GroupTestResult {
    item_id: number;
    channel_id: number;
    channel_name: string;
    model_name: string;
    passed: boolean;
    attempts: number;
    status_code: number;
    response_text?: string;
    message?: string;
}

export interface GroupTestSummary {
    passed: boolean;
    completed: number;
    total: number;
    results: GroupTestResult[];
}

export interface GroupTestProgress extends GroupTestSummary {
    id: string;
    done: boolean;
    message?: string;
}

export interface AutoGroupCreatedItem {
    name: string;
    endpoint_type: string;
    matched_models: string[];
}

export interface AutoGroupSkippedItem {
    name: string;
    endpoint_type: string;
    reason: string;
}

export interface AutoGroupResult {
    total_channels: number;
    total_models_seen: number;
    total_distinct_raw_models: number;
    total_candidates: number;
    created_groups: number;
    skipped_existing_groups: number;
    skipped_covered_models: number;
    failed_groups: number;
    created: AutoGroupCreatedItem[];
    skipped: AutoGroupSkippedItem[];
}

export type AIRouteScope = 'group' | 'table';

export interface GenerateAIRouteRequest {
    scope?: AIRouteScope;
    group_id?: number;
}

export interface GenerateAIRouteResult {
    scope?: AIRouteScope;
    group_id?: number;
    group_count: number;
    route_count: number;
    item_count: number;
}

export type AIRouteTaskStatus = 'queued' | 'running' | 'completed' | 'failed' | 'timeout';

export type AIRouteTaskStep =
    | 'queued'
    | 'collecting_models'
    | 'building_batches'
    | 'analyzing_batches'
    | 'parsing_response'
    | 'validating_routes'
    | 'writing_groups'
    | 'finalizing'
    | 'completed'
    | 'failed'
    | 'timeout';

export type AIRouteChannelStatus = 'pending' | 'running' | 'completed' | 'failed';

export type AIRouteBatchStatus = 'running' | 'parsing' | 'retrying' | 'failed';

export interface GenerateAIRouteProgressSummary {
    total_channels: number;
    completed_channels: number;
    running_channels: number;
    pending_channels: number;
    failed_channels: number;
    total_models: number;
    completed_models: number;
}

export interface GenerateAIRouteCurrentBatch {
    index: number;
    total: number;
    endpoint_type?: string;
    model_count: number;
    channel_ids?: number[];
    channel_names?: string[];
    service_name?: string;
    attempt?: number;
    status?: string;
    message?: string;
    message_key?: string;
    message_args?: Record<string, unknown>;
}

export interface GenerateAIRouteRunningBatch {
    index: number;
    total: number;
    endpoint_type?: string;
    model_count: number;
    channel_ids?: number[];
    channel_names?: string[];
    service_name?: string;
    attempt?: number;
    status?: AIRouteBatchStatus;
    message?: string;
    message_key?: string;
    message_args?: Record<string, unknown>;
}

export interface GenerateAIRouteChannelProgress {
    channel_id: number;
    channel_name?: string;
    provider?: string;
    status?: AIRouteChannelStatus;
    total_models: number;
    processed_models: number;
    message?: string;
    message_key?: string;
    message_args?: Record<string, unknown>;
}

export interface GenerateAIRouteProgress {
    id: string;
    scope?: AIRouteScope;
    group_id?: number;
    status?: AIRouteTaskStatus;
    current_step?: AIRouteTaskStep;
    progress_percent: number;
    total_batches: number;
    completed_batches: number;
    done: boolean;
    result_ready: boolean;
    message?: string;
    message_key?: string;
    message_args?: Record<string, unknown>;
    error_reason?: string;
    error_reason_key?: string;
    error_reason_args?: Record<string, unknown>;
    started_at?: string;
    updated_at?: string;
    heartbeat_at?: string;
    finished_at?: string;
    event_sequence?: number;
    summary?: GenerateAIRouteProgressSummary;
    current_batch?: GenerateAIRouteCurrentBatch;
    running_batches?: GenerateAIRouteRunningBatch[];
    channels: GenerateAIRouteChannelProgress[];
    result?: GenerateAIRouteResult;
}

function normalizeGenerateAIRouteProgress(progress: GenerateAIRouteProgress): GenerateAIRouteProgress {
    const normalizedChannels = Array.isArray(progress.channels)
        ? progress.channels.map((channel) => ({
            ...channel,
            status: channel.status ?? 'pending',
            total_models: typeof channel.total_models === 'number' ? channel.total_models : 0,
            processed_models: typeof channel.processed_models === 'number' ? channel.processed_models : 0,
        }))
        : [];

    const normalizedSummary = progress.summary
        ? {
            total_channels: typeof progress.summary.total_channels === 'number' ? progress.summary.total_channels : normalizedChannels.length,
            completed_channels: typeof progress.summary.completed_channels === 'number' ? progress.summary.completed_channels : 0,
            running_channels: typeof progress.summary.running_channels === 'number' ? progress.summary.running_channels : 0,
            pending_channels: typeof progress.summary.pending_channels === 'number' ? progress.summary.pending_channels : 0,
            failed_channels: typeof progress.summary.failed_channels === 'number' ? progress.summary.failed_channels : 0,
            total_models: typeof progress.summary.total_models === 'number' ? progress.summary.total_models : 0,
            completed_models: typeof progress.summary.completed_models === 'number' ? progress.summary.completed_models : 0,
        }
        : undefined;

    const normalizedBatch = progress.current_batch
        ? {
            ...progress.current_batch,
            index: typeof progress.current_batch.index === 'number' ? progress.current_batch.index : 0,
            total: typeof progress.current_batch.total === 'number' ? progress.current_batch.total : 0,
            model_count: typeof progress.current_batch.model_count === 'number' ? progress.current_batch.model_count : 0,
            channel_ids: Array.isArray(progress.current_batch.channel_ids) ? progress.current_batch.channel_ids : [],
            channel_names: Array.isArray(progress.current_batch.channel_names) ? progress.current_batch.channel_names : [],
            service_name: typeof progress.current_batch.service_name === 'string' ? progress.current_batch.service_name : undefined,
            attempt: typeof progress.current_batch.attempt === 'number' ? progress.current_batch.attempt : 0,
            status: typeof progress.current_batch.status === 'string' ? progress.current_batch.status : undefined,
            message: typeof progress.current_batch.message === 'string' ? progress.current_batch.message : undefined,
        }
        : undefined;

    const normalizedRunningBatches = Array.isArray(progress.running_batches)
        ? progress.running_batches.map((batch) => ({
            ...batch,
            index: typeof batch.index === 'number' ? batch.index : 0,
            total: typeof batch.total === 'number' ? batch.total : 0,
            model_count: typeof batch.model_count === 'number' ? batch.model_count : 0,
            channel_ids: Array.isArray(batch.channel_ids) ? batch.channel_ids : [],
            channel_names: Array.isArray(batch.channel_names) ? batch.channel_names : [],
            service_name: typeof batch.service_name === 'string' ? batch.service_name : undefined,
            attempt: typeof batch.attempt === 'number' ? batch.attempt : 0,
            status: typeof batch.status === 'string' ? batch.status as AIRouteBatchStatus : undefined,
            message: typeof batch.message === 'string' ? batch.message : undefined,
        }))
        : [];

    return {
        ...progress,
        done: Boolean(progress.done),
        result_ready: Boolean(progress.result_ready ?? progress.result),
        status: progress.status ?? (progress.done ? (progress.message ? 'failed' : 'completed') : 'running'),
        current_step: progress.current_step ?? (progress.done ? 'completed' : 'queued'),
        progress_percent: typeof progress.progress_percent === 'number' ? progress.progress_percent : 0,
        total_batches: typeof progress.total_batches === 'number' ? progress.total_batches : 0,
        completed_batches: typeof progress.completed_batches === 'number' ? progress.completed_batches : 0,
        event_sequence: typeof progress.event_sequence === 'number' ? progress.event_sequence : 0,
        error_reason: typeof progress.error_reason === 'string' ? progress.error_reason : undefined,
        summary: normalizedSummary,
        current_batch: normalizedBatch,
        running_batches: normalizedRunningBatches,
        channels: normalizedChannels,
    };
}

export interface DeleteAllGroupsResult {
    deleted_count: number;
}

export function isGenerateAIRouteTerminal(progress: GenerateAIRouteProgress | null | undefined) {
    if (!progress) {
        return false;
    }

    switch (progress.status) {
        case 'completed':
            return progress.done && progress.result_ready;
        case 'failed':
        case 'timeout':
            return progress.done;
        default:
            return false;
    }
}

function normalizeGroupTestProgress(progress: GroupTestProgress): GroupTestProgress {
    return {
        ...progress,
        results: Array.isArray(progress.results) ? progress.results : [],
        completed: typeof progress.completed === 'number' ? progress.completed : 0,
        total: typeof progress.total === 'number' ? progress.total : 0,
        done: Boolean(progress.done),
        passed: Boolean(progress.passed),
    };
}

/**
 * 新增 item 请求
 */
export interface GroupItemAddRequest {
    channel_id: number;
    model_name: string;
    priority: number;
    weight: number;
}

/**
 * 更新 item 请求 (仅 priority)
 */
export interface GroupItemUpdateRequest {
    id: number;
    priority: number;
    weight: number;
}

/**
 * 分组更新请求 - 仅包含变更的数据
 */
export interface GroupUpdateRequest {
    id: number;
    name?: string;
    endpoint_type?: string;
    mode?: GroupMode;
    match_regex?: string;
    first_token_time_out?: number;
    session_keep_time?: number;
    items_to_add?: GroupItemAddRequest[];
    items_to_update?: GroupItemUpdateRequest[];
    items_to_delete?: number[];
}

export function useGroupList() {
    return useQuery({
        queryKey: ['groups', 'list'],
        queryFn: async () => {
            return apiClient.get<Group[]>('/api/v1/group/list');
        },
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

export function useCreateGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: Group) => {
            return apiClient.post<Group>('/api/v1/group/create', data);
        },
        onSuccess: (data) => {
            logger.log('分组创建成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组创建失败:', error);
        },
    });
}

export function useUpdateGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: GroupUpdateRequest) => {
            return apiClient.post<Group>('/api/v1/group/update', data);
        },
        onSuccess: (data) => {
            logger.log('分组更新成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组更新失败:', error);
        },
    });
}

export function useDeleteGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/group/delete/${id}`);
        },
        onSuccess: () => {
            logger.log('分组删除成功');
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组删除失败:', error);
        },
    });
}

export function useAutoGroupModels() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async () => {
            return apiClient.post<AutoGroupResult>('/api/v1/group/auto-group', {});
        },
        onSuccess: (data) => {
            logger.log('自动分组成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('自动分组失败:', error);
        },
    });
}

export function useDeleteAllGroups() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async () => {
            return apiClient.delete<DeleteAllGroupsResult>('/api/v1/group/delete-all');
        },
        onSuccess: (data) => {
            logger.log('全部分组删除成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['settings', 'list'] });
        },
        onError: (error) => {
            logger.error('全部分组删除失败:', error);
        },
    });
}

export function useGenerateAIRoute() {
    return useMutation({
        mutationFn: async (data: GenerateAIRouteRequest) => {
            const progress = await apiClient.post<GenerateAIRouteProgress>('/api/v1/route/ai-generate', data);
            return normalizeGenerateAIRouteProgress(progress);
        },
        onSuccess: (data) => {
            logger.log('AI 路由任务已启动:', data);
        },
        onError: (error) => {
            logger.error('AI 路由生成失败:', error);
        },
    });
}

function useGenerateAIRouteProgressStream(progressId: string | null) {
    const queryClient = useQueryClient();
    const token = useAuthStore((state) => state.token);
    const [isConnected, setIsConnected] = useState(false);
    const abortRef = useRef<AbortController | null>(null);

    useEffect(() => {
        if (!progressId || !token) {
            setIsConnected(false);
            return;
        }

        let cancelled = false;
        let retryTimer: number | null = null;
        let retryAttempt = 0;

        const waitForRetry = (delayMs: number) =>
            new Promise<void>((resolve) => {
                retryTimer = window.setTimeout(() => {
                    retryTimer = null;
                    resolve();
                }, delayMs);
            });

        const connect = async () => {
            while (!cancelled) {
                const latestProgress = queryClient.getQueryData<GenerateAIRouteProgress>(['groups', 'ai-route-progress', progressId]);
                if (isGenerateAIRouteTerminal(latestProgress)) {
                    setIsConnected(false);
                    return;
                }

                try {
                    const controller = new AbortController();
                    abortRef.current = controller;

                    const response = await fetch(`${API_BASE_URL}/api/v1/route/ai-generate/stream/${progressId}`, {
                        method: 'GET',
                        headers: {
                            Authorization: `Bearer ${token}`,
                        },
                        signal: controller.signal,
                    });
                    if (cancelled) {
                        return;
                    }
                    if (!response.ok) {
                        throw new Error(`AI 路由进度流连接失败: ${response.status}`);
                    }
                    if (!response.body) {
                        throw new Error('AI 路由进度流响应为空');
                    }

                    retryAttempt = 0;
                    setIsConnected(true);

                    const reader = response.body.getReader();
                    const decoder = new TextDecoder();
                    let buffer = '';

                    const handleEvent = (chunk: string) => {
                        const lines = chunk.split('\n');
                        const dataLines: string[] = [];
                        for (const line of lines) {
                            if (line.startsWith('data:')) {
                                dataLines.push(line.slice(5).trimStart());
                            }
                        }
                        if (dataLines.length === 0) {
                            return;
                        }

                        try {
                            const incoming = normalizeGenerateAIRouteProgress(
                                JSON.parse(dataLines.join('\n')) as GenerateAIRouteProgress,
                            );
                            queryClient.setQueryData<GenerateAIRouteProgress>(
                                ['groups', 'ai-route-progress', progressId],
                                (current) => {
                                    if (current && (current.event_sequence ?? 0) > (incoming.event_sequence ?? 0)) {
                                        return current;
                                    }
                                    return incoming;
                                },
                            );
                        } catch (error) {
                            logger.error('解析 AI 路由进度流失败:', error);
                        }
                    };

                    while (!cancelled) {
                        const { value, done } = await reader.read();
                        if (done) {
                            break;
                        }

                        buffer += decoder.decode(value, { stream: true });
                        let boundary = buffer.indexOf('\n\n');
                        while (boundary >= 0) {
                            const eventChunk = buffer.slice(0, boundary);
                            buffer = buffer.slice(boundary + 2);
                            handleEvent(eventChunk);
                            boundary = buffer.indexOf('\n\n');
                        }

                        const nextProgress = queryClient.getQueryData<GenerateAIRouteProgress>(['groups', 'ai-route-progress', progressId]);
                        if (isGenerateAIRouteTerminal(nextProgress)) {
                            controller.abort();
                            setIsConnected(false);
                            return;
                        }
                    }

                    if (cancelled) {
                        return;
                    }
                    setIsConnected(false);
                } catch (error) {
                    if (cancelled) {
                        return;
                    }
                    if (error instanceof Error && error.name === 'AbortError') {
                        return;
                    }

                    setIsConnected(false);
                    logger.warn('AI 路由进度流连接失败，准备回退轮询:', error);
                } finally {
                    abortRef.current = null;
                }

                const delayMs = Math.min(1000 * 2 ** retryAttempt, 10000);
                retryAttempt += 1;
                await waitForRetry(delayMs);
            }
        };

        connect();

        return () => {
            cancelled = true;
            if (retryTimer !== null) {
                window.clearTimeout(retryTimer);
            }
            abortRef.current?.abort();
            abortRef.current = null;
            setIsConnected(false);
        };
    }, [progressId, queryClient, token]);

    return { isConnected };
}

export function useGenerateAIRouteProgress(progressId: string | null) {
    const { isConnected } = useGenerateAIRouteProgressStream(progressId);

    const query = useQuery({
        queryKey: ['groups', 'ai-route-progress', progressId],
        queryFn: async () => {
            const progress = await apiClient.get<GenerateAIRouteProgress>(`/api/v1/route/ai-generate/progress/${progressId}`);
            return normalizeGenerateAIRouteProgress(progress);
        },
        enabled: Boolean(progressId),
        retry: false,
        refetchInterval: (query) => {
            const data = query.state.data;
            if (!progressId || isGenerateAIRouteTerminal(data)) {
                return false;
            }
            if (isConnected) {
                return false;
            }
            return 800;
        },
    });

    return {
        ...query,
        isStreamConnected: isConnected,
    };
}

export function useTestGroup() {
    return useMutation({
        mutationFn: async (groupId: number) => {
            const progress = await apiClient.post<GroupTestProgress>('/api/v1/group/test', { group_id: groupId });
            return normalizeGroupTestProgress(progress);
        },
        onSuccess: (data) => {
            logger.log('分组检测成功:', data);
        },
        onError: (error) => {
            logger.error('分组检测失败:', error);
        },
    });
}

export function useGroupTestProgress(progressId: string | null) {
    return useQuery({
        queryKey: ['groups', 'test-progress', progressId],
        queryFn: async () => {
            const progress = await apiClient.get<GroupTestProgress>(`/api/v1/group/test/progress/${progressId}`);
            return normalizeGroupTestProgress(progress);
        },
        enabled: Boolean(progressId),
        refetchInterval: (query) => {
            const data = query.state.data;
            if (!progressId || data?.done) {
                return false;
            }
            return 800;
        },
    });
}

/**
 * 自动添加分组 item Hook
 *
 * 后端路由: POST /api/v1/group/auto-add-item
 * Body: { id: number }
 *
 * @example
 * const autoAdd = useAutoAddGroupItem();
 * autoAdd.mutate(1); // 为 groupId=1 自动添加匹配的 items
 */
// export function useAutoAddGroupItem() {
//     const queryClient = useQueryClient();

//     return useMutation({
//         mutationFn: async (groupId: number) => {
//             return apiClient.post<null>(`/api/v1/group/auto-add-item`, { id: groupId });
//         },
//         onSuccess: () => {
//             logger.log('自动添加分组 item 成功');
//             queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
//         },
//         onError: (error) => {
//             logger.error('自动添加分组 item 失败:', error);
//         },
//     });
// }

