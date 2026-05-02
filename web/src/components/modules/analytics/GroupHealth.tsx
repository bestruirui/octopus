'use client';

import { Activity, AlertTriangle, CircleOff, ShieldCheck, Radar } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAnalyticsGroupHealth } from '@/api/endpoints/analytics';
import { ObservatorySection, QueryState, StatusBadge, formatUnixTime } from './shared';

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
        <ObservatorySection
            eyebrow={t('cards.routeHealth.title')}
            title={t('cards.routeHealth.title')}
            description={t('routeHealth.description')}
            icon={Radar}
        >
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
                                className="waterhouse-pod rounded-[1.75rem] border border-border/30 bg-background/40 p-4 shadow-waterhouse-soft backdrop-blur-md"
                            >
                                <div className="flex items-start justify-between gap-3">
                                    <div className="min-w-0">
                                        <div className="flex items-center gap-2">
                                            <div className="flex h-9 w-9 items-center justify-center rounded-[1.05rem] bg-primary/10 text-primary shadow-waterhouse-soft">
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
                                    <div className="waterhouse-pod rounded-[1.3rem] border border-border/25 bg-background/48 p-3 shadow-waterhouse-soft">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.healthScore')}</div>
                                        <div className="mt-2 text-2xl font-semibold">{item.health_score}</div>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.3rem] border border-border/25 bg-background/48 p-3 shadow-waterhouse-soft">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.failureCount')}</div>
                                        <div className="mt-2 text-2xl font-semibold">{item.failure_count}</div>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.3rem] border border-border/25 bg-background/48 p-3 shadow-waterhouse-soft">
                                        <div className="text-xs text-muted-foreground">{t('routeHealth.enabledItems')}</div>
                                        <div className="mt-2 text-sm font-semibold">
                                            {item.enabled_item_count} / {item.item_count}
                                        </div>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.3rem] border border-border/25 bg-background/48 p-3 shadow-waterhouse-soft">
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
        </ObservatorySection>
    );
}
