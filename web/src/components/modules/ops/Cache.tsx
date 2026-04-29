'use client';

import { Activity, Database, Gauge, SlidersHorizontal } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useOpsCacheStatus } from '@/api/endpoints/ops';
import { useNavStore } from '@/components/modules/navbar';
import { Button } from '@/components/ui/button';
import { MetricCard, QueryState, StatusBadge, formatPercent } from '@/components/modules/analytics/shared';

export function Cache() {
    const t = useTranslations('ops');
    const { setActiveItem } = useNavStore();
    const { data, isLoading, error } = useOpsCacheStatus();

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-1">
                    <h3 className="text-base font-semibold">{t('tabs.cache')}</h3>
                    <p className="text-sm leading-6 text-muted-foreground">{t('cache.description')}</p>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    className="rounded-xl"
                    onClick={() => setActiveItem('setting')}
                >
                    {t('actions.openSettings')}
                </Button>
            </div>

            <QueryState
                loading={isLoading}
                error={error}
                empty={!data}
                emptyLabel={t('states.loading')}
            >
                <div className="space-y-4">
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                        <MetricCard
                            title={t('cache.metrics.hitRate')}
                            value={formatPercent(data?.hit_rate).formatted.value}
                            unit={formatPercent(data?.hit_rate).formatted.unit}
                            icon={Gauge}
                            accentClassName="bg-emerald-500/10 text-emerald-600"
                        />
                        <MetricCard
                            title={t('cache.metrics.currentEntries')}
                            value={data?.current_entries ?? 0}
                            helper={`${data?.current_entries ?? 0} / ${data?.max_entries ?? 0}`}
                            icon={Database}
                        />
                        <MetricCard
                            title={t('cache.metrics.ttlSeconds')}
                            value={data?.ttl_seconds ?? 0}
                            unit="s"
                            icon={Activity}
                        />
                        <MetricCard
                            title={t('cache.metrics.threshold')}
                            value={data?.threshold ?? 0}
                            unit="%"
                            icon={SlidersHorizontal}
                            accentClassName="bg-chart-4/10 text-chart-4"
                        />
                    </div>

                    <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                            <div className="space-y-3">
                                <div className="flex flex-wrap items-center gap-2">
                                    <StatusBadge
                                        label={data?.enabled ? t('cache.status.configuredOn') : t('cache.status.configuredOff')}
                                        tone={data?.enabled ? 'success' : 'neutral'}
                                    />
                                    <StatusBadge
                                        label={data?.runtime_enabled ? t('cache.status.runtimeOn') : t('cache.status.runtimeOff')}
                                        tone={data?.runtime_enabled ? 'success' : (data?.enabled ? 'warning' : 'neutral')}
                                    />
                                </div>
                                <p className="text-sm leading-6 text-muted-foreground">
                                    {data?.runtime_enabled ? t('cache.status.runtimeHint') : t('cache.status.runtimeMissing')}
                                </p>
                            </div>

                            <div className="grid grid-cols-2 gap-3 lg:min-w-[320px]">
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('cache.detail.hits')}</div>
                                    <div className="mt-2 text-xl font-semibold">{data?.hits ?? 0}</div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('cache.detail.misses')}</div>
                                    <div className="mt-2 text-xl font-semibold">{data?.misses ?? 0}</div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('cache.detail.maxEntries')}</div>
                                    <div className="mt-2 text-sm font-semibold">{data?.max_entries ?? 0}</div>
                                </div>
                                <div className="rounded-2xl border border-border/40 bg-card p-3">
                                    <div className="text-xs text-muted-foreground">{t('cache.detail.usageRate')}</div>
                                    <div className="mt-2 text-sm font-semibold">
                                        {formatPercent(data?.usage_rate).formatted.value}
                                        {formatPercent(data?.usage_rate).formatted.unit}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </article>
                </div>
            </QueryState>
        </section>
    );
}

