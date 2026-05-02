'use client';

import type { ReactNode } from 'react';
import { KeyRound, Layers3, Radio, Waves } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAnalyticsUtilization, type AnalyticsRange, type AnalyticsProviderBreakdownItem, type AnalyticsModelBreakdownItem, type AnalyticsAPIKeyBreakdownItem } from '@/api/endpoints/analytics';
import { formatCount, formatMoney } from '@/lib/utils';
import { ObservatorySection, QueryState, StatusBadge, formatPercent } from './shared';

type BreakdownItem = AnalyticsProviderBreakdownItem | AnalyticsModelBreakdownItem | AnalyticsAPIKeyBreakdownItem;

function BreakdownCard({
    title,
    icon: Icon,
    items,
    getName,
    getMeta,
}: {
    title: string;
    icon: typeof Radio;
    items: BreakdownItem[];
    getName: (item: BreakdownItem) => string;
    getMeta?: (item: BreakdownItem) => ReactNode;
}) {
    return (
        <article className="waterhouse-pod rounded-[1.75rem] border border-border/30 bg-background/40 p-4 shadow-waterhouse-soft backdrop-blur-md">
            <div className="mb-4 flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-[1.05rem] bg-primary/10 text-primary shadow-waterhouse-soft">
                    <Icon className="h-4 w-4" />
                </div>
                <h4 className="text-sm font-semibold">{title}</h4>
            </div>

            <div className="max-h-80 space-y-3 overflow-y-auto pr-1">
                {items.map((item) => (
                    <div key={`${title}-${getName(item)}`} className="waterhouse-pod rounded-[1.35rem] border border-border/25 bg-background/48 px-3 py-3 shadow-waterhouse-soft">
                        <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0">
                                <div className="truncate text-sm font-medium">{getName(item)}</div>
                                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                                    <span>{formatCount(item.request_count).formatted.value}{formatCount(item.request_count).formatted.unit}</span>
                                    <span>{formatPercent(item.success_rate).formatted.value}%</span>
                                    {getMeta ? getMeta(item) : null}
                                </div>
                            </div>
                            <div className="shrink-0 text-right">
                                <div className="text-sm font-semibold">
                                    {formatMoney(item.total_cost).formatted.value}
                                    <span className="ml-0.5 text-xs text-muted-foreground">{formatMoney(item.total_cost).formatted.unit}</span>
                                </div>
                                <div className="mt-1 text-xs text-muted-foreground">
                                    {formatCount(item.total_tokens).formatted.value}
                                    <span className="ml-0.5">{formatCount(item.total_tokens).formatted.unit}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </article>
    );
}

export function Utilization({ range }: { range: AnalyticsRange }) {
    const t = useTranslations('analytics');
    const { data, isLoading, error } = useAnalyticsUtilization(range);

    const isEmpty =
        !data ||
        (
            data.provider_breakdown.length === 0 &&
            data.model_breakdown.length === 0 &&
            data.apikey_breakdown.length === 0
        );

    return (
        <ObservatorySection
            eyebrow={t('cards.utilization.title')}
            title={t('cards.utilization.title')}
            description={t('utilization.description')}
            icon={Waves}
        >
            <QueryState
                loading={isLoading}
                error={error}
                empty={isEmpty}
                emptyLabel={isLoading ? t('states.loading') : t('utilization.empty')}
            >
                <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
                    <BreakdownCard
                        title={t('utilization.providers')}
                        icon={Radio}
                        items={data?.provider_breakdown ?? []}
                        getName={(item) => (item as AnalyticsProviderBreakdownItem).channel_name}
                        getMeta={(item) => {
                            const provider = item as AnalyticsProviderBreakdownItem;
                            return (
                                <StatusBadge
                                    label={provider.enabled ? t('utilization.enabled') : t('utilization.disabled')}
                                    tone={provider.enabled ? 'success' : 'neutral'}
                                />
                            );
                        }}
                    />
                    <BreakdownCard
                        title={t('utilization.models')}
                        icon={Layers3}
                        items={data?.model_breakdown ?? []}
                        getName={(item) => (item as AnalyticsModelBreakdownItem).model_name}
                    />
                    <BreakdownCard
                        title={t('utilization.apikeys')}
                        icon={KeyRound}
                        items={data?.apikey_breakdown ?? []}
                        getName={(item) => (item as AnalyticsAPIKeyBreakdownItem).name}
                    />
                </div>
            </QueryState>
        </ObservatorySection>
    );
}
