'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import {
    Activity,
    Clock,
    Gauge,
    Hash,
    HelpCircle,
    Route,
    ShieldAlert,
    Timer,
    ToggleLeft,
    Zap,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

const PROBE_METHODS = ['auto', 'models', 'head', 'chat', 'custom'] as const;

export function SettingHealthProbe() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [enabled, setEnabled] = useState('true');
    const [interval, setInterval] = useState('120');
    const [timeout, setTimeoutSec] = useState('8');
    const [method, setMethod] = useState('auto');
    const [path, setPath] = useState('');
    const [model, setModel] = useState('gpt-4o-mini');
    const [failThresh, setFailThresh] = useState('3');
    const [degradeMs, setDegradeMs] = useState('5000');
    const [tripOnFail, setTripOnFail] = useState('false');
    const [dashboardRefresh, setDashboardRefresh] = useState('15');

    const initEnabled = useRef('');
    const initInterval = useRef('');
    const initTimeout = useRef('');
    const initMethod = useRef('');
    const initPath = useRef('');
    const initModel = useRef('');
    const initFailThresh = useRef('');
    const initDegradeMs = useRef('');
    const initTripOnFail = useRef('');
    const initDashboardRefresh = useRef('');

    useEffect(() => {
        if (!settings) return;
        const get = (k: string, fallback: string) => settings.find((s) => s.key === k)?.value ?? fallback;

        const e = get(SettingKey.HealthProbeEnabled, 'true');
        const i = get(SettingKey.HealthProbeInterval, '120');
        const to = get(SettingKey.HealthProbeTimeout, '8');
        const m = get(SettingKey.HealthProbeMethod, 'auto');
        const p = get(SettingKey.HealthProbePath, '');
        const mo = get(SettingKey.HealthProbeModel, 'gpt-4o-mini');
        const ft = get(SettingKey.HealthProbeFailThreshold, '3');
        const dm = get(SettingKey.HealthProbeDegradeMS, '5000');
        const tf = get(SettingKey.HealthProbeTripOnFail, 'false');
        const dr = get(SettingKey.HealthDashboardRefresh, '15');

        queueMicrotask(() => {
            setEnabled(e);
            setInterval(i);
            setTimeoutSec(to);
            setMethod(m);
            setPath(p);
            setModel(mo);
            setFailThresh(ft);
            setDegradeMs(dm);
            setTripOnFail(tf);
            setDashboardRefresh(dr);
        });
        initEnabled.current = e;
        initInterval.current = i;
        initTimeout.current = to;
        initMethod.current = m;
        initPath.current = p;
        initModel.current = mo;
        initFailThresh.current = ft;
        initDegradeMs.current = dm;
        initTripOnFail.current = tf;
        initDashboardRefresh.current = dr;
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string, onOk: () => void) => {
        if (value === initialValue) return;
        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    onOk();
                },
                onError: (err) => {
                    toast.error(err instanceof Error ? err.message : String(err));
                },
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Activity className="h-5 w-5" />
                {t('healthProbe.title')}
                <TooltipProvider>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent className="max-w-xs">
                            {t('healthProbe.hint')}
                        </TooltipContent>
                    </Tooltip>
                </TooltipProvider>
            </h2>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <ToggleLeft className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.enabled.label')}</span>
                </div>
                <select
                    value={enabled}
                    onChange={(e) => setEnabled(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeEnabled, enabled, initEnabled.current, () => {
                            initEnabled.current = enabled;
                        })
                    }
                    className="w-48 h-9 rounded-xl border border-input bg-transparent px-3 text-sm"
                >
                    <option value="true">{t('healthProbe.bool.true')}</option>
                    <option value="false">{t('healthProbe.bool.false')}</option>
                </select>
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.interval.label')}</span>
                </div>
                <Input
                    type="number"
                    value={interval}
                    onChange={(e) => setInterval(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeInterval, interval, initInterval.current, () => {
                            initInterval.current = interval;
                        })
                    }
                    placeholder={t('healthProbe.interval.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Timer className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.timeout.label')}</span>
                </div>
                <Input
                    type="number"
                    value={timeout}
                    onChange={(e) => setTimeoutSec(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeTimeout, timeout, initTimeout.current, () => {
                            initTimeout.current = timeout;
                        })
                    }
                    placeholder={t('healthProbe.timeout.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Route className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.method.label')}</span>
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent className="max-w-xs whitespace-pre-line">
                                {t('healthProbe.method.hint')}
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                </div>
                <select
                    value={method}
                    onChange={(e) => setMethod(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeMethod, method, initMethod.current, () => {
                            initMethod.current = method;
                        })
                    }
                    className="w-48 h-9 rounded-xl border border-input bg-transparent px-3 text-sm"
                >
                    {PROBE_METHODS.map((m) => (
                        <option key={m} value={m}>
                            {t(`healthProbe.method.options.${m}`)}
                        </option>
                    ))}
                </select>
            </div>

            {method === 'custom' && (
                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Zap className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('healthProbe.path.label')}</span>
                    </div>
                    <Input
                        value={path}
                        onChange={(e) => setPath(e.target.value)}
                        onBlur={() =>
                            handleSave(SettingKey.HealthProbePath, path, initPath.current, () => {
                                initPath.current = path;
                            })
                        }
                        placeholder={t('healthProbe.path.placeholder')}
                        className="w-48 rounded-xl"
                    />
                </div>
            )}

            {(method === 'chat' || method === 'auto') && (
                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Hash className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('healthProbe.model.label')}</span>
                    </div>
                    <Input
                        value={model}
                        onChange={(e) => setModel(e.target.value)}
                        onBlur={() =>
                            handleSave(SettingKey.HealthProbeModel, model, initModel.current, () => {
                                initModel.current = model;
                            })
                        }
                        placeholder={t('healthProbe.model.placeholder')}
                        className="w-48 rounded-xl"
                    />
                </div>
            )}

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <ShieldAlert className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.failThreshold.label')}</span>
                </div>
                <Input
                    type="number"
                    value={failThresh}
                    onChange={(e) => setFailThresh(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeFailThreshold, failThresh, initFailThresh.current, () => {
                            initFailThresh.current = failThresh;
                        })
                    }
                    placeholder={t('healthProbe.failThreshold.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Gauge className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.degradeMs.label')}</span>
                </div>
                <Input
                    type="number"
                    value={degradeMs}
                    onChange={(e) => setDegradeMs(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeDegradeMS, degradeMs, initDegradeMs.current, () => {
                            initDegradeMs.current = degradeMs;
                        })
                    }
                    placeholder={t('healthProbe.degradeMs.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <ToggleLeft className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.tripOnFail.label')}</span>
                </div>
                <select
                    value={tripOnFail}
                    onChange={(e) => setTripOnFail(e.target.value)}
                    onBlur={() =>
                        handleSave(SettingKey.HealthProbeTripOnFail, tripOnFail, initTripOnFail.current, () => {
                            initTripOnFail.current = tripOnFail;
                        })
                    }
                    className="w-48 h-9 rounded-xl border border-input bg-transparent px-3 text-sm"
                >
                    <option value="true">{t('healthProbe.bool.true')}</option>
                    <option value="false">{t('healthProbe.bool.false')}</option>
                </select>
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('healthProbe.dashboardRefresh.label')}</span>
                </div>
                <Input
                    type="number"
                    value={dashboardRefresh}
                    onChange={(e) => setDashboardRefresh(e.target.value)}
                    onBlur={() =>
                        handleSave(
                            SettingKey.HealthDashboardRefresh,
                            dashboardRefresh,
                            initDashboardRefresh.current,
                            () => {
                                initDashboardRefresh.current = dashboardRefresh;
                            }
                        )
                    }
                    placeholder={t('healthProbe.dashboardRefresh.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="pt-2 border-t border-border">
                <button
                    type="button"
                    onClick={async () => {
                        try {
                            const { apiClient } = await import('@/api/client');
                            await apiClient.post('/api/v1/stats/health/probe', {});
                            toast.success(t('healthProbe.probeNow.success'));
                        } catch (err) {
                            toast.error(err instanceof Error ? err.message : String(err));
                        }
                    }}
                    className="w-full h-9 rounded-xl border border-border bg-muted/40 text-sm font-medium hover:bg-muted transition-colors"
                >
                    {t('healthProbe.probeNow.label')}
                </button>
            </div>
        </div>
    );
}
