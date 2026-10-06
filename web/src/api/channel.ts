import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './client';
import { channelStatsQueryOptions, groupListQueryOptions, modelListQueryOptions } from './queries';
import { formatStatsMetrics, type StatsMetrics, type StatsMetricsFormatted } from './stats';

// Protocol 是渠道支持的上游线协议位掩码，位值与后端 model.Protocol 一致，不可变更。
// 一条授权可同时支持多个协议，按位或组合；1 << 0 由后端保留待用。
export const Protocol = {
    OpenAIChatCompletion: 1 << 1,
    OpenAIResponse: 1 << 2,
    AnthropicMessage: 1 << 3,
} as const;

// Dialect 是上游在标准协议之上的方言，决定出站转换器的厂商特化配置。
// 地址与路径不属于方言范畴，由前端按服务商预填到渠道字段上。
export type Dialect = 'generic';

type CustomHeader = {
    header_key: string;
    header_value: string;
};

// ChannelKey 是渠道下的一份上游凭据；名称在渠道内唯一，读写都按它引用。
export type ChannelKey = {
    name: string;
    key: string;
    enabled: boolean;
};

// ChannelGrant 是渠道内的一条上游授权：指定模型使用指定凭据时支持的协议集合，也是转发的最小单位。
// 两侧按名称引用，读写同构：名称在渠道内唯一，新增的模型与凭据在后端同一事务内才分配主键，
// 故提交侧只能按名称引用，读侧也随之给名称，页面无需在主键与名称之间翻译。
// 授权本身没有启用状态: 不再授权即删掉该组合, 临时收回则停用凭据或摘掉协议位。
export type ChannelGrant = {
    model_name: string;
    key_name: string;
    protocols: number; // Protocol 位掩码。
};

// ChannelGrantCandidate 是分组页可选取的一条授权，字段与 GroupItem 的展示字段一一对应。
// 分组页由此不必拉整份渠道列表：那里带着统计、路径、代理与凭据明文，与选取成员无关。
// available 与分组成员同一口径，均由后端定稿，前后端不会各判一套。
export type ChannelGrantCandidate = {
    id: number; // 授权主键，分组成员按它引用。
    channel_id: number;
    channel_name: string;
    model_name: string;
    key_name: string;
    protocols: number; // Protocol 位掩码。
    available: boolean;
};

/**
 * 渠道完整配置（与后端 model.ChannelDetail 对齐）
 * 读写同构：编辑表单读到什么形状就提交什么形状，故创建与更新共用此类型，无需另建提交类型。
 * 提交即全量，未列出的凭据与模型会被删除并级联删除其授权；创建时 id 取 0，由后端分配。
 * keys、models、grants 和 custom_header 恒为数组，后端读取侧承诺不为 null。
 */
export type ChannelDetail = {
    id: number;
    name: string;
    dialect: Dialect;
    enabled: boolean;
    base_url: string; // 上游地址，各协议共用。
    openai_chat_completion_path: string;
    openai_response_path: string;
    anthropic_message_path: string;
    keys: ChannelKey[];
    models: string[]; // 上游模型名称；模型除名称外没有界面用得上的字段。
    grants: ChannelGrant[];
    proxy: boolean;
    custom_header: CustomHeader[];
    param_override: string;
    channel_proxy: string;
    match_regex: string;
};

// ChannelModelStats 是单个渠道模型的累计统计，自带名称。
export type ChannelModelStats = StatsMetrics & {
    model_id: number;
    model_name: string;
};

// ChannelStats 是单个渠道及其模型的累计统计，自带名称与启停状态。
// 这一份同时充当渠道列表项：列表页要展示的名称、启停与模型个数（即 models.length）都在此，
// 故没有单独的渠道概览接口，整份配置在点开编辑时由 useChannelDetail 单独取。
export type ChannelStats = StatsMetrics & {
    channel_id: number;
    channel_name: string;
    enabled: boolean;
    models: ChannelModelStats[];
};

// ChannelModelStatsFormatted 是单个渠道模型的展示用统计。
export type ChannelModelStatsFormatted = {
    model_id: number;
    model_name: string;
    formatted: StatsMetricsFormatted;
};

