'use client';

import { useMemo } from 'react';
import { useModelMarket, useUpdateModelPrice } from '@/api/endpoints/model';
import { ModelItem } from './Item';
import { ModelMarketSummary } from './MarketSummary';
import { useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { sortModelMarketItems } from './sort';

export function Model() {
    const { data: market } = useModelMarket();
    const updateModelPrice = useUpdateModelPrice();
    const pageKey = 'model' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const layout = useToolbarViewOptionsStore((s) => s.getLayout(pageKey));
    const filter = useToolbarViewOptionsStore((s) => s.modelFilter);
    const modelSortMode = useToolbarViewOptionsStore((s) => s.modelSortMode);

    const sortedModels = useMemo(() => {
        const items = market?.items ?? [];
        return sortModelMarketItems(items, modelSortMode);
    }, [market, modelSortMode]);

    const visibleModels = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        const byName = !term ? sortedModels : sortedModels.filter((m) => m.name.toLowerCase().includes(term));
        const hasPricing = (model: (typeof byName)[number]) =>
            model.input + model.output + model.cache_read + model.cache_write > 0;

        if (filter === 'priced') {
            return byName.filter(hasPricing);
        }
        if (filter === 'free') {
            return byName.filter((m) => !hasPricing(m));
        }

        return byName;
    }, [sortedModels, searchTerm, filter]);

    const summary = market?.summary ?? {
        model_count: 0,
        coverage_count: 0,
        unique_channel_count: 0,
        average_latency_ms: 0,
        last_update_time: '',
    };

    return (
        <section className="model-shadowless relative flex h-full min-h-0 flex-col gap-4" aria-label={pageKey}>
            <ModelMarketSummary
                summary={summary}
                onRefresh={() => updateModelPrice.mutate()}
                isRefreshing={updateModelPrice.isPending}
            />

            <section className="waterhouse-island relative flex min-h-0 flex-1 flex-col overflow-clip rounded-[2.25rem] border border-border/35 bg-card/56 p-3 text-card-foreground shadow-none backdrop-blur-[var(--waterhouse-shell-blur)] md:p-4">
                <div className="relative mb-3 flex items-center gap-2 px-1">
                    <span className="h-2.5 w-10 rounded-full bg-primary/18 shadow-waterhouse-soft" />
                    <span className="h-2.5 w-24 rounded-full bg-background/60 shadow-inner" />
                    <span className="ml-auto h-2.5 w-16 rounded-full bg-background/40" />
                </div>

                <div className="relative min-h-0 flex-1">
                    {visibleModels.length > 0 ? (
                        <VirtualizedGrid
                            items={visibleModels}
                            layout={layout}
                            columns={{ default: 1, md: 2, lg: 3 }}
                            estimateItemHeight={228}
                            getItemKey={(model) => `model-${model.name}`}
                            renderItem={(model) => <ModelItem model={model} layout={layout} />}
                        />
                    ) : (
                        <div className="waterhouse-pod relative flex h-full min-h-[18rem] items-center justify-center overflow-hidden rounded-[2rem] border border-dashed border-border/35 bg-background/28 shadow-waterhouse-soft backdrop-blur-md">
                            <div className="pointer-events-none absolute inset-x-[16%] bottom-0 h-28 rounded-t-[999px] border border-primary/12 bg-[linear-gradient(180deg,color-mix(in_oklch,var(--primary)_14%,transparent),color-mix(in_oklch,var(--waterhouse-highlight)_22%,transparent))]" />
                            <div className="relative flex items-end gap-3">
                                <span className="h-24 w-16 rounded-[1.7rem] border border-border/30 bg-background/56 shadow-waterhouse-soft" />
                                <span className="h-28 w-20 rounded-[2rem] border border-primary/18 bg-background/64 shadow-waterhouse-soft" />
                                <span className="h-20 w-14 rounded-[1.45rem] border border-border/30 bg-background/46 shadow-waterhouse-soft" />
                            </div>
                        </div>
                    )}
                </div>
            </section>
        </section>
    );
}
