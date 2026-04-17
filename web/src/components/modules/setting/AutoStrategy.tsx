'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Sparkles } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';

export function SettingAutoStrategy() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [minSamples, setMinSamples] = useState('');
    const [timeWindow, setTimeWindow] = useState('');
    const [sampleThreshold, setSampleThreshold] = useState('');

    const initialMinSamples = useRef('');
    const initialTimeWindow = useRef('');
    const initialSampleThreshold = useRef('');

    useEffect(() => {
        if (settings) {
            const ms = settings.find(s => s.key === SettingKey.AutoStrategyMinSamples);
            const tw = settings.find(s => s.key === SettingKey.AutoStrategyTimeWindow);
            const st = settings.find(s => s.key === SettingKey.AutoStrategySampleThreshold);

            if (ms) {
                queueMicrotask(() => setMinSamples(ms.value));
                initialMinSamples.current = ms.value;
            }
            if (tw) {
                queueMicrotask(() => setTimeWindow(tw.value));
                initialTimeWindow.current = tw.value;
            }
            if (st) {
                queueMicrotask(() => setSampleThreshold(st.value));
                initialSampleThreshold.current = st.value;
            }
        }
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string) => {
        if (value === initialValue) return;

        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                }
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Sparkles className="h-5 w-5" />
                {t('autoStrategy.title')}
            </h2>

            <p className="text-sm text-muted-foreground">
                {t('autoStrategy.hint')}
            </p>

            <div className="space-y-4">
                <div className="flex items-center justify-between gap-4">
                    <div className="flex flex-col gap-1">
                        <span className="text-sm font-medium">{t('autoStrategy.minSamples.label')}</span>
                        <span className="text-xs text-muted-foreground">{t('autoStrategy.minSamples.hint')}</span>
                    </div>
                    <Input
                        type="number"
                        min="1"
                        value={minSamples}
                        onChange={(e) => setMinSamples(e.target.value)}
                        onBlur={() => handleSave(SettingKey.AutoStrategyMinSamples, minSamples, initialMinSamples.current)}
                        placeholder={t('autoStrategy.minSamples.placeholder')}
                        className="w-32 rounded-xl"
                    />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex flex-col gap-1">
                        <span className="text-sm font-medium">{t('autoStrategy.timeWindow.label')}</span>
                        <span className="text-xs text-muted-foreground">{t('autoStrategy.timeWindow.hint')}</span>
                    </div>
                    <Input
                        type="number"
                        min="1"
                        value={timeWindow}
                        onChange={(e) => setTimeWindow(e.target.value)}
                        onBlur={() => handleSave(SettingKey.AutoStrategyTimeWindow, timeWindow, initialTimeWindow.current)}
                        placeholder={t('autoStrategy.timeWindow.placeholder')}
                        className="w-32 rounded-xl"
                    />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex flex-col gap-1">
                        <span className="text-sm font-medium">{t('autoStrategy.sampleThreshold.label')}</span>
                        <span className="text-xs text-muted-foreground">{t('autoStrategy.sampleThreshold.hint')}</span>
                    </div>
                    <Input
                        type="number"
                        min="1"
                        value={sampleThreshold}
                        onChange={(e) => setSampleThreshold(e.target.value)}
                        onBlur={() => handleSave(SettingKey.AutoStrategySampleThreshold, sampleThreshold, initialSampleThreshold.current)}
                        placeholder={t('autoStrategy.sampleThreshold.placeholder')}
                        className="w-32 rounded-xl"
                    />
                </div>
            </div>
        </div>
    );
}
