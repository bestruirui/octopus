'use client';

import type { ReactNode } from 'react';
import { Activity, ArrowRight, Database, Orbit, Radar, Route, Waves } from 'lucide-react';
import { useTranslations } from 'next-intl';
import {
    type SemanticCacheEvaluationSummary,
    useAnalyticsEvaluationRuntime,
    useAnalyticsEvaluationSummary,
} from '@/api/endpoints/analytics';
import { useNavStore } from '@/components/modules/navbar';
import { Button } from '@/components/ui/button';
import { formatCount } from '@/lib/utils';
import { ObservatorySection, StatusBadge, formatPercent } from './shared';

function getStatusTone(status?: string) {
    switch (status) {
        case 'completed':
            return 'success' as const;
        case 'unavailable':
            return 'warning' as const;
        case 'failed':
        case 'timeout':
            return 'danger' as const;
        case 'running':
            return 'warning' as const;
        default:
            return 'neutral' as const;
    }
}

function getSemanticCacheStatus(
    semanticCache: SemanticCacheEvaluationSummary | undefined,
    hasError: boolean,
) {
    if (hasError && !semanticCache) {
        return { tone: 'neutral' as const, key: 'unavailable' as const };
    }
    if (!semanticCache) {
        return { tone: 'neutral' as const, key: 'loading' as const };
    }
    if (semanticCache.runtime_enabled) {
        return { tone: 'success' as const, key: 'runtimeOn' as const };
    }
    if (semanticCache.enabled) {
        return { tone: 'warning' as const, key: 'runtimeOff' as const };
    }
    return { tone: 'neutral' as const, key: 'disabled' as const };
}

function formatMetricCount(value: number | undefined) {
    const { formatted } = formatCount(value);
    return `${formatted.value}${formatted.unit}`;
}

function formatMetricPercent(value: number | undefined) {
    const { formatted } = formatPercent(value);
    return `${formatted.value}${formatted.unit}`;
}

function SummaryStat({
    label,
    value,
}: {
    label: string;
    value: string;
}) {
    return (
        <div className="waterhouse-pod rounded-[1.25rem] border border-border/25 bg-background/48 p-3 shadow-waterhouse-soft">
            <div className="text-xs text-muted-foreground">{label}</div>
            <div className="mt-2 text-sm font-semibold">{value}</div>
        </div>
    );
}

function EntryCard({
    icon: Icon,
    title,
    description,
    hint,
    status,
    action,
}: {
    icon: typeof Activity;
    title: string;
    description: string;
    hint: string;
    status?: { label: string; tone: 'success' | 'warning' | 'danger' | 'neutral' };
    action: ReactNode;
}) {
    return (
        <article className="waterhouse-pod rounded-[1.8rem] border border-border/30 bg-background/40 p-4 shadow-waterhouse-soft backdrop-blur-md">
            <div className="flex items-start justify-between gap-3">
                <div className="waterhouse-pod grid h-10 w-10 shrink-0 place-items-center rounded-[1.15rem] border border-border/25 bg-background/48 text-primary shadow-waterhouse-soft">
                    <Icon className="h-4 w-4" />
                </div>
                {status ? <StatusBadge label={status.label} tone={status.tone} /> : null}
            </div>
            <div className="mt-4 space-y-2">
                <h4 className="text-sm font-semibold">{title}</h4>
                <p className="text-sm leading-6 text-muted-foreground">{description}</p>
                <div className="rounded-[1.2rem] border border-border/20 bg-background/42 px-3 py-2 text-sm text-muted-foreground shadow-waterhouse-soft">
                    {hint}
                </div>
            </div>
            <div className="mt-4">{action}</div>
        </article>
    );
}

