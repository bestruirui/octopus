'use client';

import { BarChart3, Boxes, CircleCheckBig, Coins, DollarSign, GitBranchPlus, KeyRound, Radio } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAnalyticsOverview, type AnalyticsRange } from '@/api/endpoints/analytics';
import { formatCount, formatMoney } from '@/lib/utils';
import { MetricCard, QueryState, formatPercent } from './shared';

export function Overview({ range }: { range: AnalyticsRange }) {
    const t = useTranslations('analytics');
    const { data, isLoading, error } = useAnalyticsOverview(range);
    const description = t('overview.description');

    const cards = data ? [
        {
            title: t('metrics.requestCount'),
            value: formatCount(data.request_count).formatted.value,
            unit: formatCount(data.request_count).formatted.unit,
            icon: BarChart3,
        },
        {
            title: t('metrics.successRate'),
            value: formatPercent(data.success_rate).formatted.value,
            unit: formatPercent(data.success_rate).formatted.unit,
            icon: CircleCheckBig,
            accentClassName: 'bg-emerald-500/10 text-emerald-600',
        },
        {
            title: t('metrics.totalTokens'),
            value: formatCount(data.total_tokens).formatted.value,
            unit: formatCount(data.total_tokens).formatted.unit,
            icon: Coins,
        },
        {
            title: t('metrics.totalCost'),
            value: formatMoney(data.total_cost).formatted.value,
            unit: formatMoney(data.total_cost).formatted.unit,
            icon: DollarSign,
            accentClassName: 'bg-chart-1/10 text-chart-1',
        },
        {
            title: t('metrics.providerCount'),
            value: formatCount(data.provider_count).formatted.value,
            unit: formatCount(data.provider_count).formatted.unit,
            icon: Radio,
        },
        {
            title: t('metrics.apiKeyCount'),
            value: formatCount(data.api_key_count).formatted.value,
            unit: formatCount(data.api_key_count).formatted.unit,
            icon: KeyRound,
        },
        {
            title: t('metrics.modelCount'),
            value: formatCount(data.model_count).formatted.value,
            unit: formatCount(data.model_count).formatted.unit,
            icon: Boxes,
        },
        {
            title: t('metrics.fallbackRate'),
            value: formatPercent(data.fallback_rate).formatted.value,
            unit: formatPercent(data.fallback_rate).formatted.unit,
            icon: GitBranchPlus,
            accentClassName: 'bg-chart-4/10 text-chart-4',
        },
    ] : [];

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('cards.overview.title')}</h3>
                {description ? (
                    <p className="text-sm leading-6 text-muted-foreground">{description}</p>
                ) : null}
            </div>
            <QueryState
                loading={isLoading}
                error={error}
                empty={!data}
                emptyLabel={t('states.loading')}
            >
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                    {cards.map((card) => (
                        <MetricCard key={card.title} {...card} />
                    ))}
                </div>
            </QueryState>
        </section>
    );
}
