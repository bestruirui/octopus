'use client';

import type { ReactNode } from 'react';
import { KeyRound, Layers3, Radio } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAnalyticsUtilization, type AnalyticsRange, type AnalyticsProviderBreakdownItem, type AnalyticsModelBreakdownItem, type AnalyticsAPIKeyBreakdownItem } from '@/api/endpoints/analytics';
import { formatCount, formatMoney } from '@/lib/utils';
import { QueryState, StatusBadge, formatPercent } from './shared';

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
        <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
            <div className="mb-4 flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                    <Icon className="h-4 w-4" />
                </div>
                <h4 className="text-sm font-semibold">{title}</h4>
            </div>

            <div className="max-h-80 space-y-3 overflow-y-auto pr-1">
                {items.map((item) => (
                    <div key={`${title}-${getName(item)}`} className="rounded-2xl border border-border/40 bg-card px-3 py-3">
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
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('cards.utilization.title')}</h3>
                <p className="text-sm leading-6 text-muted-foreground">{t('utilization.description')}</p>
            </div>
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
        </section>
    );
}
