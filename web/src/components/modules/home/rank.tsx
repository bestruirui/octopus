'use client';

import { useChannelList } from '@/api/endpoints/channel';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { TrendingUp, CheckCircle2, XCircle } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContents, TabsContent } from '@/components/animate-ui/components/animate/tabs';

type SortMode = 'cost' | 'count';
type ChannelData = NonNullable<ReturnType<typeof useChannelList>['data']>[number];

export function Rank() {
    const { data: channelData } = useChannelList();
    const t = useTranslations('home.rank');

    const rankedByCost = useMemo<ChannelData[]>(() => {
        if (!channelData) return [];
        return [...channelData].sort((a, b) => b.formatted.total_cost.raw - a.formatted.total_cost.raw);
    }, [channelData]);

    const rankedByCount = useMemo<ChannelData[]>(() => {
        if (!channelData) return [];
        return [...channelData].sort((a, b) => b.formatted.request_count.raw - a.formatted.request_count.raw);
    }, [channelData]);

    const getMedalEmoji = (rank: number): string => {
        switch (rank) {
            case 1: return '🥇';
            case 2: return '🥈';
            case 3: return '🥉';
            default: return '';
        }
    };

    const renderList = (channels: ChannelData[], mode: SortMode) => {
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
                    const successCount = channel.formatted.request_success.raw;
                    const failedCount = channel.formatted.request_failed.raw;
                    const totalCount = successCount + failedCount;
                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

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
                                {mode === 'count' && (
                                    <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                                        <span>{t('successRate')}:</span>
                                        <span>{successRate.toFixed(1)}%</span>
                                    </div>
                                )}
                            </div>

                            <div className="flex items-center gap-1 text-right shrink-0">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-3 text-sm">
                                        <div className="flex items-center gap-1 text-accent">
                                            <CheckCircle2 className="w-3.5 h-3.5" />
                                            <span>{channel.formatted.request_success.formatted.value}</span>
                                        </div>
                                        <div className="flex items-center gap-1 text-destructive">
                                            <XCircle className="w-3.5 h-3.5" />
                                            <span>{channel.formatted.request_failed.formatted.value}</span>
                                        </div>
                                    </div>
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

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border custom-shadow p-4">
            <Tabs defaultValue="cost">
                <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <TabsList>
                        <TabsTrigger value="cost">{t('sortByCost')}</TabsTrigger>
                        <TabsTrigger value="count">{t('sortByCount')}</TabsTrigger>
                    </TabsList>
                </div>
                <TabsContents>
                    <TabsContent value="cost">
                        {renderList(rankedByCost, 'cost')}
                    </TabsContent>
                    <TabsContent value="count">
                        {renderList(rankedByCount, 'count')}
                    </TabsContent>
                </TabsContents>
            </Tabs>
        </div>
    );
}
