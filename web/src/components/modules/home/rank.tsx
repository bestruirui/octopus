'use client';

import { useChannelList } from '@/api/endpoints/channel';
import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { TrendingUp } from 'lucide-react';
import { type ChannelData } from '@/api/endpoints/channel';

export function Rank() {
    const { data: channelData } = useChannelList();
    const t = useTranslations('home.rank');

    const rankedChannels = useMemo<ChannelData[]>(() => {
        if (!channelData) return [];
        return [...channelData].sort((a, b) => b.formatted.total_cost.raw - a.formatted.total_cost.raw);
    }, [channelData]);

    const getMedalEmoji = (rank: number): string => {
        switch (rank) {
            case 1: return '🥇';
            case 2: return '🥈';
            case 3: return '🥉';
            default: return '';
        }
    };

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border custom-shadow p-4">
            {rankedChannels.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="w-12 h-12 mb-3 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            ) : (
                <div className="space-y-3 max-h-[300px] overflow-y-auto">
                    {rankedChannels.map((channel, index) => {
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
                                </div>

                                <div className="flex items-baseline gap-1 text-right shrink-0">
                                    <span className="font-semibold text-base">
                                        {channel.formatted.total_cost.formatted.value}
                                        <span className="text-xs text-muted-foreground">
                                            {channel.formatted.total_cost.formatted.unit}
                                        </span>
                                    </span>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
