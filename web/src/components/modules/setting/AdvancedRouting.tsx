'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import {
    AlertTriangle,
    Gauge,
    HelpCircle,
    Route,
    Settings2,
    Timer,
    Zap,
} from 'lucide-react';
import { toast } from 'sonner';
import { SettingKey, useSetSetting, useSettingList } from '@/api/endpoints/setting';
import { Switch } from '@/components/ui/switch';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

function BoolSwitch({
    label,
    tip,
    value,
    onChange,
}: {
    label: string;
    tip?: string;
    value: boolean;
    onChange: (v: boolean) => void;
}) {
    return (
        <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-2 min-w-0">
                <Label className="text-sm font-medium truncate">{label}</Label>
                {tip && (
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                            </TooltipTrigger>
                            <TooltipContent className="max-w-xs text-xs">{tip}</TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                )}
            </div>
            <Switch checked={value} onCheckedChange={onChange} />
        </div>
    );
}

function NumField({
    icon,
    label,
    tip,
    value,
    min,
    max,
    onCommit,
}: {
    icon: React.ReactNode;
    label: string;
    tip?: string;
    value: number;
    min: number;
    max: number;
    onCommit: (v: number) => void;
}) {
    const [local, setLocal] = useState(String(value));
    useEffect(() => setLocal(String(value)), [value]);
    return (
        <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
                {icon}
                <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                        <Label className="text-sm font-medium truncate">{label}</Label>
                        {tip && (
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger asChild>
                                        <HelpCircle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                    </TooltipTrigger>
                                    <TooltipContent className="max-w-xs text-xs">{tip}</TooltipContent>
                                </Tooltip>
                            </TooltipProvider>
                        )}
                    </div>
                </div>
            </div>
            <Input
                type="number"
                className="w-24 h-8 text-right"
                value={local}
                min={min}
                max={max}
                onChange={(e) => setLocal(e.target.value)}
                onBlur={() => {
                    const n = parseInt(local, 10);
                    if (Number.isNaN(n) || n < min || n > max) {
                        setLocal(String(value));
                        return;
                    }
                    if (n !== value) onCommit(n);
                }}
            />
        </div>
    );
}

