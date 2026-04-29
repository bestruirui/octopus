'use client';

import { useState } from 'react';
import { BarChart3 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { PageWrapper } from '@/components/common/PageWrapper';
import { Tabs, TabsContents, TabsContent, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import type { AnalyticsRange } from '@/api/endpoints/analytics';
import { Overview } from './Overview';
import { Utilization } from './Utilization';
import { GroupHealth } from './GroupHealth';
import { Evaluation } from './Evaluation';

type AnalyticsTab = 'overview' | 'utilization' | 'route-health' | 'evaluation';

const RANGE_OPTIONS: AnalyticsRange[] = ['1d', '7d', '30d', '90d', 'ytd', 'all'];

export function Analytics() {
    const t = useTranslations('analytics');
    const [activeTab, setActiveTab] = useState<AnalyticsTab>('overview');
    const [range, setRange] = useState<AnalyticsRange>('7d');

    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
            <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
                <div className="flex items-start gap-4">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <BarChart3 className="h-5 w-5" />
                    </div>
                    <div className="min-w-0 space-y-2">
                        <h2 className="text-2xl font-bold">{t('title')}</h2>
                        <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                            {t('subtitle')}
                        </p>
                    </div>
                </div>
            </section>

            <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as AnalyticsTab)}>
                <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
                    <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
                        <div className="overflow-x-auto">
                            <TabsList className="w-max min-w-full xl:min-w-0">
                                <TabsTrigger value="overview">{t('cards.overview.title')}</TabsTrigger>
                                <TabsTrigger value="utilization">{t('cards.utilization.title')}</TabsTrigger>
                                <TabsTrigger value="route-health">{t('cards.routeHealth.title')}</TabsTrigger>
                                <TabsTrigger value="evaluation">{t('evaluation.title')}</TabsTrigger>
                            </TabsList>
                        </div>

                        <Tabs value={range} onValueChange={(value) => setRange(value as AnalyticsRange)}>
                            <div className="overflow-x-auto">
                                <TabsList className="w-max min-w-full xl:min-w-0">
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
                    <TabsContent value="overview">
                        <Overview range={range} />
                    </TabsContent>
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