// ChannelStatsFormatted 是单个渠道及其模型的展示用统计，同时充当渠道列表项。
// 名称与启停随统计一并给出，列表页由此只消费这一条查询：模型个数即 models.length，
// 整份配置在点开编辑时由 useChannelDetail 单独取。
export type ChannelStatsFormatted = {
    channel_id: number;
    channel_name: string;
    enabled: boolean;
    models: ChannelModelStatsFormatted[];
    formatted: StatsMetricsFormatted;
};

// FetchModelRequest 按指定凭据试拉上游模型列表。
// 渠道尚未保存时也可试拉，故随请求携带整份渠道配置：探测用的地址、协议路径、代理、Header 与过滤表达式
// 必须和保存后生效的完全一致，直接给编辑态即可，探测用不上的字段后端忽略。
// 名称可为空：探测常发生在渠道尚未命名时，后端只要求地址非空。
// 不带协议：后端一次同时探 OpenAI 与 Anthropic 两侧，协议支持由各侧响应决定。
type FetchModelRequest = {
    channel: Omit<ChannelDetail, 'id' | 'keys' | 'models' | 'grants'>;
    key: string;
};

// FetchModel 是探测到的单个上游模型及其支持的协议集合。
// OpenAI 侧记为 Responses 而非 Chat：Chat Completions 已被官方标记弃用，需要 Chat 的渠道由用户手动勾选。
export type FetchModel = {
    name: string;
    protocols: number; // Protocol 位掩码。
};

// ChannelTestModelRequest 用当前编辑中的渠道配置与指定凭据真实请求一次上游模型。
// 配置字段与保存后生效的完全一致，与探测同一口径；不带 prompt：提示词由后端读取全局 test_prompt 设置。
// protocol 只取 Protocol 里的某一位，一次测试只走一个协议。
type ChannelTestModelRequest = {
    channel: Omit<ChannelDetail, 'id' | 'keys' | 'models' | 'grants'>;
    key: string;
    model: string;
    protocol: number;
};

// ChannelTestModelResponse 是一次模型测试的结果：上游回复正文、端到端耗时与后端实际使用的协议。
// 正文原样返回，前端按纯文本渲染；失败不走这里，由 ApiError 抛出。
export type ChannelTestModelResponse = {
    content: string;
    latency_ms: number;
    protocol: number; // Protocol 位掩码的单一位。
};

// channelGrantListQueryOptions 供分组页查询可选授权。
// 与渠道列表分开: 选取成员只需名称与可用性, 拉整份渠道会连带路径, 代理与凭据明文。
export const channelGrantListQueryOptions = queryOptions({
    queryKey: ['channels', 'grants'],
    queryFn: () => apiRequest<ChannelGrantCandidate[]>('/api/v1/channel/grants'),
});

// useChannelGrantList 获取分组页可选的全部渠道授权。
export function useChannelGrantList(enabled = true) {
    return useQuery({ ...channelGrantListQueryOptions, enabled, refetchOnMount: 'always' });
}

// channelStatsFormattedQueryOptions 统一渠道统计查询, 格式化和刷新策略。
// 与配置分开刷新: 统计每次转发都在变, 配置只在人工改动后由 mutation 失效。
const channelStatsFormattedQueryOptions = queryOptions({
    ...channelStatsQueryOptions,
    select: (data) => data.map((item): ChannelStatsFormatted => ({
        channel_id: item.channel_id,
        channel_name: item.channel_name,
        enabled: item.enabled,
        models: item.models.map((channelModel) => ({
            model_id: channelModel.model_id,
            model_name: channelModel.model_name,
            formatted: formatStatsMetrics(channelModel),
        })),
        formatted: formatStatsMetrics(item),
    })),
    refetchInterval: 30000,
    refetchOnMount: 'always',
});

// useChannelStats 获取全部渠道及其模型的展示用统计, 也是渠道列表页的数据来源。
export function useChannelStats(enabled = true) {
    return useQuery({ ...channelStatsFormattedQueryOptions, enabled });
}

/**
 * 获取单个渠道完整配置 Hook, 供编辑表单打开时读取; id 为空时不发请求。
 * 不随统计一并取回: 整份配置带着路径、代理与凭据明文, 只有正在编辑的那一个渠道用得上。
 *
 * @example
 * const { data: detail } = useChannelDetail(channelId);
 */
