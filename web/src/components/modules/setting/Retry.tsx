'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { RotateCcw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';

export function SettingRetry() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [retryCount, setRetryCount] = useState('');
    const initialRetryCount = useRef('');

    useEffect(() => {
        if (settings) {
            const retry = settings.find(s => s.key === SettingKey.RelayRetryCount);
            if (retry) {
                queueMicrotask(() => setRetryCount(retry.value));
                initialRetryCount.current = retry.value;
            }
        }
    }, [settings]);

    const handleSave = () => {
        if (retryCount === initialRetryCount.current) return;

        setSetting.mutate(
            { key: SettingKey.RelayRetryCount, value: retryCount },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    initialRetryCount.current = retryCount;
                }
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <RotateCcw className="h-5 w-5" />
                {t('retry.title')}
            </h2>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <RotateCcw className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('retry.count.label')}</span>
                </div>
                <Input
                    type="number"
                    min="1"
                    value={retryCount}
                    onChange={(e) => setRetryCount(e.target.value)}
                    onBlur={handleSave}
                    placeholder={t('retry.count.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>
        </div>
    );
}