export function Evaluation() {
    const t = useTranslations('analytics');
    const { setActiveItem } = useNavStore();
    const sectionDescription = t('evaluation.description');
    const summaryDescription = t('evaluation.semanticCache.summaryDescription');
    const runtime = useAnalyticsEvaluationRuntime();
    const semanticCacheQuery = useAnalyticsEvaluationSummary();
    const semanticCache = semanticCacheQuery.data?.semantic_cache;
    const aiRoute = runtime.aiRouteProgress;
    const groupTest = runtime.groupTestProgress;
    const passedCount = (groupTest?.results ?? []).filter((result) => result.passed).length;
    const failedCount = (groupTest?.results ?? []).filter((result) => !result.passed).length;
    const hasAiRouteUnavailable = Boolean(runtime.aiRouteTask && runtime.aiRouteError && !aiRoute);
    const hasGroupTestUnavailable = Boolean(runtime.groupTestTask && runtime.groupTestError && !groupTest);
    const groupTestHasFailures = failedCount > 0 || Boolean(groupTest?.message);
    const aiRouteStatus = hasAiRouteUnavailable ? 'unavailable' : (aiRoute?.status ?? 'idle');
    const aiRouteStep = aiRoute?.current_step ?? 'idle';
    const groupTestStatus = groupTest
        ? (groupTest.done ? (groupTestHasFailures ? 'failed' : 'completed') : 'running')
        : hasGroupTestUnavailable
            ? 'unavailable'
            : 'idle';
    const groupTestResultLabel = !groupTest
        ? t('evaluation.summary.empty')
        : !groupTest.done
            ? t('evaluation.runtime.status.running')
            : groupTestHasFailures
                ? t('evaluation.summary.partialFailed')
                : t('evaluation.summary.allPassed');
    const semanticCacheStatus = getSemanticCacheStatus(
        semanticCache,
        !semanticCache && !!semanticCacheQuery.error,
    );
    const statusButtonClassName = 'rounded-[1.2rem] border-border/25 bg-background/44 shadow-waterhouse-soft hover:bg-background/64';

    return (
        <ObservatorySection
            eyebrow={t('evaluation.title')}
            title={t('evaluation.title')}
            description={sectionDescription}
            icon={Radar}
        >
            <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
                <EntryCard
                    icon={Activity}
                    title={t('evaluation.availability.title')}
                    description={t('evaluation.availability.description')}
                    hint={runtime.isLoading
                        ? t('states.loading')
                        : runtime.hasGroups
                            ? t('evaluation.availability.hint', { count: runtime.groupCount })
                            : t('evaluation.availability.empty')}
                    action={
                        <Button className={statusButtonClassName} onClick={() => setActiveItem('group')}>
                            {t('evaluation.actions.openGroupTest')}
                            <ArrowRight className="size-4" />
                        </Button>
                    }
                />
                <EntryCard
                    icon={Route}
                    title={t('evaluation.aiRoute.title')}
                    description={t('evaluation.aiRoute.description')}
                    hint={aiRoute
                        ? t('evaluation.aiRoute.hint', { step: t(`evaluation.runtime.step.${aiRouteStep}`) })
                        : hasAiRouteUnavailable
                            ? t('evaluation.aiRoute.unavailable')
                            : t('evaluation.aiRoute.empty')}
                    status={{
                        label: t(`evaluation.runtime.status.${aiRouteStatus}`),
                        tone: getStatusTone(aiRouteStatus),
                    }}
                    action={
                        <Button className={statusButtonClassName} onClick={() => setActiveItem('group')}>
                            {t('evaluation.actions.openAIRoute')}
                            <ArrowRight className="size-4" />
                        </Button>
                    }
                />
                <EntryCard
                    icon={Database}
                    title={t('evaluation.semanticCache.title')}
                    description={t('evaluation.semanticCache.description')}
                    hint={t('evaluation.semanticCache.hint')}
                    status={{
                        label: t(`evaluation.semanticCache.status.${semanticCacheStatus.key}`),
                        tone: semanticCacheStatus.tone,
                    }}
                    action={
                        <Button className={statusButtonClassName} onClick={() => setActiveItem('setting')}>
                            {t('evaluation.actions.openSemanticCache')}
                            <ArrowRight className="size-4" />
                        </Button>
                    }
                />
            </div>

            <div className="mt-4 space-y-4">
                <div className="waterhouse-pod rounded-[1.8rem] border border-dashed border-border/30 bg-background/34 p-4 shadow-waterhouse-soft backdrop-blur-md">
                    <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-primary/10 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft">
                        <Orbit className="h-3.5 w-3.5" />
                        {t('evaluation.summary.title')}
                    </div>
                    <p className="mt-1 text-sm leading-6 text-muted-foreground">{t('evaluation.summary.description')}</p>
                </div>

                <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                    <article className="waterhouse-pod rounded-[1.8rem] border border-border/30 bg-background/38 p-4 shadow-waterhouse-soft backdrop-blur-md">
                        <div className="flex h-10 w-10 items-center justify-center rounded-[1.15rem] bg-primary/10 text-primary shadow-waterhouse-soft">
                            <Route className="h-4 w-4" />
                        </div>
                        <div className="mt-4 flex items-center justify-between gap-3">
                            <h4 className="text-sm font-semibold">{t('evaluation.summary.aiRoute')}</h4>
                            <StatusBadge
                                label={t(`evaluation.runtime.status.${aiRouteStatus}`)}
                                tone={getStatusTone(aiRouteStatus)}
                            />
                        </div>
                        {aiRoute ? (
                            <div className="mt-4 grid grid-cols-2 gap-3">
                                <SummaryStat
                                    label={t('evaluation.summary.status')}
                                    value={t(`evaluation.runtime.step.${aiRouteStep}`)}
                                />
                                <SummaryStat
                                    label={t('evaluation.summary.progress')}
                                    value={`${aiRoute.completed_batches} / ${aiRoute.total_batches}`}
                                />
                                <SummaryStat
                                    label={t('evaluation.summary.groups')}
                                    value={String(aiRoute.result?.group_count ?? 0)}
                                />
                                <SummaryStat
                                    label={t('evaluation.summary.routes')}
                                    value={`${aiRoute.result?.route_count ?? 0} / ${aiRoute.result?.item_count ?? 0}`}
                                />
                            </div>
                        ) : (
                            <div className="mt-4 rounded-[1.3rem] border border-border/20 bg-background/44 px-4 py-3 text-sm leading-6 text-muted-foreground shadow-waterhouse-soft">
                                {t('evaluation.aiRoute.empty')}
                            </div>
                        )}
                    </article>

                    <article className="waterhouse-pod rounded-[1.8rem] border border-border/30 bg-background/38 p-4 shadow-waterhouse-soft backdrop-blur-md">
                        <div className="flex h-10 w-10 items-center justify-center rounded-[1.15rem] bg-primary/10 text-primary shadow-waterhouse-soft">
                            <Activity className="h-4 w-4" />
                        </div>
                        <div className="mt-4 flex items-center justify-between gap-3">
                            <h4 className="text-sm font-semibold">{t('evaluation.summary.groupTest')}</h4>
                            <StatusBadge
                                label={t(`evaluation.runtime.status.${groupTestStatus}`)}
                                tone={getStatusTone(groupTestStatus)}
                            />
                        </div>
                        {groupTest ? (
                            <>
                                <div className="mt-4 grid grid-cols-2 gap-3">
                                    <SummaryStat
                                        label={t('evaluation.summary.progress')}
                                        value={`${groupTest.completed} / ${groupTest.total}`}
                                    />
                                    <SummaryStat
                                        label={t('evaluation.summary.result')}
                                        value={groupTestResultLabel}
                                    />
                                    <SummaryStat
                                        label={t('evaluation.summary.passed')}
                                        value={String(passedCount)}
                                    />
                                    <SummaryStat
                                        label={t('evaluation.summary.failed')}
                                        value={String(failedCount)}
                                    />
                                </div>
                                {groupTest.message ? (
                                    <p className="mt-3 text-sm leading-6 text-destructive">{groupTest.message}</p>
                                ) : null}
                            </>
                        ) : (
                            <div className="mt-4 rounded-[1.3rem] border border-border/20 bg-background/44 px-4 py-3 text-sm leading-6 text-muted-foreground shadow-waterhouse-soft">
                                {hasGroupTestUnavailable
                                    ? t('evaluation.summary.unavailable')
                                    : t('evaluation.summary.empty')}
                            </div>
                        )}
                    </article>
                </div>

                <article className="waterhouse-pod rounded-[1.9rem] border border-border/30 bg-background/36 p-4 shadow-waterhouse-soft backdrop-blur-md md:p-5">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                        <div className="min-w-0">
                            <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft">
                                <Waves className="h-3.5 w-3.5" />
                                {t('evaluation.semanticCache.summaryTitle')}
                            </div>
                            <div className="flex h-10 w-10 items-center justify-center rounded-[1.15rem] bg-primary/10 text-primary shadow-waterhouse-soft">
                                <Database className="h-4 w-4" />
                            </div>
                            <h4 className="mt-4 text-sm font-semibold">{t('evaluation.semanticCache.summaryTitle')}</h4>
                            {summaryDescription ? (
                                <p className="mt-2 text-sm leading-6 text-muted-foreground">
                                    {summaryDescription}
                                </p>
                            ) : null}
                        </div>
                        <div className="flex flex-col items-start gap-3 lg:items-end">
                            <StatusBadge
                                label={t(`evaluation.semanticCache.status.${semanticCacheStatus.key}`)}
                                tone={semanticCacheStatus.tone}
                            />
                            <Button className={statusButtonClassName} onClick={() => setActiveItem('setting')}>
                                {t('evaluation.actions.openSemanticCache')}
                                <ArrowRight className="size-4" />
                            </Button>
                        </div>
                    </div>

                    {semanticCacheQuery.isLoading && !semanticCache ? (
                        <div className="mt-4 rounded-[1.3rem] border border-dashed border-border/30 bg-background/44 p-4 text-sm text-muted-foreground shadow-waterhouse-soft">
                            {t('states.loading')}
                        </div>
                    ) : !semanticCache ? (
                        <div className="mt-4 rounded-[1.3rem] border border-dashed border-border/30 bg-background/44 p-4 text-sm text-muted-foreground shadow-waterhouse-soft">
                            {t('evaluation.semanticCache.unavailable')}
                        </div>
                    ) : (
                        <>
                            <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.hits')}
                                    value={formatMetricCount(semanticCache.hits)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.misses')}
                                    value={formatMetricCount(semanticCache.misses)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.hitRate')}
                                    value={formatMetricPercent(semanticCache.hit_rate)}
                                />
                            </div>

                            <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.currentEntries')}
                                    value={formatMetricCount(semanticCache.current_entries)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.maxEntries')}
                                    value={formatMetricCount(semanticCache.max_entries)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.usageRate')}
                                    value={formatMetricPercent(semanticCache.usage_rate)}
                                />
                            </div>

                            <div className="mt-3 grid grid-cols-2 gap-3 xl:grid-cols-5">
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.evaluatedRequests')}
                                    value={formatMetricCount(semanticCache.evaluated_requests)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.cacheHitResponses')}
                                    value={formatMetricCount(semanticCache.cache_hit_responses)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.cacheMissRequests')}
                                    value={formatMetricCount(semanticCache.cache_miss_requests)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.bypassedRequests')}
                                    value={formatMetricCount(semanticCache.bypassed_requests)}
                                />
                                <SummaryStat
                                    label={t('evaluation.semanticCache.metrics.storedResponses')}
                                    value={formatMetricCount(semanticCache.stored_responses)}
                                />
                            </div>
                        </>
                    )}
                </article>
            </div>
        </ObservatorySection>
    );
}
