'use client';

import { useChannelList } from '@/api/endpoints/channel';
import { useStatsAPIKey } from '@/api/endpoints/stats';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Loader2, TrendingUp } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContents, TabsContent } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type RankSortMode } from '@/components/modules/home/store';

type ChannelData = NonNullable<ReturnType<typeof useChannelList>['data']>[number];
type APIKeyStatsData = NonNullable<ReturnType<typeof useStatsAPIKey>['data']>[number];
type APIKeyRankData = APIKeyStatsData & { name: string };

export function Rank() {
    const {
        data: channelData,
        isLoading: isChannelListLoading,
    } = useChannelList();
    const t = useTranslations('home.rank');
    const rankSortMode = useHomeViewStore((state) => state.rankSortMode);
    const setRankSortMode = useHomeViewStore((state) => state.setRankSortMode);
    const {
        data: apiKeyStats,
        isLoading: isAPIKeyStatsLoading,
    } = useStatsAPIKey({ enabled: rankSortMode === 'key-usage' });

    const channelsWithUsage = useMemo<ChannelData[]>(() => {
        if (!channelData) return [];
        return channelData.filter((channel) => channel.formatted.request_count.raw > 0);
    }, [channelData]);

    const rankedByCost = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.formatted.total_cost.raw - a.formatted.total_cost.raw);
    }, [channelsWithUsage]);

    const rankedByCount = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.formatted.request_count.raw - a.formatted.request_count.raw);
    }, [channelsWithUsage]);

    const rankedByTokens = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.formatted.total_token.raw - a.formatted.total_token.raw);
    }, [channelsWithUsage]);

    const rankedByKeyUsage = useMemo<APIKeyRankData[]>(() => {
        if (!apiKeyStats) return [];

        return apiKeyStats
            .filter((stats) => stats.request_count.raw > 0)
            .map((stats) => ({
                ...stats,
                name: stats.name || `Key #${stats.api_key_id}`,
            }))
            .sort((a, b) => b.request_count.raw - a.request_count.raw);
    }, [apiKeyStats]);

    const getMedalEmoji = (rank: number): string => {
        switch (rank) {
            case 1: return '🥇';
            case 2: return '🥈';
            case 3: return '🥉';
            default: return '';
        }
    };

    const renderChannelList = (channels: ChannelData[], mode: Exclude<RankSortMode, 'key-usage'>, isLoading: boolean) => {
        if (isLoading) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Loader2 className="w-12 h-12 mb-3 opacity-50 animate-spin" />
                    <p className="text-sm">{t('loading')}</p>
                </div>
            );
        }

        if (channels.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {channels.map((channel, index) => {
                    const rank = index + 1;
                    const medal = getMedalEmoji(rank);

                    return (
                        <div
                            key={channel.raw.id}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                        >
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-lg shrink-0">
                                {medal || rank}
                            </div>

                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{channel.raw.name}</p>
                                {mode === 'count' && (() => {
                                    const successCount = channel.formatted.request_success.raw;
                                    const failedCount = channel.formatted.request_failed.raw;
                                    const totalCount = successCount + failedCount;
                                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                                    return (
                                        <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                            <span>{t('successRate')}:</span>
                                            <span>{successRate.toFixed(1)}%</span>
                                        </div>
                                    );
                                })()}
                            </div>

                            <div className="flex items-center gap-1 text-right shrink-0">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {channel.formatted.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.formatted.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {channel.formatted.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.formatted.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {channel.formatted.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.formatted.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {channel.formatted.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.formatted.total_cost.formatted.unit}
                                        </span>
                                    </span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    const renderAPIKeyList = (apiKeys: APIKeyRankData[], isLoading: boolean) => {
        if (isLoading) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <Loader2 className="w-12 h-12 mb-3 opacity-50 animate-spin" />
                    <p className="text-sm">{t('loading')}</p>
                </div>
            );
        }

        if (apiKeys.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }

        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {apiKeys.map((apiKey, index) => {
                    const rank = index + 1;
                    const medal = getMedalEmoji(rank);
                    const successCount = apiKey.request_success.raw;
                    const failedCount = apiKey.request_failed.raw;
                    const totalCount = successCount + failedCount;
                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                    return (
                        <div
                            key={apiKey.api_key_id}
                            className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                        >
                            <div className="w-8 h-8 rounded-lg flex items-center justify-center font-bold text-lg shrink-0">
                                {medal || rank}
                            </div>

                            <div className="flex-1 min-w-0">
                                <p className="font-medium text-sm truncate">{apiKey.name}</p>
                                <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                    <span>{t('successRate')}:</span>
                                    <span>{successRate.toFixed(1)}%</span>
                                </div>
                            </div>

                            <div className="flex items-center gap-1 text-sm font-medium tabular-nums text-right shrink-0">
                                <span className="text-accent">
                                    {apiKey.request_success.formatted.value}
                                    <span className="text-xs text-muted-foreground">
                                        {apiKey.request_success.formatted.unit}
                                    </span>
                                </span>
                                <span className="text-muted-foreground/40 font-light">/</span>
                                <span className="text-destructive">
                                    {apiKey.request_failed.formatted.value}
                                    <span className="text-xs text-muted-foreground">
                                        {apiKey.request_failed.formatted.unit}
                                    </span>
                                </span>
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border p-4">
            <Tabs
                value={rankSortMode}
                onValueChange={(value) => {
                    setRankSortMode(value as RankSortMode);
                }}
            >
                <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <TabsList>
                        <TabsTrigger value="cost">{t('sortByCost')}</TabsTrigger>
                        <TabsTrigger value="count">{t('sortByCount')}</TabsTrigger>
                        <TabsTrigger value="tokens">{t('sortByTokens')}</TabsTrigger>
                        <TabsTrigger value="key-usage">{t('sortByKeyUsage')}</TabsTrigger>
                    </TabsList>
                </div>
                <TabsContents>
                    <TabsContent value="cost">
                        {renderChannelList(rankedByCost, 'cost', isChannelListLoading)}
                    </TabsContent>
                    <TabsContent value="count">
                        {renderChannelList(rankedByCount, 'count', isChannelListLoading)}
                    </TabsContent>
                    <TabsContent value="tokens">
                        {renderChannelList(rankedByTokens, 'tokens', isChannelListLoading)}
                    </TabsContent>
                    <TabsContent value="key-usage">
                        {renderAPIKeyList(rankedByKeyUsage, isAPIKeyStatsLoading)}
                    </TabsContent>
                </TabsContents>
            </Tabs>
        </div>
    );
}
