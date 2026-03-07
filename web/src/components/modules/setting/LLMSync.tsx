'use client';

import { useEffect, useState, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { RefreshCw, Clock } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { useLastSyncTime, useSyncChannel } from '@/api/endpoints/channel';
import { toast } from '@/components/common/Toast';

export function SettingLLMSync() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();
    const syncChannel = useSyncChannel();
    const { data: lastSyncTime } = useLastSyncTime();

    const [syncInterval, setSyncInterval] = useState('');
    const initialSyncInterval = useRef('');

    useEffect(() => {
        if (settings) {
            const interval = settings.find(s => s.key === SettingKey.SyncLLMInterval);
            if (interval) {
                queueMicrotask(() => setSyncInterval(interval.value));
                initialSyncInterval.current = interval.value;
            }
        }
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string) => {
        if (value === initialValue) return;

        setSetting.mutate({ key, value }, {
            onSuccess: () => {
                toast.success(t('saved'));
                initialSyncInterval.current = value;
            }
        });
    };

    const handleManualSync = () => {
        syncChannel.mutate(undefined, {
            onSuccess: () => {
                toast.success(t('llmSync.syncSuccess'));
            },
            onError: () => {
                toast.error(t('llmSync.syncFailed'));
            }
        });
    };

    const formatLastSyncTime = (timeStr: string | undefined) => {
        if (!timeStr) return t('llmSync.neverSynced');
        const date = new Date(timeStr);
        if (date.getFullYear() === 1) return t('llmSync.neverSynced');
        return date.toLocaleString();
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <RefreshCw className="h-5 w-5" />
                {t('llmSync.title')}
            </h2>

            {/* 同步间隔 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('llmSync.syncInterval.label')}</span>
                </div>
                <Input
                    type="number"
                    value={syncInterval}
                    onChange={(e) => setSyncInterval(e.target.value)}
                    onBlur={() => handleSave(SettingKey.SyncLLMInterval, syncInterval, initialSyncInterval.current)}
                    placeholder={t('llmSync.syncInterval.placeholder')}
                    className="w-48 rounded-xl"
                />
            </div>

            {/* 手动同步 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-3">
                        <RefreshCw className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('llmSync.manualSync.label')}</span>
                    </div>
                    <span className="text-xs text-muted-foreground ml-8">
                        {t('llmSync.lastSync')}: {formatLastSyncTime(lastSyncTime)}
                    </span>
                </div>
                <Button
                    variant="outline"
                    size="sm"
                    onClick={handleManualSync}
                    disabled={syncChannel.isPending}
                    className="rounded-xl"
                >
                    {syncChannel.isPending ? t('llmSync.manualSync.syncing') : t('llmSync.manualSync.button')}
                </Button>
            </div>
        </div>
    );
}

