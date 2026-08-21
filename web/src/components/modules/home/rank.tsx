import { useChannelList } from '@/api/channel';
import { useStatsGroup, type StatsMetricsFormatted } from '@/api/stats';
import { useMemo } from 'react';
import { useTranslations } from 'use-intl';
import { TrendingUp } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { useHomeViewStore, type RankSortMode, type RankDimension } from '@/components/modules/home/store';

type ChannelData = NonNullable<ReturnType<typeof useChannelList>['data']>[number];

type RankItem = {
    key: number;
    name: string;
    formatted: StatsMetricsFormatted;
};

export function Rank() {
    const { data: channelData } = useChannelList();
    const { data: groupData } = useStatsGroup();
    const t = useTranslations('home.rank');
    const rankSortMode = useHomeViewStore((state) => state.rankSortMode);
    const setRankSortMode = useHomeViewStore((state) => state.setRankSortMode);
    const rankDimension = useHomeViewStore((state) => state.rankDimension);
    const setRankDimension = useHomeViewStore((state) => state.setRankDimension);

    const channelItems = useMemo<RankItem[]>(() => {
        if (!channelData) return [];
        return channelData.map((channel): RankItem => ({
            key: channel.raw.id,
            name: channel.raw.name,
            formatted: channel.formatted,
        }));
    }, [channelData]);

    const groupItems = useMemo<RankItem[]>(() => {
        if (!groupData) return [];
        return groupData.map((group): RankItem => ({
            key: group.group_id,
            name: group.group_name,
            formatted: group,
        }));
    }, [groupData]);

    const items = rankDimension === 'group' ? groupItems : channelItems;

    const rankedByCost = useMemo<RankItem[]>(() => {
        return [...items].sort((a, b) => b.formatted.total_cost.raw - a.formatted.total_cost.raw);
    }, [items]);

    const rankedByCount = useMemo<RankItem[]>(() => {
        return [...items].sort((a, b) => b.formatted.request_count.raw - a.formatted.request_count.raw);
    }, [items]);

    const rankedByTokens = useMemo<RankItem[]>(() => {
        return [...items].sort((a, b) => b.formatted.total_token.raw - a.formatted.total_token.raw);
    }, [items]);

    const renderList = (list: RankItem[], mode: RankSortMode) => {
        if (list.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }
        return (
            <div className="space-y-3 max-h-[300px] overflow-y-auto">
                {list.map((item, index) => {
                    const rank = index + 1;

                    return (
                        <div
                            key={item.key}
                            className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 py-3"
                        >
                            <div className="flex items-center justify-center font-bold text-lg">
                                {rank}
                            </div>

                            <div className="min-w-0">
                                <p className="font-medium text-sm truncate">{item.name}</p>
                                {mode === 'count' && (() => {
                                    const successCount = item.formatted.request_success.raw;
                                    const failedCount = item.formatted.request_failed.raw;
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

                            <div className="flex items-center gap-1 text-right">
                                {mode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {item.formatted.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {item.formatted.request_success.formatted.unit}
                                            </span>
                                        </span>
                                        <span className="text-muted-foreground/40 font-light">/</span>
                                        <span className="text-destructive">
                                            {item.formatted.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">
                                                {item.formatted.request_failed.formatted.unit}
                                            </span>
                                        </span>
                                    </div>
                                ) : mode === 'tokens' ? (
                                    <span className="font-semibold text-base">
                                        {item.formatted.total_token.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {item.formatted.total_token.formatted.unit}
                                        </span>
                                    </span>
                                ) : (
                                    <span className="font-semibold text-base">
                                        {item.formatted.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {item.formatted.total_cost.formatted.unit}
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
        <div className="rounded-3xl bg-card text-card-foreground border-border border pt-2 px-4">
            <Tabs value={rankSortMode} onValueChange={(value) => setRankSortMode(value as RankSortMode)}>
                <div className="flex items-center justify-between">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <div className="flex items-center gap-2">
                        <Tabs value={rankDimension} onValueChange={(value) => setRankDimension(value as RankDimension)}>
                            <TabsList variant="text" className="p-0">
                                <TabsTrigger value="channel" className="pr-0">{t('channel')}</TabsTrigger>
                                <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                                <TabsTrigger value="group" className="pl-0">{t('group')}</TabsTrigger>
                            </TabsList>
                        </Tabs>
                        <span aria-hidden="true" className="text-muted-foreground/30">|</span>
                        <TabsList variant="text" className="p-0">
                            <TabsTrigger value="cost" className="pr-0">{t('sortByCost')}</TabsTrigger>
                            <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                            <TabsTrigger value="count" className="px-0">{t('sortByCount')}</TabsTrigger>
                            <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                            <TabsTrigger value="tokens" className="pl-0">{t('sortByTokens')}</TabsTrigger>
                        </TabsList>
                    </div>
                </div>
                <TabsContent value="cost">
                    {renderList(rankedByCost, 'cost')}
                </TabsContent>
                <TabsContent value="count">
                    {renderList(rankedByCount, 'count')}
                </TabsContent>
                <TabsContent value="tokens">
                    {renderList(rankedByTokens, 'tokens')}
                </TabsContent>
            </Tabs>
        </div>
    );
}
