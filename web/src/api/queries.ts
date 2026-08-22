import { queryOptions } from '@tanstack/react-query';
import type { APIKey, APIKeyStatsResponse } from './apikey';
import type { ChannelServer } from './channel';
import type { Group } from './group';
import type { LLMChannel, LLMInfo } from './model';
import type { StatsDailyResponse, StatsGroup, StatsHourly, StatsTotal } from './stats';
import { apiRequest } from './client';

// apiKeyDashboardStatsQueryOptions 供页面查询和启动预取共享 API Key 统计定义。
export const apiKeyDashboardStatsQueryOptions = queryOptions({
    queryKey: ['apikey', 'dashboard', 'stats'],
    queryFn: () => apiRequest<APIKeyStatsResponse>('/api/v1/apikey/stats'),
});

// apiKeyListQueryOptions 供页面查询和启动预取共享 API Key 列表定义。
export const apiKeyListQueryOptions = queryOptions({
    queryKey: ['apikeys', 'list'],
    queryFn: () => apiRequest<APIKey[]>('/api/v1/apikey/list'),
});

// channelListQueryOptions 供页面查询和启动预取共享渠道列表定义。
export const channelListQueryOptions = queryOptions({
    queryKey: ['channels', 'list'],
    queryFn: () => apiRequest<ChannelServer[]>('/api/v1/channel/list'),
});

// groupListQueryOptions 供页面查询和启动预取共享分组列表定义。
export const groupListQueryOptions = queryOptions({
    queryKey: ['groups', 'list'],
    queryFn: () => apiRequest<Group[]>('/api/v1/group/list'),
});

// modelListQueryOptions 供页面查询和启动预取共享模型列表定义。
export const modelListQueryOptions = queryOptions({
    queryKey: ['models', 'list'],
    queryFn: () => apiRequest<LLMInfo[]>('/api/v1/model/list'),
});

// modelChannelListQueryOptions 供页面查询和启动预取共享模型渠道关联定义。
export const modelChannelListQueryOptions = queryOptions({
    queryKey: ['models', 'channel'],
    queryFn: () => apiRequest<LLMChannel[]>('/api/v1/model/channel'),
});

// statsDailyQueryOptions 供页面查询和启动预取共享每日统计定义。
export const statsDailyQueryOptions = queryOptions({
    queryKey: ['stats', 'daily'],
    queryFn: () => apiRequest<StatsDailyResponse>('/api/v1/stats/daily'),
});

// statsHourlyQueryOptions 供页面查询和启动预取共享每小时统计定义。
export const statsHourlyQueryOptions = queryOptions({
    queryKey: ['stats', 'hourly'],
    queryFn: () => apiRequest<StatsHourly[]>('/api/v1/stats/hourly'),
});

// statsTotalQueryOptions 供页面查询和启动预取共享总计统计定义。
export const statsTotalQueryOptions = queryOptions({
    queryKey: ['stats', 'total'],
    queryFn: () => apiRequest<StatsTotal>('/api/v1/stats/total'),
});

// statsGroupQueryOptions 供页面查询和启动预取共享分组统计定义。
export const statsGroupQueryOptions = queryOptions({
    queryKey: ['stats', 'group'],
    queryFn: () => apiRequest<StatsGroup[]>('/api/v1/stats/group'),
});
