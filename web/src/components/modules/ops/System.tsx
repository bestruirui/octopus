'use client';

import { Boxes, Cpu, GitBranch, KeyRound, Radio, Server, Settings2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useOpsSystemSummary } from '@/api/endpoints/ops';
import { useNavStore } from '@/components/modules/navbar';
import { Button } from '@/components/ui/button';
import { MetricCard, QueryState, StatusBadge } from '@/components/modules/analytics/shared';

function InfoRow({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-start justify-between gap-4 border-b border-border/40 py-3 last:border-b-0 last:pb-0 first:pt-0">
            <span className="text-sm text-muted-foreground">{label}</span>
            <span className="max-w-[60%] break-all text-right text-sm font-medium">{value || '-'}</span>
        </div>
    );
}

export function System() {
    const t = useTranslations('ops');
    const { setActiveItem } = useNavStore();
    const { data, isLoading, error } = useOpsSystemSummary();

    const aiRouteModeLabel = data?.ai_route_legacy_mode ? t('system.fields.legacyMode') : t('system.fields.servicePoolMode');

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-1">
                    <h3 className="text-base font-semibold">{t('tabs.system')}</h3>
                    <p className="text-sm leading-6 text-muted-foreground">{t('system.description')}</p>
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
                            title={t('system.metrics.channels')}
                            value={data?.channel_count ?? 0}
                            icon={Radio}
                        />
                        <MetricCard
                            title={t('system.metrics.groups')}
                            value={data?.group_count ?? 0}
                            icon={Boxes}
                        />
                        <MetricCard
                            title={t('system.metrics.apiKeys')}
                            value={data?.api_key_count ?? 0}
                            icon={KeyRound}
                        />
                        <MetricCard
                            title={t('system.metrics.aiRouteServices')}
                            value={data?.ai_route_enabled_service_count ?? 0}
                            helper={`${data?.ai_route_enabled_service_count ?? 0} / ${data?.ai_route_service_count ?? 0}`}
                            icon={GitBranch}
                            accentClassName="bg-chart-4/10 text-chart-4"
                        />
                    </div>

                    <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.2fr_0.8fr]">
                        <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                            <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
                                <Server className="h-4 w-4" />
                                {t('system.sections.runtime')}
                            </div>
                            <InfoRow label={t('system.fields.version')} value={data?.version || '-'} />
                            <InfoRow label={t('system.fields.commit')} value={data?.commit || '-'} />
                            <InfoRow label={t('system.fields.buildTime')} value={data?.build_time || '-'} />
                            <InfoRow label={t('system.fields.databaseType')} value={data?.database_type || '-'} />
                            <InfoRow label={t('system.fields.publicApiBaseUrl')} value={data?.public_api_base_url || '-'} />
                            <InfoRow label={t('system.fields.proxyUrl')} value={data?.proxy_url || '-'} />
                            <InfoRow
                                label={t('system.fields.relayLogRetention')}
                                value={data?.relay_log_keep_enabled ? `${data?.relay_log_keep_days}d` : t('system.fields.disabled')}
                            />
                            <InfoRow
                                label={t('system.fields.statsSaveInterval')}
                                value={`${data?.stats_save_interval_minutes ?? 0}m`}
                            />
                            <InfoRow
                                label={t('system.fields.syncLlmInterval')}
                                value={`${data?.sync_llm_interval_hours ?? 0}h`}
                            />
                            <InfoRow
                                label={t('system.fields.modelInfoInterval')}
                                value={`${data?.model_info_update_interval_hours ?? 0}h`}
                            />
                        </article>

                        <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                            <div className="mb-2 flex items-center gap-2 text-sm font-semibold">
                                <Settings2 className="h-4 w-4" />
                                {t('system.sections.operations')}
                            </div>
                            <InfoRow
                                label={t('system.fields.importExport')}
                                value={`${data?.import_enabled ? t('system.fields.enabled') : t('system.fields.disabled')} / ${data?.export_enabled ? t('system.fields.enabled') : t('system.fields.disabled')}`}
                            />
                            <InfoRow label={t('system.fields.aiRouteGroup')} value={String(data?.ai_route_group_id ?? 0)} />
                            <InfoRow
                                label={t('system.fields.aiRouteTimeout')}
                                value={`${data?.ai_route_timeout_seconds ?? 0}s`}
                            />
                            <InfoRow
                                label={t('system.fields.aiRouteParallelism')}
                                value={String(data?.ai_route_parallelism ?? 0)}
                            />
                            <InfoRow label={t('system.fields.aiRouteMode')} value={aiRouteModeLabel} />

                            <div className="mt-4 rounded-2xl border border-border/40 bg-card p-3">
                                <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
                                    <Cpu className="h-4 w-4" />
                                    {t('system.sections.services')}
                                </div>

                                <QueryState
                                    loading={false}
                                    error={null}
                                    empty={!data || data.ai_route_services.length === 0}
                                    emptyLabel={t('system.fields.notConfigured')}
                                >
                                    <div className="space-y-3">
                                        {(data?.ai_route_services ?? []).map((service) => (
                                            <article
                                                key={`${service.name}-${service.model}`}
                                                className="rounded-2xl border border-border/40 bg-background/70 p-3"
                                            >
                                                <div className="flex items-start justify-between gap-3">
                                                    <div className="min-w-0">
                                                        <div className="truncate text-sm font-semibold">{service.name}</div>
                                                        <div className="mt-1 text-xs text-muted-foreground">{service.base_url || '-'}</div>
                                                        <div className="mt-1 text-xs text-muted-foreground">{service.model || '-'}</div>
                                                    </div>
                                                    <StatusBadge
                                                        label={service.enabled ? t('system.fields.enabled') : t('system.fields.disabled')}
                                                        tone={service.enabled ? 'success' : 'neutral'}
                                                    />
                                                </div>
                                            </article>
                                        ))}
                                    </div>
                                </QueryState>
                            </div>
                        </article>
                    </div>
                </div>
            </QueryState>
        </section>
    );
}

