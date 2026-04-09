'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { SlidersHorizontal, Gauge, TimerReset, KeyRound, Cable, ArrowBigUp, Scale, ShieldAlert, BadgeDollarSign, Sparkles } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';

type HealthScoreWeights = {
    success_rate: number;
    avg_wait: number;
    key_availability: number;
    base_delay: number;
    priority_boost: number;
    weight_boost: number;
    recent_use_bonus: number;
    rate_limit_penalty: number;
    cost_penalty: number;
    cold_start_score: number;
};

const DEFAULT_WEIGHTS: HealthScoreWeights = {
    success_rate: 70,
    avg_wait: 20,
    key_availability: 10,
    base_delay: 10,
    priority_boost: 10,
    weight_boost: 0.5,
    recent_use_bonus: 5,
    rate_limit_penalty: 30,
    cost_penalty: 2,
    cold_start_score: 70,
};

const FIELD_META = [
    { key: 'success_rate', icon: Gauge },
    { key: 'avg_wait', icon: TimerReset },
    { key: 'key_availability', icon: KeyRound },
    { key: 'base_delay', icon: Cable },
    { key: 'priority_boost', icon: ArrowBigUp },
    { key: 'weight_boost', icon: Scale },
    { key: 'recent_use_bonus', icon: Sparkles },
    { key: 'rate_limit_penalty', icon: ShieldAlert },
    { key: 'cost_penalty', icon: BadgeDollarSign },
    { key: 'cold_start_score', icon: Gauge },
] as const;

function normalizeWeights(raw: string | undefined) {
    if (!raw) return DEFAULT_WEIGHTS;
    try {
        return { ...DEFAULT_WEIGHTS, ...JSON.parse(raw) } as HealthScoreWeights;
    } catch {
        return DEFAULT_WEIGHTS;
    }
}

export function SettingHealthScore() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [weights, setWeights] = useState<HealthScoreWeights>(DEFAULT_WEIGHTS);
    const initialValue = useRef(JSON.stringify(DEFAULT_WEIGHTS));

    useEffect(() => {
        if (!settings) return;
        const item = settings.find((s) => s.key === SettingKey.HealthScoreWeights);
        const next = normalizeWeights(item?.value);
        // Settings are loaded from the server and need to hydrate the local editable form state.
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setWeights(next);
        initialValue.current = JSON.stringify(next);
    }, [settings]);

    const save = (next: HealthScoreWeights) => {
        const payload = JSON.stringify(next);
        if (payload === initialValue.current) return;
        setSetting.mutate(
            { key: SettingKey.HealthScoreWeights, value: payload },
            {
                onSuccess: () => {
                    initialValue.current = payload;
                    toast.success(t('saved'));
                },
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <SlidersHorizontal className="h-5 w-5" />
                {t('healthScore.title')}
            </h2>
            <p className="text-sm text-muted-foreground">{t('healthScore.hint')}</p>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                {FIELD_META.map(({ key, icon: Icon }) => (
                    <div key={key} className="flex items-center justify-between gap-4 rounded-2xl border border-border/60 px-4 py-3">
                        <div className="flex items-center gap-3 min-w-0">
                            <Icon className="h-5 w-5 text-muted-foreground shrink-0" />
                            <div className="min-w-0">
                                <div className="text-sm font-medium">{t(`healthScore.fields.${key}.label`)}</div>
                                <div className="text-xs text-muted-foreground">{t(`healthScore.fields.${key}.description`)}</div>
                            </div>
                        </div>
                        <Input
                            type="number"
                            step="0.1"
                            min="0"
                            value={String(weights[key as keyof HealthScoreWeights])}
                            onChange={(e) => {
                                const next = Number(e.target.value);
                                setWeights((prev) => ({
                                    ...prev,
                                    [key]: Number.isFinite(next) && next >= 0 ? next : 0,
                                }));
                            }}
                            onBlur={() => save(weights)}
                            className="w-28 rounded-xl"
                        />
                    </div>
                ))}
            </div>
        </div>
    );
}
