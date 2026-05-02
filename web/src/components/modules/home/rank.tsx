'use client';

import { useStatsAPIKey, useStatsChannel } from '@/api/endpoints/stats';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Leaf, Loader2, TrendingUp, Trophy } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContents, TabsContent } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type RankSortMode } from '@/components/modules/home/store';
import { cn } from '@/lib/utils';

type ChannelData = NonNullable<ReturnType<typeof useStatsChannel>['data']>[number];
type APIKeyStatsData = NonNullable<ReturnType<typeof useStatsAPIKey>['data']>[number];
type APIKeyRankData = APIKeyStatsData & { name: string };

export function Rank() {
    const {
        data: channelData,
        isLoading: isChannelListLoading,
    } = useStatsChannel();
    const t = useTranslations('home.rank');
    const rankSortMode = useHomeViewStore((state) => state.rankSortMode);
    const setRankSortMode = useHomeViewStore((state) => state.setRankSortMode);
    const {
        data: apiKeyStats,
        isLoading: isAPIKeyStatsLoading,
    } = useStatsAPIKey({ enabled: rankSortMode === 'key-usage' });

    const channelsWithUsage = useMemo<ChannelData[]>(() => {
        if (!channelData) return [];
        return channelData.filter((channel) => channel.request_count.raw > 0);
    }, [channelData]);

    const rankedByCost = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.total_cost.raw - a.total_cost.raw);
    }, [channelsWithUsage]);

    const rankedByCount = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.request_count.raw - a.request_count.raw);
    }, [channelsWithUsage]);

    const rankedByTokens = useMemo<ChannelData[]>(() => {
        return [...channelsWithUsage].sort((a, b) => b.total_token.raw - a.total_token.raw);
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

    const getRankToneClass = (rank: number): string => {
        if (rank === 1) return 'border-primary/30 bg-primary/14 text-primary';
        if (rank === 2) return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
        if (rank === 3) return 'border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300';
        return 'border-border/35 bg-background/45 text-muted-foreground';
    };

    const renderChannelList = (channels: ChannelData[], mode: Exclude<RankSortMode, 'key-usage'>, isLoading: boolean) => {
        if (isLoading) {
            return (
                <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center rounded-[1.7rem] border-border/30 bg-background/36 py-8 text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                    <Loader2 className="mb-3 h-10 w-10 animate-spin opacity-50" />
                    <p className="text-sm">{t('loading')}</p>
                </div>
            );
        }

        if (channels.length === 0) {
            return (
                <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center rounded-[1.7rem] border-border/30 bg-background/36 py-8 text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                    <TrendingUp className="mb-3 h-10 w-10 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="max-h-[328px] space-y-2.5 overflow-y-auto pr-1 [scrollbar-width:thin]">
                {channels.map((channel, index) => {
                    const rank = index + 1;

                    return (
                        <div
                            key={channel.channel_id}
                            className={cn(
                                'waterhouse-pod group relative flex items-center gap-3 overflow-hidden rounded-[1.45rem] border-border/25 bg-background/34 p-2.5 shadow-waterhouse-soft backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:border-primary/18 hover:shadow-[var(--waterhouse-shadow-soft)]',
                                rank <= 3 && 'bg-background/42',
                            )}
                        >
                            <div className="pointer-events-none absolute -right-10 top-1/2 h-20 w-20 -translate-y-1/2 rounded-full bg-primary/6 blur-2xl opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                            <div className={cn('relative flex h-9 w-9 shrink-0 items-center justify-center rounded-[1.1rem] border text-sm font-bold shadow-sm', getRankToneClass(rank))}>
                                {rank === 1 ? <Trophy className="h-4 w-4" /> : rank}
                            </div>

                            <div className="relative min-w-0 flex-1">
                                <p className="truncate text-sm font-semibold">{channel.channel_name}</p>
                                {mode === 'count' && (() => {
                                    const successCount = channel.request_success.raw;
                                    const failedCount = channel.request_failed.raw;
                                    const totalCount = successCount + failedCount;
                                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                                    return (
                                        <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                                            <span>{t('successRate')}:</span>
                                            <span>{successRate.toFixed(1)}%</span>
                                        </div>
                                    );
                                })()}
                            </div>

                            <div className="relative flex shrink-0 items-center gap-1 text-right">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {channel.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {channel.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {channel.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {channel.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {channel.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.total_cost.formatted.unit}
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
                <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center rounded-[1.7rem] border-border/30 bg-background/36 py-8 text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                    <Loader2 className="mb-3 h-10 w-10 animate-spin opacity-50" />
                    <p className="text-sm">{t('loading')}</p>
                </div>
            );
        }

        if (apiKeys.length === 0) {
            return (
                <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center rounded-[1.7rem] border-border/30 bg-background/36 py-8 text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                    <TrendingUp className="mb-3 h-10 w-10 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }

        return (
            <div className="max-h-[328px] space-y-2.5 overflow-y-auto pr-1 [scrollbar-width:thin]">
                {apiKeys.map((apiKey, index) => {
                    const rank = index + 1;
                    const successCount = apiKey.request_success.raw;
                    const failedCount = apiKey.request_failed.raw;
                    const totalCount = successCount + failedCount;
                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                    return (
                        <div
                            key={apiKey.api_key_id}
                            className={cn(
                                'waterhouse-pod group relative flex items-center gap-3 overflow-hidden rounded-[1.45rem] border-border/25 bg-background/34 p-2.5 shadow-waterhouse-soft backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:border-primary/18 hover:shadow-[var(--waterhouse-shadow-soft)]',
                                rank <= 3 && 'bg-background/42',
                            )}
                        >
                            <div className="pointer-events-none absolute -right-10 top-1/2 h-20 w-20 -translate-y-1/2 rounded-full bg-primary/6 blur-2xl opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                            <div className={cn('relative flex h-9 w-9 shrink-0 items-center justify-center rounded-[1.1rem] border text-sm font-bold shadow-sm', getRankToneClass(rank))}>
                                {rank === 1 ? <Trophy className="h-4 w-4" /> : rank}
                            </div>

                            <div className="relative min-w-0 flex-1">
                                <p className="truncate text-sm font-semibold">{apiKey.name}</p>
                                <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <span>{t('successRate')}:</span>
                                    <span>{successRate.toFixed(1)}%</span>
                                </div>
                            </div>

                            <div className="relative flex shrink-0 items-center gap-1 text-right text-sm font-medium tabular-nums">
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
        <div className="waterhouse-pod relative h-full overflow-hidden rounded-[1.85rem] border-border/28 bg-card/42 p-3.5 text-card-foreground shadow-waterhouse-soft backdrop-blur-[var(--waterhouse-shell-blur)] md:p-4">
            <div className="pointer-events-none absolute -right-12 -top-12 h-32 w-32 rounded-full bg-primary/6 blur-3xl" />
            <div className="pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-primary/18 to-transparent" />
            <Tabs
                value={rankSortMode}
                onValueChange={(value) => {
                    setRankSortMode(value as RankSortMode);
                }}
            >
                <div className="relative flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="inline-flex w-max items-center gap-2 rounded-full border border-primary/10 bg-background/34 px-2.5 py-1 text-[0.64rem] font-semibold uppercase tracking-[0.18em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                        <Leaf className="h-3.5 w-3.5" />
                        <span>{t('title')}</span>
                    </div>
                    <TabsList className="waterhouse-pod flex w-full flex-wrap rounded-[1.25rem] border-border/25 bg-background/30 p-1 shadow-waterhouse-soft backdrop-blur-md lg:w-max">
                        <TabsTrigger value="cost">{t('sortByCost')}</TabsTrigger>
                        <TabsTrigger value="count">{t('sortByCount')}</TabsTrigger>
                        <TabsTrigger value="tokens">{t('sortByTokens')}</TabsTrigger>
                        <TabsTrigger value="key-usage">{t('sortByKeyUsage')}</TabsTrigger>
                    </TabsList>
                </div>
                <TabsContents className="relative mt-4">
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
