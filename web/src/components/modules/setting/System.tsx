'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Monitor, Globe, Clock, Shield, HelpCircle, ShieldAlert } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

function TipLabel({ text, tip }: { text: string; tip: string }) {
    return (
        <span className="flex items-center gap-1.5 text-sm font-medium">
            {text}
            <TooltipProvider>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent>{tip}</TooltipContent>
                </Tooltip>
            </TooltipProvider>
        </span>
    );
}

export function SettingSystem() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [proxyUrl, setProxyUrl] = useState('');
    const [statsSaveInterval, setStatsSaveInterval] = useState('');
    const [corsAllowOrigins, setCorsAllowOrigins] = useState('');

    const [cbEnabled, setCBEnabled] = useState(false);
    const [cbFailureThreshold, setCBFailureThreshold] = useState('');
    const [cbBaseCooldownS, setCBBaseCooldownS] = useState('');
    const [cbMaxCooldownS, setCBMaxCooldownS] = useState('');
    const [cbBackoffFactor, setCBBackoffFactor] = useState('');

    const initialProxyUrl = useRef('');
    const initialStatsSaveInterval = useRef('');
    const initialCorsAllowOrigins = useRef('');
    const initialCBEnabled = useRef(false);
    const initialCBFailureThreshold = useRef('');
    const initialCBBaseCooldownMS = useRef('');
    const initialCBMaxCooldownMS = useRef('');
    const initialCBBackoffFactor = useRef('');

    useEffect(() => {
        if (!settings) return;
        const get = (key: string) => settings.find(s => s.key === key)?.value ?? '';
        const proxy = get(SettingKey.ProxyURL);
        const interval = get(SettingKey.StatsSaveInterval);
        const cors = get(SettingKey.CORSAllowOrigins);
        const cbEnabledV = get(SettingKey.CBEnabled) === 'true';
        const cbFailureThresholdV = get(SettingKey.CBFailureThreshold);
        const cbBaseCooldownMSV = get(SettingKey.CBBaseCooldownMS);
        const cbMaxCooldownMSV = get(SettingKey.CBMaxCooldownMS);
        const cbBackoffFactorV = get(SettingKey.CBBackoffFactor);

        queueMicrotask(() => {
            setProxyUrl(proxy);
            setStatsSaveInterval(interval);
            setCorsAllowOrigins(cors);
            setCBEnabled(cbEnabledV);
            setCBFailureThreshold(cbFailureThresholdV);
            setCBBaseCooldownS(msToSecondsString(cbBaseCooldownMSV));
            setCBMaxCooldownS(msToSecondsString(cbMaxCooldownMSV));
            setCBBackoffFactor(cbBackoffFactorV);
        });

        initialProxyUrl.current = proxy;
        initialStatsSaveInterval.current = interval;
        initialCorsAllowOrigins.current = cors;
        initialCBEnabled.current = cbEnabledV;
        initialCBFailureThreshold.current = cbFailureThresholdV;
        initialCBBaseCooldownMS.current = cbBaseCooldownMSV;
        initialCBMaxCooldownMS.current = cbMaxCooldownMSV;
        initialCBBackoffFactor.current = cbBackoffFactorV;
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string, onSaved?: () => void) => {
        if (value === initialValue) return;
        setSetting.mutate({ key, value }, {
            onSuccess: () => {
                toast.success(t('saved'));
                onSaved?.();
            }
        });
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 custom-shadow space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Monitor className="h-5 w-5" />
                {t('system')}
            </h2>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Globe className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('proxyUrl.label')}</span>
                </div>
                <Input
                    value={proxyUrl}
                    onChange={(e) => setProxyUrl(e.target.value)}
                    onBlur={() => handleSave(SettingKey.ProxyURL, proxyUrl, initialProxyUrl.current, () => {
                        initialProxyUrl.current = proxyUrl;
                    })}
                    placeholder={t('proxyUrl.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('statsSaveInterval.label')}</span>
                </div>
                <Input
                    type="number"
                    value={statsSaveInterval}
                    onChange={(e) => setStatsSaveInterval(e.target.value)}
                    onBlur={() => handleSave(SettingKey.StatsSaveInterval, statsSaveInterval, initialStatsSaveInterval.current, () => {
                        initialStatsSaveInterval.current = statsSaveInterval;
                    })}
                    placeholder={t('statsSaveInterval.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Shield className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('corsAllowOrigins.label')}</span>
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent>
                                {t('corsAllowOrigins.hint')}
                                <br />
                                {t('corsAllowOrigins.example')}
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                </div>
                <Input
                    value={corsAllowOrigins}
                    onChange={(e) => setCorsAllowOrigins(e.target.value)}
                    onBlur={() => handleSave(SettingKey.CORSAllowOrigins, corsAllowOrigins, initialCorsAllowOrigins.current, () => {
                        initialCorsAllowOrigins.current = corsAllowOrigins;
                    })}
                    className="w-48 rounded-xl"
                />
            </div>

            <div className="pt-2 border-t border-border/50 space-y-3">
                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <ShieldAlert className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('circuitBreaker.enabled')}</span>
                    </div>
                    <Switch
                        checked={cbEnabled}
                        onCheckedChange={(checked) => {
                            setCBEnabled(checked);
                            handleSave(SettingKey.CBEnabled, checked ? 'true' : 'false', initialCBEnabled.current ? 'true' : 'false', () => {
                                initialCBEnabled.current = checked;
                            });
                        }}
                    />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div className="space-y-1">
                        <TipLabel text={t('circuitBreaker.failureThreshold')} tip={t('circuitBreaker.failureThresholdTip')} />
                        <Input
                            type="number"
                            value={cbFailureThreshold}
                            onChange={(e) => setCBFailureThreshold(e.target.value)}
                            onBlur={() => handleSave(SettingKey.CBFailureThreshold, cbFailureThreshold, initialCBFailureThreshold.current, () => {
                                initialCBFailureThreshold.current = cbFailureThreshold;
                            })}
                            className="rounded-xl"
                        />
                    </div>

                    <div className="space-y-1">
                        <TipLabel text={t('circuitBreaker.baseCooldownS')} tip={t('circuitBreaker.baseCooldownSTip')} />
                        <Input
                            type="number"
                            value={cbBaseCooldownS}
                            onChange={(e) => setCBBaseCooldownS(e.target.value)}
                            onBlur={() => {
                                const ms = secondsToMSString(cbBaseCooldownS);
                                handleSave(SettingKey.CBBaseCooldownMS, ms, initialCBBaseCooldownMS.current, () => {
                                    initialCBBaseCooldownMS.current = ms;
                                    setCBBaseCooldownS(msToSecondsString(ms));
                                });
                            }}
                            className="rounded-xl"
                        />
                    </div>

                    <div className="space-y-1">
                        <TipLabel text={t('circuitBreaker.maxCooldownS')} tip={t('circuitBreaker.maxCooldownSTip')} />
                        <Input
                            type="number"
                            value={cbMaxCooldownS}
                            onChange={(e) => setCBMaxCooldownS(e.target.value)}
                            onBlur={() => {
                                const ms = secondsToMSString(cbMaxCooldownS);
                                handleSave(SettingKey.CBMaxCooldownMS, ms, initialCBMaxCooldownMS.current, () => {
                                    initialCBMaxCooldownMS.current = ms;
                                    setCBMaxCooldownS(msToSecondsString(ms));
                                });
                            }}
                            className="rounded-xl"
                        />
                    </div>

                    <div className="space-y-1">
                        <TipLabel text={t('circuitBreaker.backoffFactor')} tip={t('circuitBreaker.backoffFactorTip')} />
                        <Input
                            type="number"
                            step="0.1"
                            value={cbBackoffFactor}
                            onChange={(e) => setCBBackoffFactor(e.target.value)}
                            onBlur={() => handleSave(SettingKey.CBBackoffFactor, cbBackoffFactor, initialCBBackoffFactor.current, () => {
                                initialCBBackoffFactor.current = cbBackoffFactor;
                            })}
                            className="rounded-xl"
                        />
                    </div>
                </div>
            </div>
        </div>
    );
}

function msToSecondsString(ms: string): string {
    const n = Number(ms);
    if (!Number.isFinite(n) || n <= 0) return '';
    return String(Math.round(n / 1000));
}

function secondsToMSString(sec: string): string {
    const n = Number(sec);
    if (!Number.isFinite(n) || n <= 0) return '0';
    return String(Math.round(n * 1000));
}
