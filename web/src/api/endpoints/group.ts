import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

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

