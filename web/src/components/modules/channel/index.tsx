'use client';

import { useChannelList } from '@/api/endpoints/channel';
import { Card } from './Card';
import { useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { useSearchableList, useChannelFilter, createChannelFilterPredicate } from '@/hooks/use-searchable-list';
import { LoadingState } from '@/components/common/LoadingState';
import { ErrorState } from '@/components/common/ErrorState';

export function Channel() {
    const { data: channelsData, isLoading, isError, refetch } = useChannelList();
    const pageKey = 'channel' as const;
    const layout = useToolbarViewOptionsStore((s) => s.getLayout(pageKey));
    const filter = useChannelFilter();

    const { visibleItems: visibleChannels } = useSearchableList({
        data: channelsData,
        pageKey,
        filter,
        filterPredicate: (item, f) => createChannelFilterPredicate(f as 'all' | 'enabled' | 'disabled')(item.raw),
    });

    return (
        <section className="relative flex h-full min-h-0 flex-col" aria-label={pageKey}>
            <div className="waterhouse-island relative flex h-full min-h-0 flex-col rounded-[2.35rem] border border-border/35 bg-card/54 p-3 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-4">
                <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_14%,color-mix(in_oklch,var(--waterhouse-highlight)_20%,transparent)_0%,transparent_30%),radial-gradient(circle_at_84%_18%,color-mix(in_oklch,var(--primary)_14%,transparent)_0%,transparent_26%),linear-gradient(180deg,color-mix(in_oklch,white_16%,transparent),transparent_22%,color-mix(in_oklch,var(--waterhouse-highlight)_8%,transparent))]" />
                <div className="relative mb-3 flex items-center gap-2 px-1">
                    <span className="h-2.5 w-10 rounded-full bg-primary/18 shadow-waterhouse-soft" />
                    <span className="h-2.5 w-24 rounded-full bg-background/65 shadow-inner" />
                    <span className="ml-auto h-2.5 w-14 rounded-full bg-background/45" />
                </div>

                <div className="relative min-h-0 flex-1">
                    {isLoading ? (
                        <LoadingState />
                    ) : isError ? (
                        <ErrorState onRetry={() => refetch()} />
                    ) : visibleChannels.length > 0 ? (
                        <VirtualizedGrid
                            items={visibleChannels}
                            layout={layout}
                            columns={{ default: 1, sm: 2, md: 2, lg: 3 }}
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
