'use client';

import { useEffect, useState, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { RefreshCw, HelpCircle } from 'lucide-react';
import { Switch } from '@/components/ui/switch';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

export function SettingFailover() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [emptyResponseIsFailure, setEmptyResponseIsFailure] = useState(false);
    const initialEmptyResponseIsFailure = useRef(false);

    useEffect(() => {
        if (settings) {
            const s = settings.find(s => s.key === SettingKey.EmptyResponseIsFailure);
            if (s) {
                const val = s.value === 'true';
                queueMicrotask(() => setEmptyResponseIsFailure(val));
                initialEmptyResponseIsFailure.current = val;
            }
        }
    }, [settings]);

    const handleEmptyResponseChange = (checked: boolean) => {
        setEmptyResponseIsFailure(checked);
        setSetting.mutate(
            { key: SettingKey.EmptyResponseIsFailure, value: checked ? 'true' : 'false' },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    initialEmptyResponseIsFailure.current = checked;
                }
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <RefreshCw className="h-5 w-5" />
                {t('failover.title')}
            </h2>

            {/* 空响应视为失败 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <RefreshCw className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('failover.emptyResponseIsFailure.label')}</span>
                    <TooltipProvider>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent>
                                {t('failover.emptyResponseIsFailure.hint')}
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                </div>
                <Switch
                    checked={emptyResponseIsFailure}
                    onCheckedChange={handleEmptyResponseChange}
                />
            </div>
        </div>
    );
}
