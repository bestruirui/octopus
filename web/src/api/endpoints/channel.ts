import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from '../client';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { StatsChannel, type StatsMetricsFormatted } from './stats';
/**
 * 渠道类型枚举
 */
export enum ChannelType {
    OpenAIChat = 'openai',
    OpenAIResponse = 'openai_responses',
    Anthropic = 'anthropic',
    Gemini = 'gemini',
    Volcengine = 'volcengine',
    OpenAIEmbedding = 'openai/embeddings',
}

/**
 * 自动分组类型枚举
 */
export enum AutoGroupType {
    None = 0,   // 不自动分组
    Fuzzy = 1,  // 模糊匹配
    Exact = 2,  // 准确匹配
    Regex = 3,  // 正则匹配
}

export type BaseUrl = {
    url: string;
    delay: number;
};

export type CustomHeader = {
    header_key: string;
    header_value: string;
};

/**
 * 余额查询方式枚举（与后端 model.BalanceQueryType 对齐）
 */
export enum BalanceQueryType {
    Custom = 'custom',      // 自定义查询脚本
    DeepSeek = 'deepseek',  // DeepSeek 官方余额查询
}

/**
 * 余额查询配置（与后端 model.BalanceQuery 对齐）
 */
export type BalanceQuery = {
    enabled: boolean;
    type?: BalanceQueryType; // 查询方式,默认为 custom
    script?: string;         // 完整查询脚本:({request: {...}, extractor: function (response) {...}}),仅 custom 使用
    timeout?: number;        // 单次查询超时(秒),默认 10
    interval?: number;       // 自动查询间隔(分钟),0 表示不自动查询,默认 5
};

/**
 * 余额快照（与后端 model.Balance 对齐）
 */
export type Balance = {
    total: number;
    used: number;
    remaining: number;
    unit: string;
    plan_name?: string;
    extra?: string;
    updated_at: number;
    error?: string;
};

export type ChannelKey = {
    id: number;
    channel_id: number;
    enabled: boolean;
    channel_key: string;
    status_code: number;
    last_use_time_stamp: number;
    total_cost: number;
    remark: string;
};

/**
 * 渠道完整数据（与后端 model.Channel 对齐；数组字段在前端保证为 []）
 */
export type Channel = {
    id: number;
    name: string;
    type: ChannelType;
    enabled: boolean;
    base_urls: BaseUrl[];
    keys: ChannelKey[];
    model: string;
    custom_model: string;
    proxy: boolean;
    auto_sync: boolean;
    auto_group: AutoGroupType;
    custom_header: CustomHeader[];
    param_override?: string | null;
    channel_proxy?: string | null;
    match_regex?: string | null;
    balance_query?: BalanceQuery | null;
    balance?: Balance | null;
    stats: StatsChannel;
};

// Internal type: backend may return null for slice fields; normalize to [] in select()
type ChannelServer = Omit<Channel, 'base_urls' | 'custom_header' | 'keys'> & {
    base_urls: BaseUrl[] | null;
    custom_header: CustomHeader[] | null;
    keys: ChannelKey[] | null;
};

/**
 * 创建渠道请求：必填字段 + 可选字段
 */
export type CreateChannelRequest = {
    name: string;
    type: ChannelType;
    enabled?: boolean;
    base_urls: BaseUrl[];
    keys: Array<Pick<ChannelKey, 'enabled' | 'channel_key' | 'remark'>>;
    model: string;
    custom_model?: string;
    proxy?: boolean;
    auto_sync?: boolean;
    auto_group?: AutoGroupType;
    custom_header?: CustomHeader[];
    channel_proxy?: string | null;
    param_override?: string | null;
    match_regex?: string | null;
    balance_query?: BalanceQuery | null;
};

/**
 * 更新渠道请求：id + 可选字段 + keys diff
 */
export type UpdateChannelRequest = {
    id: number;
    name?: string;
    type?: ChannelType;
    enabled?: boolean;
    base_urls?: BaseUrl[];
    model?: string;
    custom_model?: string;
    proxy?: boolean;
    auto_sync?: boolean;
    auto_group?: AutoGroupType;
    custom_header?: CustomHeader[];
    channel_proxy?: string | null;
    param_override?: string | null;
    match_regex?: string | null;
    balance_query?: BalanceQuery | null;
    // keys diff
    keys_to_add?: Array<Pick<ChannelKey, 'enabled' | 'channel_key' | 'remark'>>;
    keys_to_update?: Array<{ id: number; enabled?: boolean; channel_key?: string; remark?: string }>;
    keys_to_delete?: number[];
};

export type FetchModelRequest = {
    type: ChannelType;
    base_urls: BaseUrl[];
    keys: Array<Pick<ChannelKey, 'enabled' | 'channel_key'>>;
    proxy?: boolean;
    channel_proxy?: string | null;
    match_regex?: string | null;
    custom_header?: CustomHeader[];
};

/**
 * 获取渠道列表 Hook
 * 
 * @example
 * const { data: channels, isLoading, error } = useChannelList();
 * 
 * if (isLoading) return <Loading />;
 * if (error) return <Error message={error.message} />;
 * 
 * channels?.forEach(channel => console.log(channel.raw.name));
 */
