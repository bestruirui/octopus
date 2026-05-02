'use client';

import { motion } from 'motion/react';
import { BarChart3, Boxes, CircleCheckBig, Coins, DollarSign, GitBranchPlus, KeyRound, Radio, ScanLine, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Tabs, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import { useAnalyticsOverview } from '@/api/endpoints/analytics';
import { QueryState, formatPercent } from '@/components/modules/analytics/shared';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { EASING } from '@/lib/animations/fluid-transitions';
import { cn, formatCount, formatMoney } from '@/lib/utils';
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
            className="waterhouse-island relative overflow-hidden rounded-[2.15rem] border-border/35 bg-card/58 p-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-6"
            initial={{ opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.45, ease: EASING.easeOutExpo }}
        >
            <div className="pointer-events-none absolute -left-16 top-8 h-44 w-44 rounded-full bg-primary/8 blur-3xl" />
            <div className="pointer-events-none absolute right-8 top-0 h-px w-2/5 bg-gradient-to-r from-transparent via-primary/30 to-transparent" />
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-2">
                    <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-background/45 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                        <Sparkles className="h-3.5 w-3.5" />
                        <span>{t('badge')}</span>
                    </div>
                    <div>
                        <h2 className="text-xl font-semibold tracking-tight md:text-2xl">{t('title')}</h2>
                        <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">{t('description')}</p>
                    </div>
                </div>

                <Tabs value={range} onValueChange={(value) => setRange(value as OverviewRange)}>
                    <TabsList className="waterhouse-pod w-max rounded-[1.4rem] border-border/30 bg-background/38 p-1 shadow-waterhouse-soft backdrop-blur-md">
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
                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
                        {cards.map((card, index) => (
                            <article
                                key={card.title}
                                className={cn(
                                    'waterhouse-pod group overflow-hidden rounded-[1.65rem] border-border/35 bg-background/46 p-4 shadow-waterhouse-soft backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:border-primary/22 hover:shadow-[var(--waterhouse-shadow-soft)]',
                                    index < 2 ? 'xl:min-h-36' : 'xl:min-h-32',
                                )}
                            >
                                <div className="pointer-events-none absolute inset-x-4 top-0 h-px bg-gradient-to-r from-transparent via-primary/20 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
                                <div className="flex items-start justify-between gap-3">
                                    <div className="min-w-0">
                                        <div className="mb-3 flex items-center gap-2 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground/70">
                                            <ScanLine className="h-3.5 w-3.5 text-primary/55" />
                                            <span>{card.title}</span>
                                        </div>
                                        <div className="flex items-baseline gap-1">
                                            <span className="text-2xl font-semibold tracking-tight md:text-3xl">
                                                <AnimatedNumber value={card.value} />
                                            </span>
                                            {card.unit ? <span className="text-sm text-muted-foreground">{card.unit}</span> : null}
                                        </div>
                                    </div>
                                    <div
                                        className={cn(
                                            'flex h-11 w-11 shrink-0 items-center justify-center rounded-[1.25rem] bg-primary/10 text-primary shadow-sm',
                                            card.accentClassName,
                                        )}
                                    >
                                        <card.icon className="h-[1.125rem] w-[1.125rem]" />
                                    </div>
                                </div>
                            </article>
                        ))}
                    </div>
                </QueryState>
            </div>
        </motion.section>
    );
}