export function useChannelDetail(id?: number) {
    return useQuery({
        queryKey: ['channels', 'detail', id],
        queryFn: () => apiRequest<ChannelDetail>(`/api/v1/channel/detail/${id}`),
        enabled: id !== undefined,
        refetchOnMount: 'always',
    });
}

/**
 * 创建渠道 Hook；提交整份配置，id 取 0 由后端分配。
 * 授权与凭据、模型在同一请求提交：授权按名称引用两侧，后端在同一事务内解析为主键。
 *
 * @example
 * const createChannel = useCreateChannel();
 *
 * createChannel.mutate({
 *   id: 0,
 *   name: 'OpenAI',
 *   base_url: 'https://api.openai.com',
 *   keys: [{ name: 'default', key: 'sk-xxx', enabled: true }],
 *   models: ['gpt-4o'],
 *   grants: [{ model_name: 'gpt-4o', key_name: 'default', protocols: Protocol.OpenAIResponse }],
 *   // ...其余配置字段
 * });
 */
export function useCreateChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: ChannelDetail) =>
            apiRequest<ChannelDetail>('/api/v1/channel/create', { method: 'POST', body: data }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels'] });
            queryClient.invalidateQueries({ queryKey: modelListQueryOptions.queryKey });
        },
    });
}

/**
 * 更新渠道 Hook；提交整份配置，整体替换。
 * keys、models 和 grants 提交即覆盖：未列出的凭据和模型会被删除并级联删除其授权。
 * grants 的 protocols 不能为 0 或含未定义位，名称也必须属于同一渠道，否则后端整单拒绝。
 *
 * @example
 * const updateChannel = useUpdateChannel();
 *
 * updateChannel.mutate({ ...detail, enabled: false });
 */
export function useUpdateChannel() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: ChannelDetail) =>
            apiRequest<ChannelDetail>('/api/v1/channel/update', { method: 'POST', body: data }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels'] });
            queryClient.invalidateQueries({ queryKey: modelListQueryOptions.queryKey });
            queryClient.invalidateQueries({ queryKey: groupListQueryOptions.queryKey });
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
            queryClient.invalidateQueries({ queryKey: ['channels'] });
            queryClient.invalidateQueries({ queryKey: modelListQueryOptions.queryKey });
            queryClient.invalidateQueries({ queryKey: groupListQueryOptions.queryKey });
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
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['channels'] });
            // 渠道启停会改变其授权在分组内的可用性，成员列表要跟着刷新。
            queryClient.invalidateQueries({ queryKey: groupListQueryOptions.queryKey });
        },
    });
}

/**
 * 获取渠道模型列表 Hook
 *
 * @example
 * const fetchModel = useFetchModel();
 *
 * fetchModel.mutate({
 *   channel: { base_url: 'https://api.openai.com', openai_response_path: '/v1/responses', ... },
 *   key: 'sk-xxx',
 * });
 *
 * // 在 onSuccess 中获取模型列表
 * fetchModel.data // [{ name: 'gpt-4o', protocols: 4 }, ...]
 */
export function useFetchModel() {
    return useMutation({
        mutationFn: (data: FetchModelRequest) =>
            apiRequest<FetchModel[]>('/api/v1/channel/fetch-model', { method: 'POST', body: data }),
    });
}

/**
 * 测试渠道模型 Hook；用未保存的当前配置与指定凭据真实请求一次上游。
 * 提示词由后端读取全局 test_prompt 设置，前端不传；失败以 ApiError 抛出，由调用方展示。
 * 只做一次请求，不重试也不回落到其他协议。
 *
 * @example
 * const testModel = useTestModel();
 *
 * testModel.mutate({
 *   channel: { base_url: 'https://api.openai.com', openai_response_path: '/v1/responses', ... },
 *   key: 'sk-xxx',
 *   model: 'gpt-4o',
 *   protocol: Protocol.OpenAIResponse,
 * });
 */
export function useTestModel() {
    return useMutation({
        mutationFn: (data: ChannelTestModelRequest) =>
            apiRequest<ChannelTestModelResponse>('/api/v1/channel/test-model', { method: 'POST', body: data }),
    });
}
