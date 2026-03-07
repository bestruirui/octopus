'use client';

import { useEffect, useState, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { DollarSign, Clock, RefreshCw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { useUpdateModelPrice, useLastUpdateTime } from '@/api/endpoints/model';
import { toast } from '@/components/common/Toast';

export function SettingLLMPrice() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();
    const updatePrice = useUpdateModelPrice();
    const { data: lastUpdateTime } = useLastUpdateTime();

    const [updateInterval, setUpdateInterval] = useState('');
    const initialUpdateInterval = useRef('');

    useEffect(() => {
        if (settings) {
            const interval = settings.find(s => s.key === SettingKey.ModelInfoUpdateInterval);
            if (interval) {
                queueMicrotask(() => setUpdateInterval(interval.value));
                initialUpdateInterval.current = interval.value;
            }
        }
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string) => {
        if (value === initialValue) return;

        setSetting.mutate({ key, value }, {
            onSuccess: () => {
                toast.success(t('saved'));
                initialUpdateInterval.current = value;
            }
        });
    };

    const handleManualUpdate = () => {
        updatePrice.mutate(undefined, {
            onSuccess: () => {
                toast.success(t('llmPrice.updateSuccess'));
            },
            onError: () => {
                toast.error(t('llmPrice.updateFailed'));
            }
        });
    };

    const formatLastUpdateTime = (timeStr: string | undefined) => {
        if (!timeStr) return t('llmPrice.neverUpdated');
        const date = new Date(timeStr);
        if (date.getFullYear() === 1) return t('llmPrice.neverUpdated');
        return date.toLocaleString();
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <DollarSign className="h-5 w-5" />
                {t('llmPrice.title')}
            </h2>

            {/* 更新间隔 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('llmPrice.updateInterval.label')}</span>
                </div>
                <Input
                    type="number"
                    value={updateInterval}
                    onChange={(e) => setUpdateInterval(e.target.value)}
                    onBlur={() => handleSave(SettingKey.ModelInfoUpdateInterval, updateInterval, initialUpdateInterval.current)}
                    placeholder={t('llmPrice.updateInterval.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            {/* 手动更新 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-3">
                        <RefreshCw className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('llmPrice.manualUpdate.label')}</span>
                    </div>
                    <span className="text-xs text-muted-foreground ml-8">
                        {t('llmPrice.lastUpdate')}: {formatLastUpdateTime(lastUpdateTime)}
                    </span>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleManualUpdate}
                    disabled={updatePrice.isPending}
                    className="rounded-xl"
                >
                    {updatePrice.isPending ? t('llmPrice.manualUpdate.updating') : t('llmPrice.manualUpdate.button')}
                </Button>
            </div>
        </div>
    );
}

