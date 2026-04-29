'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { type ModelMarketItem, useModelMarket, useUpdateModelPrice } from '@/api/endpoints/model';
import { toast } from '@/components/common/Toast';
import { ModelItem } from './Item';
import { ModelMarketSummary } from './MarketSummary';
import { useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';

function buildVisibleSummary(models: ModelMarketItem[], lastUpdateTime: string) {
    const uniqueChannels = new Set<number>();
    let coverageCount = 0;
    let weightedLatencyTotal = 0;
    let requestCountTotal = 0;

    models.forEach((model) => {
        coverageCount += model.channel_count;
        const requestCount = model.request_success + model.request_failed;
        requestCountTotal += requestCount;
        weightedLatencyTotal += model.average_latency_ms * requestCount;

        model.channels.forEach((channel) => uniqueChannels.add(channel.channel_id));
    });

    return {
        model_count: models.length,
        coverage_count: coverageCount,
        unique_channel_count: uniqueChannels.size,
        average_latency_ms: requestCountTotal > 0 ? Math.round(weightedLatencyTotal / requestCountTotal) : 0,
        last_update_time: lastUpdateTime,
    };
}

export function Model() {
    const t = useTranslations('model');
    const { data, error, isLoading } = useModelMarket();
    const updatePrice = useUpdateModelPrice();
    const pageKey = 'model' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const layout = useToolbarViewOptionsStore((s) => s.getLayout(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));
    const filter = useToolbarViewOptionsStore((s) => s.modelFilter);
    const lastUpdateTime = data?.summary.last_update_time ?? '';

    const sortedModels = useMemo(() => {
        const models = data?.items ?? [];
        return [...models].sort((a, b) =>
            sortOrder === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name)
        );
    }, [data?.items, sortOrder]);

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

    const visibleSummary = useMemo(
        () => buildVisibleSummary(visibleModels, lastUpdateTime),
        [lastUpdateTime, visibleModels]
    );

    const handleRefresh = () => {
        updatePrice.mutate(undefined, {
            onSuccess: () => {
                toast.success(t('summary.refreshSuccess'));
            },
            onError: (refreshError) => {
                toast.error(t('summary.refreshFailed'), { description: refreshError.message });
            },
        });
    };

    return (
        <div className="flex h-full min-h-0 flex-col gap-4">
            <ModelMarketSummary
                summary={visibleSummary}
                onRefresh={handleRefresh}
                isRefreshing={updatePrice.isPending}
            />

            <div className="min-h-0 flex-1">
                {error ? (
                    <div className="flex h-full items-center justify-center rounded-3xl border border-card-border bg-card p-6 text-destructive custom-shadow">
                        {error.message}
                    </div>
                ) : visibleModels.length === 0 ? (
                    <div className="flex h-full items-center justify-center rounded-3xl border border-card-border bg-card p-6 text-sm text-muted-foreground custom-shadow">
                        {isLoading ? t('loading') : t('empty')}
                    </div>
                ) : (
                    <VirtualizedGrid
                        items={visibleModels}
                        layout={layout}
                        columns={{ default: 1, md: 2, lg: 3 }}
                        estimateItemHeight={248}
                        getItemKey={(model) => `model-${model.name}`}
                        renderItem={(model) => <ModelItem model={model} layout={layout} />}
                    />
                )}
            </div>
        </div>
    );
}
