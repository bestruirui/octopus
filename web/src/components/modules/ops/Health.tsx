'use client';

import { Activity, Database, ShieldCheck, TriangleAlert, Workflow } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useOpsHealthStatus } from '@/api/endpoints/ops';
import { MetricCard, QueryState, StatusBadge, formatUnixTime } from '@/components/modules/analytics/shared';

function HealthSignalCard({
    title,
    ok,
    icon: Icon,
    okLabel,
    issueLabel,
}: {
    title: string;
    ok: boolean;
    icon: typeof Database;
    okLabel: string;
    issueLabel: string;
}) {
    return (
        <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <div className="text-xs text-muted-foreground">{title}</div>
                    <div className="mt-3 flex items-center gap-2">
                        <StatusBadge label={ok ? okLabel : issueLabel} tone={ok ? 'success' : 'warning'} />
                    </div>
                </div>
                <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl ${ok ? 'bg-emerald-500/10 text-emerald-600' : 'bg-amber-500/10 text-amber-600'}`}>
                    <Icon className="h-4 w-4" />
                </div>
            </div>
        </article>
    );
}

function getHealthTone(status: 'healthy' | 'warning' | 'degraded' | 'down' | 'empty') {
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

export function Health() {
    const t = useTranslations('ops');
    const { data, isLoading, error } = useOpsHealthStatus();

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('tabs.health')}</h3>
                <p className="text-sm leading-6 text-muted-foreground">{t('health.description')}</p>
            </div>

            <QueryState
                loading={isLoading}
                error={error}
                empty={!data}
                emptyLabel={t('states.loading')}
            >
                <div className="space-y-4">
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                        <HealthSignalCard
                            title={t('health.cards.database')}
                            ok={!!data?.database_ok}
                            icon={Database}
                            okLabel={t('health.statuses.ok')}
                            issueLabel={t('health.statuses.issue')}
                        />
                        <HealthSignalCard
                            title={t('health.cards.cache')}
                            ok={!!data?.cache_ok}
                            icon={ShieldCheck}
                            okLabel={t('health.statuses.ok')}
                            issueLabel={t('health.statuses.issue')}
                        />
                        <HealthSignalCard
                            title={t('health.cards.taskRuntime')}
                            ok={!!data?.task_runtime_ok}
                            icon={Workflow}
                            okLabel={t('health.statuses.ok')}
                            issueLabel={t('health.statuses.issue')}
                        />
                        <MetricCard
                            title={t('health.cards.recentErrors')}
                            value={data?.recent_error_count ?? 0}
                            icon={TriangleAlert}
                            accentClassName="bg-destructive/10 text-destructive"
                        />
                    </div>

                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                        <MetricCard
                            title={t('health.cards.healthyGroups')}
                            value={data?.healthy_group_count ?? 0}
                            icon={ShieldCheck}
                            accentClassName="bg-emerald-500/10 text-emerald-600"
                        />
                        <MetricCard
                            title={t('health.cards.warningGroups')}
                            value={(data?.warning_group_count ?? 0) + (data?.degraded_group_count ?? 0)}
                            icon={Activity}
                        />
                        <MetricCard
                            title={t('health.cards.downGroups')}
                            value={data?.down_group_count ?? 0}
                            icon={TriangleAlert}
                            accentClassName="bg-destructive/10 text-destructive"
                        />
                        <MetricCard
                            title={t('health.cards.emptyGroups')}
                            value={data?.empty_group_count ?? 0}
                            icon={Database}
                        />
                    </div>

                    <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                        <div className="mb-4 flex items-center justify-between gap-3">
                            <div>
                                <h4 className="text-sm font-semibold">{t('health.detail.failingGroups')}</h4>
                                <p className="mt-1 text-xs text-muted-foreground">
                                    {formatUnixTime(data?.checked_at)}
                                </p>
                            </div>
                        </div>

                        <QueryState
                            loading={false}
                            error={null}
                            empty={!data || data.failing_groups.length === 0}
                            emptyLabel={t('health.detail.noIssues')}
                        >
                            <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                                {(data?.failing_groups ?? []).map((item) => (
                                    <article
                                        key={`${item.group_id}-${item.endpoint_type}`}
                                        className="rounded-2xl border border-border/40 bg-card p-4"
                                    >
                                        <div className="flex items-start justify-between gap-3">
                                            <div className="min-w-0">
                                                <h5 className="truncate text-sm font-semibold">{item.group_name}</h5>
                                                <p className="mt-1 text-xs text-muted-foreground">
                                                    {t('health.detail.endpointType')}: {item.endpoint_type}
                                                </p>
                                            </div>
                                            <StatusBadge
                                                label={t(`health.groupStatuses.${item.status}`)}
                                                tone={getHealthTone(item.status)}
                                            />
                                        </div>

                                        <div className="mt-4 grid grid-cols-2 gap-3">
                                            <div className="rounded-2xl border border-border/40 bg-background/70 p-3">
                                                <div className="text-xs text-muted-foreground">{t('health.detail.healthScore')}</div>
                                                <div className="mt-2 text-lg font-semibold">{item.health_score}</div>
                                            </div>
                                            <div className="rounded-2xl border border-border/40 bg-background/70 p-3">
                                                <div className="text-xs text-muted-foreground">{t('health.detail.failureCount')}</div>
                                                <div className="mt-2 text-lg font-semibold">{item.failure_count}</div>
                                            </div>
                                        </div>
                                    </article>
                                ))}
                            </div>
                        </QueryState>
                    </article>
                </div>
            </QueryState>
        </section>
    );
}