export function SettingAdvancedRouting() {
    const t = useTranslations('setting.advancedRouting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();
    const saving = useRef(false);

    const get = (key: string, def = '') =>
        settings?.find((s) => s.key === key)?.value ?? def;
    const getBool = (key: string, def = false) => {
        const v = get(key);
        if (v === '') return def;
        return v === 'true';
    };
    const getNum = (key: string, def: number) => {
        const n = parseInt(get(key), 10);
        return Number.isNaN(n) ? def : n;
    };

    const save = async (key: string, value: string) => {
        if (saving.current) return;
        saving.current = true;
        try {
            await setSetting.mutateAsync({ key, value });
            toast.success(t('saved'));
        } catch (err) {
            toast.error(err instanceof Error ? err.message : String(err));
        } finally {
            saving.current = false;
        }
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-5 space-y-5">
            <div className="flex items-start gap-3">
                <Route className="h-5 w-5 text-muted-foreground mt-0.5" />
                <div>
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <p className="text-xs text-muted-foreground mt-1 leading-relaxed">{t('hint')}</p>
                </div>
            </div>

            <div className="space-y-4">
                <BoolSwitch
                    label={t('healthScore.label')}
                    tip={t('healthScore.tip')}
                    value={getBool(SettingKey.HealthScoreRouting, true)}
                    onChange={(v) => save(SettingKey.HealthScoreRouting, String(v))}
                />
                <BoolSwitch
                    label={t('semantic.label')}
                    tip={t('semantic.tip')}
                    value={getBool(SettingKey.SemanticRouteEnabled, false)}
                    onChange={(v) => save(SettingKey.SemanticRouteEnabled, String(v))}
                />
            </div>

            <div className="pt-2 border-t border-border space-y-4">
                <div className="flex items-center gap-2 text-sm font-medium">
                    <Settings2 className="h-4 w-4 text-muted-foreground" />
                    {t('stream.title')}
                </div>
                <NumField
                    icon={<Gauge className="h-4 w-4 text-muted-foreground" />}
                    label={t('stream.maxEvents')}
                    tip={t('stream.maxEventsTip')}
                    value={getNum(SettingKey.StreamLogMaxEvents, 32)}
                    min={0}
                    max={10000}
                    onCommit={(v) => save(SettingKey.StreamLogMaxEvents, String(v))}
                />
                <NumField
                    icon={<Timer className="h-4 w-4 text-muted-foreground" />}
                    label={t('stream.maxBytes')}
                    tip={t('stream.maxBytesTip')}
                    value={getNum(SettingKey.StreamLogMaxBytes, 65536)}
                    min={0}
                    max={16777216}
                    onCommit={(v) => save(SettingKey.StreamLogMaxBytes, String(v))}
                />
            </div>
        </div>
    );
}

export function SettingChaos() {
    const t = useTranslations('setting.chaos');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();
    const saving = useRef(false);

    const get = (key: string, def = '') =>
        settings?.find((s) => s.key === key)?.value ?? def;
    const getBool = (key: string, def = false) => {
        const v = get(key);
        if (v === '') return def;
        return v === 'true';
    };
    const getNum = (key: string, def: number) => {
        const n = parseInt(get(key), 10);
        return Number.isNaN(n) ? def : n;
    };

    const save = async (key: string, value: string) => {
        if (saving.current) return;
        saving.current = true;
        try {
            await setSetting.mutateAsync({ key, value });
            toast.success(t('saved'));
        } catch (err) {
            toast.error(err instanceof Error ? err.message : String(err));
        } finally {
            saving.current = false;
        }
    };

    const enabled = getBool(SettingKey.ChaosEnabled, false);

    return (
        <div className="rounded-3xl border border-border bg-card p-5 space-y-5">
            <div className="flex items-start gap-3">
                <Zap className="h-5 w-5 text-amber-500 mt-0.5" />
                <div>
                    <h3 className="font-semibold text-base flex items-center gap-2">
                        {t('title')}
                        {enabled && (
                            <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-600 font-medium">
                                ON
                            </span>
                        )}
                    </h3>
                    <p className="text-xs text-muted-foreground mt-1 leading-relaxed">{t('hint')}</p>
                </div>
            </div>

            {enabled && (
                <div className="flex items-start gap-2 rounded-xl bg-amber-500/10 border border-amber-500/20 p-3 text-xs text-amber-700 dark:text-amber-400">
                    <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                    <span>{t('warning')}</span>
                </div>
            )}

            <div className="space-y-4">
                <BoolSwitch
                    label={t('enabled')}
                    tip={t('enabledTip')}
                    value={enabled}
                    onChange={(v) => save(SettingKey.ChaosEnabled, String(v))}
                />
                <NumField
                    icon={<Timer className="h-4 w-4 text-muted-foreground" />}
                    label={t('delayMs')}
                    tip={t('delayMsTip')}
                    value={getNum(SettingKey.ChaosDelayMS, 0)}
                    min={0}
                    max={30000}
                    onCommit={(v) => save(SettingKey.ChaosDelayMS, String(v))}
                />
                <NumField
                    icon={<AlertTriangle className="h-4 w-4 text-muted-foreground" />}
                    label={t('errorRate')}
                    tip={t('errorRateTip')}
                    value={getNum(SettingKey.ChaosErrorRate, 0)}
                    min={0}
                    max={100}
                    onCommit={(v) => save(SettingKey.ChaosErrorRate, String(v))}
                />
                <NumField
                    icon={<Zap className="h-4 w-4 text-muted-foreground" />}
                    label={t('dropRate')}
                    tip={t('dropRateTip')}
                    value={getNum(SettingKey.ChaosDropRate, 0)}
                    min={0}
                    max={100}
                    onCommit={(v) => save(SettingKey.ChaosDropRate, String(v))}
                />
            </div>
        </div>
    );
}