export function useChannelList() {
    return useQuery({
        queryKey: ['channels', 'list'],
        queryFn: () => apiRequest<ChannelServer[]>('/api/v1/channel/list'),
        select: (data) => data.map((item) => ({
            raw: ({
                ...item,
                base_urls: item.base_urls ?? [],
                custom_header: item.custom_header ?? [],
                keys: item.keys ?? [],
            }) satisfies Channel,
            formatted: {
                input_token: formatCount(item.stats.input_token),
                output_token: formatCount(item.stats.output_token),
                total_token: formatCount(item.stats.input_token + item.stats.output_token),
                input_cost: formatMoney(item.stats.input_cost),
                output_cost: formatMoney(item.stats.output_cost),
                total_cost: formatMoney(item.stats.input_cost + item.stats.output_cost),
                request_success: formatCount(item.stats.request_success),
                request_failed: formatCount(item.stats.request_failed),
                request_count: formatCount(item.stats.request_success + item.stats.request_failed),
                wait_time: formatTime(item.stats.wait_time),
            }
        })) as Array<{ raw: Channel; formatted: StatsMetricsFormatted }>,
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 创建渠道 Hook
 * 
 * @example
 * const createChannel = useCreateChannel();
 * 
 * createChannel.mutate({
 *   name: 'OpenAI',
 *   type: ChannelType.OpenAIChat,
 *   base_urls: [{ url: 'https://api.openai.com', delay: 0 }],
 *   keys: [{ enabled: true, channel_key: 'sk-xxx' }],
 *   model: 'gpt-4',
 * });
 */
export function useCreateChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateChannelRequest) =>
            apiRequest<ChannelServer>('/api/v1/channel/create', { method: 'POST', body: data }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'channel'] });
        },
    });
}

/**
 * 更新渠道 Hook
 * 
 * @example
 * const updateChannel = useUpdateChannel();
 * 
 * updateChannel.mutate({
 *   id: 1,
 *   name: 'OpenAI Updated',
 *   type: ChannelType.OpenAIChat,
 *   enabled: true,
 *   base_urls: [{ url: 'https://api.openai.com', delay: 0 }],
 *   keys_to_add: [{ enabled: true, channel_key: 'sk-xxx' }],
 *   model: 'gpt-4-turbo',
 *   proxy: false,
 * });
 */
export function useUpdateChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: UpdateChannelRequest) =>
            apiRequest<ChannelServer>('/api/v1/channel/update', { method: 'POST', body: data }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'channel'] });
        },
    });
}

/**
 * 删除渠道 Hook
 * 
 * @example
 * const deleteChannel = useDeleteChannel();
 * 
 * deleteChannel.mutate(1); // 删除 ID 为 1 的渠道
 */
export function useDeleteChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: number) =>
            apiRequest<null>(`/api/v1/channel/delete/${id}`, { method: 'DELETE' }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels', 'list'] });
            queryClient.invalidateQueries({ queryKey: ['models', 'channel'] });
        },
    });
}

/**
 * 启用/禁用渠道 Hook
 * 
 * @example
 * const enableChannel = useEnableChannel();
 * 
 * enableChannel.mutate({ id: 1, enabled: true }); // 启用 ID 为 1 的渠道
 * enableChannel.mutate({ id: 1, enabled: false }); // 禁用 ID 为 1 的渠道
 */
export function useEnableChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: { id: number; enabled: boolean }) =>
            apiRequest<null>('/api/v1/channel/enable', { method: 'POST', body: data }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['channels', 'list'] }),
    });
}

/**
 * 获取渠道模型列表 Hook
 * 
 * @example
 * const fetchModel = useFetchModel();
 * 
 * fetchModel.mutate({
 *   type: ChannelType.OpenAIChat,
 *   base_urls: [{ url: 'https://api.openai.com', delay: 0 }],
 *   keys: [{ enabled: true, channel_key: 'sk-xxx' }],
 *   proxy: false,
 * });
 * 
 * // 在 onSuccess 中获取模型列表
 * fetchModel.data // ['gpt-4', 'gpt-3.5-turbo', ...]
 */
export function useFetchModel() {
    return useMutation({
        mutationFn: (data: FetchModelRequest) =>
            apiRequest<string[]>('/api/v1/channel/fetch-model', { method: 'POST', body: data }),
    });
}

/**
 * 获取渠道最后同步时间 Hook
 * 
 * @example
 * const lastSyncTime = useLastSyncTime();
 * 
 * if (lastSyncTime) {
 *   console.log('最后同步时间:', new Date(lastSyncTime).toLocaleString());
 * }
 */
export function useLastSyncTime() {
    return useQuery({
        queryKey: ['channels', 'last-sync-time'],
        queryFn: () => apiRequest<string>('/api/v1/channel/last-sync-time'),
        refetchInterval: 30000,
    });
}
/**
 * 同步渠道 Hook
 * 
 * @example
 * const syncChannel = useSyncChannel();
 * 
 * syncChannel.mutate();
 */
export function useSyncChannel() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => apiRequest<null>('/api/v1/channel/sync', { method: 'POST' }),
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['channels', 'last-sync-time'] }),
    });
}

/**
 * 手动查询渠道余额 Hook
 *
 * @example
 * const queryBalance = useQueryBalance();
 *
 * queryBalance.mutate(1); // 立即查询渠道 1 的余额
 */
export function useQueryBalance() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: number) =>
            apiRequest<Balance>('/api/v1/channel/query-balance', { method: 'POST', body: { id } }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels', 'list'] });
        },
    });
}

/**
 * 测试余额查询脚本 Hook(用表单配置,不落库)
 *
 * @example
 * const testBalance = useTestBalance();
 *
 * testBalance.mutate({ base_urls: [...], keys: [...], balance_query: {...} });
 */
export function useTestBalance() {
    return useMutation({
        mutationFn: (data: { base_urls: BaseUrl[]; keys: Array<Pick<ChannelKey, 'enabled' | 'channel_key'>>; balance_query: BalanceQuery }) =>
            apiRequest<Balance>('/api/v1/channel/test-balance', { method: 'POST', body: data }),
    });
}
