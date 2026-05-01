'use client';

import { motion } from 'motion/react';
import { BarChart3, Boxes, CircleCheckBig, Coins, DollarSign, GitBranchPlus, KeyRound, Radio, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Tabs, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import { useAnalyticsOverview } from '@/api/endpoints/analytics';
import { MetricCard, QueryState, formatPercent } from '@/components/modules/analytics/shared';
import { EASING } from '@/lib/animations/fluid-transitions';
import { formatCount, formatMoney } from '@/lib/utils';
import { useHomeViewStore, type OverviewRange } from './store';

const RANGE_OPTIONS: readonly OverviewRange[] = ['7d', '30d', '90d'];

export function HomeAnalyticsOverview() {
    const t = useTranslations('home.overview');
    const range = useHomeViewStore((state) => state.overviewRange);
    const setRange = useHomeViewStore((state) => state.setOverviewRange);
    const { data, isLoading, error } = useAnalyticsOverview(range);

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
            accentClassName: 'bg-sky-500/10 text-sky-600',
        },
        {
            title: t('metrics.totalCost'),
            value: formatMoney(data.total_cost).formatted.value,
            unit: formatMoney(data.total_cost).formatted.unit,
            icon: DollarSign,
            accentClassName: 'bg-amber-500/10 text-amber-600',
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
            accentClassName: 'bg-violet-500/10 text-violet-600',
        },
    ] : [];

    return (
        <motion.section
            className="rounded-[2rem] border border-card-border bg-card p-5 text-card-foreground custom-shadow md:p-6"
            initial={{ opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.45, ease: EASING.easeOutExpo }}
        >
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-2">
                    <div className="inline-flex items-center gap-2 rounded-full bg-primary/8 px-3 py-1 text-xs font-medium text-primary">
                        <Sparkles className="h-3.5 w-3.5" />
                        <span>{t('badge')}</span>
                    </div>
                    <div>
                        <h2 className="text-xl font-semibold tracking-tight md:text-2xl">{t('title')}</h2>
                        <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{t('description')}</p>
                    </div>
                </div>

                <Tabs value={range} onValueChange={(value) => setRange(value as OverviewRange)}>
                    <TabsList className="w-max">
                        {RANGE_OPTIONS.map((option) => (
                            <TabsTrigger key={option} value={option}>
                                {t(`range.${option}`)}
                            </TabsTrigger>
                        ))}
                    </TabsList>
                </Tabs>
            </div>

            <div className="mt-5">
                <QueryState
                    loading={isLoading}
                    error={error}
                    empty={!data}
                    emptyLabel={t('empty')}
                >
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                        {cards.map((card) => (
                            <MetricCard key={card.title} {...card} />
                        ))}
                    </div>
                </QueryState>
            </div>
        </motion.section>
    );
}
