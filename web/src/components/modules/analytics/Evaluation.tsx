'use client';

import { Activity, ArrowRight, Clock3, Database, Route } from 'lucide-react';
import { useTranslations } from 'next-intl';
import {
    type SemanticCacheEvaluationSummary,
    useAnalyticsEvaluationRuntime,
    useAnalyticsEvaluationSummary,
} from '@/api/endpoints/analytics';
import { useNavStore } from '@/components/modules/navbar';
import { Button } from '@/components/ui/button';
import { formatCount } from '@/lib/utils';
import { StatusBadge, formatPercent } from './shared';

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
        <div className="rounded-2xl border border-border/40 bg-card p-3">
            <div className="text-xs text-muted-foreground">{label}</div>
            <div className="mt-2 text-sm font-semibold">{value}</div>
        </div>
    );
}

export function Evaluation() {
    const t = useTranslations('analytics');
    const { setActiveItem } = useNavStore();
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

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('evaluation.title')}</h3>
                <p className="text-sm leading-6 text-muted-foreground">{t('evaluation.description')}</p>
            </div>

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Activity className="h-4 w-4" />
                    </div>
                    <h4 className="mt-4 text-sm font-semibold">{t('evaluation.availability.title')}</h4>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">
                        {t('evaluation.availability.description')}
                    </p>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {runtime.isLoading
                            ? t('states.loading')
                            : runtime.hasGroups
                                ? t('evaluation.availability.hint', { count: runtime.groupCount })
                                : t('evaluation.availability.empty')}
                    </p>
                    <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('group')}>
                        {t('evaluation.actions.openGroupTest')}
                        <ArrowRight className="size-4" />
                    </Button>
                </article>

                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Route className="h-4 w-4" />
                    </div>
                    <div className="mt-4 flex items-start justify-between gap-3">
                        <div className="min-w-0">
                            <h4 className="text-sm font-semibold">{t('evaluation.aiRoute.title')}</h4>
                            <p className="mt-2 text-sm leading-6 text-muted-foreground">
                                {t('evaluation.aiRoute.description')}
                            </p>
                        </div>
                        <StatusBadge
                            label={t(`evaluation.runtime.status.${aiRouteStatus}`)}
                            tone={getStatusTone(aiRouteStatus)}
                        />
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {aiRoute
                            ? t('evaluation.aiRoute.hint', { step: t(`evaluation.runtime.step.${aiRouteStep}`) })
                            : hasAiRouteUnavailable
                                ? t('evaluation.aiRoute.unavailable')
                                : t('evaluation.aiRoute.empty')}
                    </p>
                    <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('group')}>
                        {t('evaluation.actions.openAIRoute')}
                        <ArrowRight className="size-4" />
                    </Button>
                </article>

                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Database className="h-4 w-4" />
                    </div>
                    <div className="mt-4 flex items-start justify-between gap-3">
                        <div className="min-w-0">
                            <h4 className="text-sm font-semibold">{t('evaluation.semanticCache.title')}</h4>
                            <p className="mt-2 text-sm leading-6 text-muted-foreground">
                                {t('evaluation.semanticCache.description')}
                            </p>
                        </div>
                        <StatusBadge
                            label={t(`evaluation.semanticCache.status.${semanticCacheStatus.key}`)}
                            tone={semanticCacheStatus.tone}
                        />
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">{t('evaluation.semanticCache.hint')}</p>
                    <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('setting')}>
                        {t('evaluation.actions.openSemanticCache')}
                        <ArrowRight className="size-4" />
                    </Button>
                </article>
            </div>

            <div className="mt-4 space-y-4">
                <div className="rounded-2xl border border-dashed border-border bg-background/60 p-4">
                    <p className="text-sm font-semibold">{t('evaluation.summary.title')}</p>
                    <p className="mt-1 text-sm leading-6 text-muted-foreground">{t('evaluation.summary.description')}</p>
                </div>

                <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                    <article className="rounded-2xl border border-border/60 bg-background/60 p-4">
                        <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                            <Clock3 className="h-4 w-4" />
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
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('evaluation.summary.status')}</div>
                                    <div className="mt-2 text-sm font-semibold">
                                        {t(`evaluation.runtime.step.${aiRouteStep}`)}
                                    </div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('evaluation.summary.progress')}</div>
                                    <div className="mt-2 text-sm font-semibold">
                                        {aiRoute.completed_batches} / {aiRoute.total_batches}
                                    </div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('evaluation.summary.groups')}</div>
                                    <div className="mt-2 text-sm font-semibold">
                                        {aiRoute.result?.group_count ?? 0}
                                    </div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('evaluation.summary.routes')}</div>
                                    <div className="mt-2 text-sm font-semibold">
                                        {aiRoute.result?.route_count ?? 0} / {aiRoute.result?.item_count ?? 0}
                                    </div>
                                </div>
                            </div>
                        ) : (
                            <p className="mt-4 text-sm leading-6 text-muted-foreground">{t('evaluation.aiRoute.empty')}</p>
                        )}
                    </article>

                    <article className="rounded-2xl border border-border/60 bg-background/60 p-4">
                        <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                            <Clock3 className="h-4 w-4" />
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
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('evaluation.summary.progress')}</div>
                                        <div className="mt-2 text-sm font-semibold">
                                            {groupTest.completed} / {groupTest.total}
                                        </div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('evaluation.summary.result')}</div>
                                        <div className="mt-2 text-sm font-semibold">
                                            {groupTestResultLabel}
                                        </div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('evaluation.summary.passed')}</div>
                                        <div className="mt-2 text-sm font-semibold">{passedCount}</div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('evaluation.summary.failed')}</div>
                                        <div className="mt-2 text-sm font-semibold">{failedCount}</div>
                                    </div>
                                </div>
                                {groupTest.message ? (
                                    <p className="mt-3 text-sm leading-6 text-destructive">{groupTest.message}</p>
                                ) : null}
                            </>
                        ) : (
                            <p className="mt-4 text-sm leading-6 text-muted-foreground">
                                {hasGroupTestUnavailable
                                    ? t('evaluation.summary.unavailable')
                                    : t('evaluation.summary.empty')}
                            </p>
                        )}
                    </article>
                </div>

                <article className="rounded-2xl border border-border/60 bg-background/60 p-4">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                        <div className="min-w-0">
                            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                                <Database className="h-4 w-4" />
                            </div>
                            <h4 className="mt-4 text-sm font-semibold">{t('evaluation.semanticCache.summaryTitle')}</h4>
                            <p className="mt-2 text-sm leading-6 text-muted-foreground">
                                {t('evaluation.semanticCache.summaryDescription')}
                            </p>
                        </div>
                        <div className="flex flex-col items-start gap-3 lg:items-end">
                            <StatusBadge
                                label={t(`evaluation.semanticCache.status.${semanticCacheStatus.key}`)}
                                tone={semanticCacheStatus.tone}
                            />
                            <Button className="rounded-xl" onClick={() => setActiveItem('setting')}>
                                {t('evaluation.actions.openSemanticCache')}
                                <ArrowRight className="size-4" />
                            </Button>
                        </div>
                    </div>

                    {semanticCacheQuery.isLoading && !semanticCache ? (
                        <div className="mt-4 rounded-2xl border border-dashed border-border bg-card p-4 text-sm text-muted-foreground">
                            {t('states.loading')}
                        </div>
                    ) : !semanticCache ? (
                        <div className="mt-4 rounded-2xl border border-dashed border-border bg-card p-4 text-sm text-muted-foreground">
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
        </section>
    );
}
