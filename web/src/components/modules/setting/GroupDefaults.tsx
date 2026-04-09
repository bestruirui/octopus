'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Clock3, TimerReset, Repeat } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';

export function SettingGroupDefaults() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [firstTokenTimeOut, setFirstTokenTimeOut] = useState('');
    const [sessionKeepTime, setSessionKeepTime] = useState('');

    const initialFirstTokenTimeOut = useRef('0');
    const initialSessionKeepTime = useRef('0');

    useEffect(() => {
        if (!settings) return;

        const firstToken = settings.find((item) => item.key === SettingKey.GroupDefaultFirstTokenTimeOut);
        const sessionKeep = settings.find((item) => item.key === SettingKey.GroupDefaultSessionKeepTime);

        if (firstToken) {
            queueMicrotask(() => setFirstTokenTimeOut(firstToken.value));
            initialFirstTokenTimeOut.current = firstToken.value;
        }
        if (sessionKeep) {
            queueMicrotask(() => setSessionKeepTime(sessionKeep.value));
            initialSessionKeepTime.current = sessionKeep.value;
        }
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string) => {
        if (value === initialValue) return;

        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    if (key === SettingKey.GroupDefaultFirstTokenTimeOut) {
                        initialFirstTokenTimeOut.current = value;
                    } else if (key === SettingKey.GroupDefaultSessionKeepTime) {
                        initialSessionKeepTime.current = value;
                    }
                },
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Repeat className="h-5 w-5" />
                {t('groupDefaults.title')}
            </h2>
            <p className="text-sm text-muted-foreground">{t('groupDefaults.hint')}</p>

            <div className="space-y-4">
                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <TimerReset className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('groupDefaults.firstTokenTimeOut.label')}</span>
                    </div>
                    <Input
                        type="number"
                        min="0"
                        value={firstTokenTimeOut}
                        onChange={(event) => setFirstTokenTimeOut(event.target.value)}
                        onBlur={() => handleSave(SettingKey.GroupDefaultFirstTokenTimeOut, firstTokenTimeOut || '0', initialFirstTokenTimeOut.current)}
                        placeholder={t('groupDefaults.firstTokenTimeOut.placeholder')}
                        className="w-48 rounded-xl"
                    />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Clock3 className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('groupDefaults.sessionKeepTime.label')}</span>
                    </div>
                    <Input
                        type="number"
                        min="0"
                        value={sessionKeepTime}
                        onChange={(event) => setSessionKeepTime(event.target.value)}
                        onBlur={() => handleSave(SettingKey.GroupDefaultSessionKeepTime, sessionKeepTime || '0', initialSessionKeepTime.current)}
                        placeholder={t('groupDefaults.sessionKeepTime.placeholder')}
                        className="w-48 rounded-xl"
                    />
                </div>
            </div>
        </div>
    );
}
