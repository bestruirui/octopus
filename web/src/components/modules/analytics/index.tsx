'use client';

import { useState } from 'react';
import { BarChart3, Waves, Orbit } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { PageWrapper } from '@/components/common/PageWrapper';
import { Tabs, TabsContents, TabsContent, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import type { AnalyticsRange } from '@/api/endpoints/analytics';
import { Utilization } from './Utilization';
import { GroupHealth } from './GroupHealth';
import { Evaluation } from './Evaluation';

type AnalyticsTab = 'utilization' | 'route-health' | 'evaluation';

const RANGE_OPTIONS: AnalyticsRange[] = ['1d', '7d', '30d', '90d', 'ytd', 'all'];

export function Analytics() {
    const t = useTranslations('analytics');
    const [activeTab, setActiveTab] = useState<AnalyticsTab>('utilization');
    const [range, setRange] = useState<AnalyticsRange>('7d');
    const subtitle = t('subtitle');

    return (
        <PageWrapper className="analytics-shadowless h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 rounded-t-3xl pb-24 md:pb-4">
            <section className="waterhouse-island relative overflow-hidden rounded-[2.35rem] border border-border/35 bg-card/60 p-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-6">
                <div className="relative flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div className="flex items-start gap-4">
                        <div className="waterhouse-pod grid h-14 w-14 shrink-0 place-items-center rounded-[1.55rem] border border-border/35 bg-background/50 text-primary shadow-waterhouse-soft">
                            <BarChart3 className="h-5 w-5" />
                        </div>
                        <div className="min-w-0 space-y-3">
                            <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-background/46 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft">
                                <Waves className="h-3.5 w-3.5" />
                                {t('title')}
                            </div>
                            <div className="space-y-2">
                                <h2 className="text-2xl font-bold tracking-tight md:text-[2rem]">{t('title')}</h2>
                                {subtitle ? (
                                    <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                                        {subtitle}
                                    </p>
                                ) : null}
                            </div>
                        </div>
                    </div>
                    <div className="waterhouse-pod flex items-center gap-2 self-start rounded-[1.45rem] border border-border/25 bg-background/38 px-3 py-2 text-sm text-muted-foreground shadow-waterhouse-soft">
                        <Orbit className="h-4 w-4 text-primary" />
                        {t(`range.${range}`)}
                    </div>
                </div>
            </section>

            <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as AnalyticsTab)}>
                <section className="waterhouse-island relative overflow-hidden rounded-[2.1rem] border border-border/35 bg-card/58 p-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-5">
                    <div className="relative flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
                        <div className="overflow-x-auto">
                            <TabsList className="waterhouse-pod flex w-max min-w-full flex-wrap rounded-[1.45rem] border-border/30 bg-background/36 p-1 shadow-waterhouse-soft backdrop-blur-md xl:min-w-0">
                                <TabsTrigger value="utilization">{t('cards.utilization.title')}</TabsTrigger>
                                <TabsTrigger value="route-health">{t('cards.routeHealth.title')}</TabsTrigger>
                                <TabsTrigger value="evaluation">{t('evaluation.title')}</TabsTrigger>
                            </TabsList>
                        </div>

                        <Tabs value={range} onValueChange={(value) => setRange(value as AnalyticsRange)}>
                            <div className="overflow-x-auto">
                                <TabsList className="waterhouse-pod flex w-max min-w-full flex-wrap rounded-[1.45rem] border-border/30 bg-background/32 p-1 shadow-waterhouse-soft backdrop-blur-md xl:min-w-0">
                                    {RANGE_OPTIONS.map((option) => (
                                        <TabsTrigger key={option} value={option}>
                                            {t(`range.${option}`)}
                                        </TabsTrigger>
                                    ))}
                                </TabsList>
                            </div>
                        </Tabs>
                    </div>
                </section>

                <TabsContents>
                    <TabsContent value="utilization">
                        <Utilization range={range} />
                    </TabsContent>
                    <TabsContent value="route-health">
                        <GroupHealth />
                    </TabsContent>
                    <TabsContent value="evaluation">
                        <Evaluation />
                    </TabsContent>
                </TabsContents>
            </Tabs>
        </PageWrapper>
    );
}
