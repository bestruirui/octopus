'use client';

import { Coins, Gauge, KeyRound, ShieldAlert } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useOpsQuotaSummary, type OpsQuotaKeyItem } from '@/api/endpoints/ops';
import { useNavStore } from '@/components/modules/navbar';
import { Button } from '@/components/ui/button';
import { MetricCard, QueryState, StatusBadge } from '@/components/modules/analytics/shared';
import { formatCount, formatMoney } from '@/lib/utils';

function getQuotaTone(status: OpsQuotaKeyItem['status']) {
    switch (status) {
        case 'exhausted':
            return 'danger' as const;
        case 'expired':
        case 'limited':
            return 'warning' as const;
        case 'disabled':
            return 'neutral' as const;
        default:
            return 'success' as const;
    }
}

export function Quota() {
    const t = useTranslations('ops');
    const { setActiveItem } = useNavStore();
    const { data, isLoading, error } = useOpsQuotaSummary();

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-1">
                    <h3 className="text-base font-semibold">{t('tabs.quota')}</h3>
                    <p className="text-sm leading-6 text-muted-foreground">{t('quota.description')}</p>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    className="rounded-xl"
                    onClick={() => setActiveItem('setting')}
                >
                    {t('actions.manageKeys')}
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
                            title={t('quota.metrics.availableKeys')}
                            value={data?.available_key_count ?? 0}
                            icon={KeyRound}
                            accentClassName="bg-emerald-500/10 text-emerald-600"
                        />
                        <MetricCard
                            title={t('quota.metrics.limitedKeys')}
                            value={data?.limited_key_count ?? 0}
                            icon={Gauge}
                        />
                        <MetricCard
                            title={t('quota.metrics.exhaustedKeys')}
                            value={data?.exhausted_key_count ?? 0}
                            icon={ShieldAlert}
                            accentClassName="bg-destructive/10 text-destructive"
                        />
                        <MetricCard
                            title={t('quota.metrics.totalMaxCost')}
                            value={formatMoney(data?.total_max_cost).formatted.value}
                            unit={formatMoney(data?.total_max_cost).formatted.unit}
                            icon={Coins}
                            accentClassName="bg-chart-1/10 text-chart-1"
                        />
                    </div>

                    <QueryState
                        loading={false}
                        error={null}
                        empty={!data || data.keys.length === 0}
                        emptyLabel={t('quota.empty')}
                    >
                        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                            {(data?.keys ?? []).map((item) => (
                                <article
                                    key={item.api_key_id}
                                    className="rounded-2xl border border-border/60 bg-background/70 p-4"
                                >
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="min-w-0">
                                            <h4 className="truncate text-sm font-semibold">{item.name}</h4>
                                            <p className="mt-1 text-xs text-muted-foreground">
                                                {t('quota.fields.requestCount')}: {formatCount(item.request_count).formatted.value}
                                                {formatCount(item.request_count).formatted.unit}
                                            </p>
                                        </div>
                                        <StatusBadge
                                            label={t(`quota.statuses.${item.status}`)}
                                            tone={getQuotaTone(item.status)}
                                        />
                                    </div>

                                    <div className="mt-4 grid grid-cols-2 gap-3">
                                        <div className="rounded-2xl border border-border/40 bg-card p-3">
                                            <div className="text-xs text-muted-foreground">{t('quota.fields.totalCost')}</div>
                                            <div className="mt-2 text-sm font-semibold">
                                                {formatMoney(item.total_cost).formatted.value}
                                                {formatMoney(item.total_cost).formatted.unit}
                                            </div>
                                        </div>
                                        <div className="rounded-2xl border border-border/40 bg-card p-3">
                                            <div className="text-xs text-muted-foreground">{t('quota.fields.maxCost')}</div>
                                            <div className="mt-2 text-sm font-semibold">
                                                {item.max_cost > 0
                                                    ? `${formatMoney(item.max_cost).formatted.value}${formatMoney(item.max_cost).formatted.unit}`
                                                    : t('quota.statuses.open')}
                                            </div>
                                        </div>
                                        <div className="rounded-2xl border border-border/40 bg-card p-3">
                                            <div className="text-xs text-muted-foreground">{t('quota.fields.rpm')}</div>
                                            <div className="mt-2 text-sm font-semibold">{item.rate_limit_rpm || '-'}</div>
                                        </div>
                                        <div className="rounded-2xl border border-border/40 bg-card p-3">
                                            <div className="text-xs text-muted-foreground">{t('quota.fields.tpm')}</div>
                                            <div className="mt-2 text-sm font-semibold">{item.rate_limit_tpm || '-'}</div>
                                        </div>
                                    </div>

                                    <div className="mt-4 flex flex-wrap gap-2 text-xs text-muted-foreground">
                                        <span>{t('quota.fields.supportedModels')}: {item.supported_model_count || '-'}</span>
                                        <span>{t('quota.fields.perModelQuota')}: {item.has_per_model_quota ? t('system.fields.enabled') : t('system.fields.disabled')}</span>
                                    </div>
                                </article>
                            ))}
                        </div>
                    </QueryState>
                </div>
            </QueryState>
        </section>
    );
}

