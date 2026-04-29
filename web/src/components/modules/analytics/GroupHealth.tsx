'use client';

import { Activity, AlertTriangle, CircleOff, ShieldCheck } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAnalyticsGroupHealth } from '@/api/endpoints/analytics';
import { QueryState, StatusBadge, formatUnixTime } from './shared';

function getStatusTone(status: 'healthy' | 'warning' | 'degraded' | 'down' | 'empty') {
    switch (status) {
        case 'healthy':
            return 'success' as const;
        case 'warning':
        case 'degraded':
            return 'warning' as const;
        case 'down':
            return 'danger' as const;
        default:
            return 'neutral' as const;
    }
}

function getStatusIcon(status: 'healthy' | 'warning' | 'degraded' | 'down' | 'empty') {
    switch (status) {
        case 'healthy':
            return ShieldCheck;
        case 'down':
            return CircleOff;
        case 'warning':
        case 'degraded':
            return AlertTriangle;
        default:
            return Activity;
    }
}

export function GroupHealth() {
    const t = useTranslations('analytics');
    const { data, isLoading, error } = useAnalyticsGroupHealth();

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('cards.routeHealth.title')}</h3>
                <p className="text-sm leading-6 text-muted-foreground">{t('routeHealth.description')}</p>
            </div>
            <QueryState
                loading={isLoading}
                error={error}
                empty={!data || data.length === 0}
                emptyLabel={isLoading ? t('states.loading') : t('routeHealth.empty')}
            >
                <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                    {(data ?? []).map((item) => {
                        const StatusIcon = getStatusIcon(item.status);
                        return (
                            <article
                                key={`${item.group_id}-${item.endpoint_type}`}
                                className="rounded-2xl border border-border/60 bg-background/70 p-4"
                            >
                                <div className="flex items-start justify-between gap-3">
                                    <div className="min-w-0">
                                        <div className="flex items-center gap-2">
                                            <div className="flex h-9 w-9 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                                                <StatusIcon className="h-4 w-4" />
                                            </div>
                                            <div className="min-w-0">
                                                <h4 className="truncate text-sm font-semibold">{item.group_name}</h4>
                                                <p className="text-xs text-muted-foreground">
                                                    {t('routeHealth.endpointType')}: {item.endpoint_type}
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                    <StatusBadge
                                        label={t(`routeHealth.statuses.${item.status}`)}
                                        tone={getStatusTone(item.status)}
                                    />
                                </div>

                                <div className="mt-4 grid grid-cols-2 gap-3">
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.healthScore')}</div>
                                        <div className="mt-2 text-2xl font-semibold">{item.health_score}</div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.failureCount')}</div>
                                        <div className="mt-2 text-2xl font-semibold">{item.failure_count}</div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.enabledItems')}</div>
                                        <div className="mt-2 text-sm font-semibold">
                                            {item.enabled_item_count} / {item.item_count}
                                        </div>
                                    </div>
                                    <div className="rounded-2xl border border-border/40 bg-card p-3">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.disabledItems')}</div>
                                        <div className="mt-2 text-sm font-semibold">{item.disabled_item_count}</div>
                                    </div>
                                </div>

                                <div className="mt-4 text-xs text-muted-foreground">
                                    {t('routeHealth.lastFailure')}:
                                    <span className="ml-1 text-foreground">
                                        {item.last_failure_at ? formatUnixTime(item.last_failure_at) : t('routeHealth.lastFailureEmpty')}
                                    </span>
                                </div>
                            </article>
                        );
                    })}
                </div>
            </QueryState>
        </section>
    );
}
