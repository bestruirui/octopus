'use client';

import { useMemo } from 'react';
import { useChannelList } from '@/api/endpoints/channel';
import { Card } from './Card';
import { useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';

export function Channel() {
    const { data: channelsData } = useChannelList();
    const pageKey = 'channel' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const layout = useToolbarViewOptionsStore((s) => s.getLayout(pageKey));
    const sortField = useToolbarViewOptionsStore((s) => s.getSortField(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));
    const filter = useToolbarViewOptionsStore((s) => s.channelFilter);

    const sortedChannels = useMemo(() => {
        if (!channelsData) return [];
        return [...channelsData].sort((a, b) => {
            const diff = sortField === 'name'
                ? a.raw.name.localeCompare(b.raw.name)
                : a.raw.id - b.raw.id;
            return sortOrder === 'asc' ? diff : -diff;
        });
    }, [channelsData, sortField, sortOrder]);

    const visibleChannels = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        const byName = !term ? sortedChannels : sortedChannels.filter((c) => c.raw.name.toLowerCase().includes(term));

        if (filter === 'enabled') return byName.filter((c) => c.raw.enabled);
        if (filter === 'disabled') return byName.filter((c) => !c.raw.enabled);

        return byName;
    }, [sortedChannels, searchTerm, filter]);

    return (
        <section className="relative flex h-full min-h-0 flex-col" aria-label={pageKey}>
            <div className="waterhouse-island relative flex h-full min-h-0 flex-col overflow-hidden rounded-[2.35rem] border border-border/35 bg-card/54 p-3 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-4">
                <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_14%,color-mix(in_oklch,var(--waterhouse-highlight)_20%,transparent)_0%,transparent_30%),radial-gradient(circle_at_84%_18%,color-mix(in_oklch,var(--primary)_14%,transparent)_0%,transparent_26%),linear-gradient(180deg,color-mix(in_oklch,white_16%,transparent),transparent_22%,color-mix(in_oklch,var(--waterhouse-highlight)_8%,transparent))]" />
                <div className="relative mb-3 flex items-center gap-2 px-1">
                    <span className="h-2.5 w-10 rounded-full bg-primary/18 shadow-waterhouse-soft" />
                    <span className="h-2.5 w-24 rounded-full bg-background/65 shadow-inner" />
                    <span className="ml-auto h-2.5 w-14 rounded-full bg-background/45" />
                </div>

                <div className="relative min-h-0 flex-1">
                    {visibleChannels.length > 0 ? (
                        <VirtualizedGrid
                            items={visibleChannels}
                            layout={layout}
                            columns={{ default: 1, md: 2, lg: 3 }}
                            estimateItemHeight={232}
                            getItemKey={(item) => `channel-${item.raw.id}`}
                            renderItem={(item) => <Card channel={item.raw} stats={item.formatted} layout={layout} />}
                        />
                    ) : (
                        <div className="waterhouse-pod relative flex h-full min-h-[18rem] items-center justify-center overflow-hidden rounded-[2rem] border border-dashed border-border/35 bg-background/28 shadow-waterhouse-soft backdrop-blur-md">
                            <div className="pointer-events-none absolute inset-x-[12%] bottom-0 h-28 rounded-t-[999px] border border-primary/10 bg-[linear-gradient(180deg,color-mix(in_oklch,var(--primary)_16%,transparent),color-mix(in_oklch,var(--waterhouse-highlight)_20%,transparent))] blur-[1px]" />
                            <div className="relative flex items-center gap-3">
                                <span className="h-14 w-14 rounded-[1.45rem] border border-border/35 bg-background/56 shadow-waterhouse-soft" />
                                <span className="h-20 w-20 rounded-[2rem] border border-primary/20 bg-primary/10 shadow-waterhouse-soft" />
                                <span className="h-12 w-12 rounded-[1.2rem] border border-border/30 bg-background/42 shadow-waterhouse-soft" />
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </section>
    );
}
