import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './client';
import { apiKeyDashboardStatsQueryOptions, apiKeyListQueryOptions } from './queries';
import { useAuthStore } from './user';
import { StatsAPIKey, StatsAPIKeyFormatted } from './stats';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

/**
 * API Key 数据
 */
export interface APIKey {
    id: number;
    name: string;
    api_key: string;
    enabled: boolean;
    expire_at?: number; // Unix 时间戳（秒），不传表示永不过期
    max_cost?: number; // 不传表示无限制
    max_rpm?: number; // 不传表示不限流
    supported_models?: string; // 不传表示支持所有模型
}

/**
 * API Key Stats 响应（包含 stats 和 info）
 */
export interface APIKeyStatsResponse {
    stats: StatsAPIKey;
    info: APIKey;
}

interface APIKeyStatsResponseFormatted {
    stats: StatsAPIKeyFormatted;
    info: APIKey;
}

/**
 * API Key 登录 Hook（仅校验 key 是否有效）
 */
export function useAPIKeyLogin() {
    const { setAPIKeyAuth, logout } = useAuthStore();

    return useMutation({
        mutationFn: async (apiKey: string) => {
            setAPIKeyAuth(apiKey);
            await apiRequest<null>('/api/v1/apikey/login', { dispatchUnauthorized: false });
            return apiKey;
        },
        onError: () => {
            logout();
        },
    });
}

/**
 * 获取当前 API Key 的详细统计数据 Hook（仅 API Key 登录用户使用）
 */
export function useAPIKeyDashboardStats() {
    const { isAPIKeyAuth, isAuthenticated } = useAuthStore();

    return useQuery({
        ...apiKeyDashboardStatsQueryOptions,
        select: (data): APIKeyStatsResponseFormatted => ({
            stats: {
                api_key_id: data.stats.api_key_id,
                input_token: formatCount(data.stats.input_token),
                output_token: formatCount(data.stats.output_token),
                total_token: formatCount(data.stats.input_token + data.stats.output_token),
                input_cost: formatMoney(data.stats.input_cost),
                output_cost: formatMoney(data.stats.output_cost),
                total_cost: formatMoney(data.stats.input_cost + data.stats.output_cost),
                wait_time: formatTime(data.stats.wait_time),
                request_success: formatCount(data.stats.request_success),
                request_failed: formatCount(data.stats.request_failed),
                request_count: formatCount(data.stats.request_success + data.stats.request_failed),
            },
            info: data.info,
        }),
        enabled: isAPIKeyAuth && isAuthenticated,
        refetchInterval: 30000,
    });
}

/**
 * 创建 API Key 请求
 */
type CreateAPIKeyRequest = Omit<APIKey, 'id' | 'api_key'> & { enabled?: boolean };

/**
 * 更新 API Key 请求
 */
type UpdateAPIKeyRequest = Pick<APIKey, 'id'> & CreateAPIKeyRequest;

/**
 * 获取 API Key 列表 Hook
 * 
 * @example
 * const { data: apiKeys, isLoading, error } = useAPIKeyList();
 * 
 * if (isLoading) return <Loading />;
 * if (error) return <Error message={error.message} />;
 * 
 * apiKeys?.forEach(key => console.log(key.name));
 */
export function useAPIKeyList() {
    return useQuery({
        ...apiKeyListQueryOptions,
        refetchInterval: 30000,
    });
}

/**
 * 创建 API Key Hook
 * 
 * @example
 * const createAPIKey = useCreateAPIKey();
 * 
 * createAPIKey.mutate({
 *   name: 'My API Key',
 * });
 */
export function useCreateAPIKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateAPIKeyRequest) =>
            apiRequest<APIKey>('/api/v1/apikey/create', { method: 'POST', body: data }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyListQueryOptions.queryKey }),
    });
}

/**
 * 更新 API Key Hook
 * 
 * @example
 * const updateAPIKey = useUpdateAPIKey();
 * 
 * updateAPIKey.mutate({
 *   id: 1,
 *   name: 'Updated API Key',
 *   enabled: false,
 * });
 */
export function useUpdateAPIKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: UpdateAPIKeyRequest) =>
            apiRequest<APIKey>('/api/v1/apikey/update', { method: 'POST', body: data }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyListQueryOptions.queryKey }),
    });
}

/**
 * 删除 API Key Hook
 * 
 * @example
 * const deleteAPIKey = useDeleteAPIKey();
 * 
 * deleteAPIKey.mutate(1); // 删除 ID 为 1 的 API Key
 */
export function useDeleteAPIKey() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: number) =>
            apiRequest<null>(`/api/v1/apikey/delete/${id}`, { method: 'DELETE' }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyListQueryOptions.queryKey }),
    });
}
