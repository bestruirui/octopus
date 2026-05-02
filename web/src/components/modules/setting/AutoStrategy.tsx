'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Sparkles } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { AUTO_STRATEGY_FIELDS } from './runtime-settings';

export function SettingAutoStrategy() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [values, setValues] = useState<Record<string, string>>({});
    const initialValues = useRef<Record<string, string>>({});

    useEffect(() => {
        if (!settings) return;

        const nextValues = AUTO_STRATEGY_FIELDS.reduce<Record<string, string>>((acc, field) => {
            acc[field.key] = settings.find((item) => item.key === field.key)?.value ?? '';
            return acc;
        }, {});

        queueMicrotask(() => setValues(nextValues));
        initialValues.current = nextValues;
    }, [settings]);

    const handleSave = (key: string) => {
        const value = values[key] ?? '';
        if (value === initialValues.current[key]) return;

        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    initialValues.current = {
                        ...initialValues.current,
                        [key]: value,
                    };
                }
            }
        );
    };

    return (
        <div className="waterhouse-island rounded-[2.1rem] border-border/35 bg-card/58 p-6 space-y-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Sparkles className="h-5 w-5" />
                {t('autoStrategy.title')}
            </h2>

            <p className="text-sm text-muted-foreground">
                {t('autoStrategy.hint')}
            </p>

            <div className="space-y-4">
                {AUTO_STRATEGY_FIELDS.map((field) => (
                    <div key={field.key} className="waterhouse-pod flex flex-col gap-3 rounded-[1.5rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft md:flex-row md:items-center md:justify-between">
                        <div className="flex flex-col gap-1">
                            <span className="text-sm font-medium">{t(field.labelKey)}</span>
                            {field.hintKey ? (
                                <span className="text-xs text-muted-foreground">{t(field.hintKey)}</span>
                            ) : null}
                        </div>
                        <Input
                            type="number"
                            min={field.min}
                            max={field.max}
                            value={values[field.key] ?? ''}
                            onChange={(e) => setValues((prev) => ({ ...prev, [field.key]: e.target.value }))}
                            onBlur={() => handleSave(field.key)}
                            placeholder={t(field.placeholderKey)}
                            className="w-32 rounded-xl"
                        />
                    </div>
                ))}
            </div>
        </div>
    );
}
