'use client';

import { useEffect, useRef, useState } from 'react';
import { Bot, Clock3, Database, HardDrive, KeyRound, Link2, Percent, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { SettingKey, useSetSetting, useSettingList } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';

const defaultSettingValues: Record<string, string> = {
    [SettingKey.SemanticCacheEnabled]: 'false',
    [SettingKey.SemanticCacheTTL]: '3600',
    [SettingKey.SemanticCacheThreshold]: '98',
    [SettingKey.SemanticCacheMaxEntries]: '1000',
    [SettingKey.SemanticCacheEmbeddingBaseURL]: '',
    [SettingKey.SemanticCacheEmbeddingAPIKey]: '',
    [SettingKey.SemanticCacheEmbeddingModel]: '',
    [SettingKey.SemanticCacheEmbeddingTimeoutSeconds]: '10',
};

export function SettingSemanticCache() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [enabled, setEnabled] = useState(false);
    const [ttl, setTTL] = useState('3600');
    const [threshold, setThreshold] = useState('98');
    const [maxEntries, setMaxEntries] = useState('1000');
    const [baseURL, setBaseURL] = useState('');
    const [apiKey, setAPIKey] = useState('');
    const [model, setModel] = useState('');
    const [timeoutSeconds, setTimeoutSeconds] = useState('10');

    const intendedValuesRef = useRef<Record<string, string>>({ ...defaultSettingValues });
    const loadedKeysRef = useRef<Set<string>>(new Set());
    const hasLocalIntentRef = useRef<Record<string, boolean>>({});
    const inFlightValuesRef = useRef<Record<string, string | undefined>>({});

    useEffect(() => {
        if (!settings) return;

        const nextValues: Record<string, string> = {
            [SettingKey.SemanticCacheEnabled]:
                settings.find((item) => item.key === SettingKey.SemanticCacheEnabled)?.value || 'false',
            [SettingKey.SemanticCacheTTL]:
                settings.find((item) => item.key === SettingKey.SemanticCacheTTL)?.value || '3600',
            [SettingKey.SemanticCacheThreshold]:
                settings.find((item) => item.key === SettingKey.SemanticCacheThreshold)?.value || '98',
            [SettingKey.SemanticCacheMaxEntries]:
                settings.find((item) => item.key === SettingKey.SemanticCacheMaxEntries)?.value || '1000',
            [SettingKey.SemanticCacheEmbeddingBaseURL]:
                settings.find((item) => item.key === SettingKey.SemanticCacheEmbeddingBaseURL)?.value || '',
            [SettingKey.SemanticCacheEmbeddingAPIKey]:
                settings.find((item) => item.key === SettingKey.SemanticCacheEmbeddingAPIKey)?.value || '',
            [SettingKey.SemanticCacheEmbeddingModel]:
                settings.find((item) => item.key === SettingKey.SemanticCacheEmbeddingModel)?.value || '',
            [SettingKey.SemanticCacheEmbeddingTimeoutSeconds]:
                settings.find((item) => item.key === SettingKey.SemanticCacheEmbeddingTimeoutSeconds)?.value || '10',
        };

        const shouldApplyServerValue = (key: string, nextValue: string) => {
            if (!loadedKeysRef.current.has(key)) {
                loadedKeysRef.current.add(key);
                return true;
            }

            if (hasLocalIntentRef.current[key] && intendedValuesRef.current[key] !== nextValue) {
                return false;
            }

            hasLocalIntentRef.current[key] = false;
            return true;
        };

        queueMicrotask(() => {
            if (shouldApplyServerValue(SettingKey.SemanticCacheEnabled, nextValues[SettingKey.SemanticCacheEnabled])) {
                intendedValuesRef.current[SettingKey.SemanticCacheEnabled] = nextValues[SettingKey.SemanticCacheEnabled];
                setEnabled(nextValues[SettingKey.SemanticCacheEnabled] === 'true');
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheTTL, nextValues[SettingKey.SemanticCacheTTL])) {
                intendedValuesRef.current[SettingKey.SemanticCacheTTL] = nextValues[SettingKey.SemanticCacheTTL];
                setTTL(nextValues[SettingKey.SemanticCacheTTL]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheThreshold, nextValues[SettingKey.SemanticCacheThreshold])) {
                intendedValuesRef.current[SettingKey.SemanticCacheThreshold] = nextValues[SettingKey.SemanticCacheThreshold];
                setThreshold(nextValues[SettingKey.SemanticCacheThreshold]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheMaxEntries, nextValues[SettingKey.SemanticCacheMaxEntries])) {
                intendedValuesRef.current[SettingKey.SemanticCacheMaxEntries] = nextValues[SettingKey.SemanticCacheMaxEntries];
                setMaxEntries(nextValues[SettingKey.SemanticCacheMaxEntries]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheEmbeddingBaseURL, nextValues[SettingKey.SemanticCacheEmbeddingBaseURL])) {
                intendedValuesRef.current[SettingKey.SemanticCacheEmbeddingBaseURL] = nextValues[SettingKey.SemanticCacheEmbeddingBaseURL];
                setBaseURL(nextValues[SettingKey.SemanticCacheEmbeddingBaseURL]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheEmbeddingAPIKey, nextValues[SettingKey.SemanticCacheEmbeddingAPIKey])) {
                intendedValuesRef.current[SettingKey.SemanticCacheEmbeddingAPIKey] = nextValues[SettingKey.SemanticCacheEmbeddingAPIKey];
                setAPIKey(nextValues[SettingKey.SemanticCacheEmbeddingAPIKey]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheEmbeddingModel, nextValues[SettingKey.SemanticCacheEmbeddingModel])) {
                intendedValuesRef.current[SettingKey.SemanticCacheEmbeddingModel] = nextValues[SettingKey.SemanticCacheEmbeddingModel];
                setModel(nextValues[SettingKey.SemanticCacheEmbeddingModel]);
            }

            if (shouldApplyServerValue(SettingKey.SemanticCacheEmbeddingTimeoutSeconds, nextValues[SettingKey.SemanticCacheEmbeddingTimeoutSeconds])) {
                intendedValuesRef.current[SettingKey.SemanticCacheEmbeddingTimeoutSeconds] = nextValues[SettingKey.SemanticCacheEmbeddingTimeoutSeconds];
                setTimeoutSeconds(nextValues[SettingKey.SemanticCacheEmbeddingTimeoutSeconds]);
            }
        });
    }, [settings]);

    const flushSettingSave = (key: string) => {
        if (inFlightValuesRef.current[key] !== undefined || !hasLocalIntentRef.current[key]) {
            return;
        }

        const value = intendedValuesRef.current[key];
        inFlightValuesRef.current[key] = value;

        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                },
                onError: () => {
                    if (intendedValuesRef.current[key] === value) {
                        hasLocalIntentRef.current[key] = false;
                    }
                },
                onSettled: () => {
                    delete inFlightValuesRef.current[key];
                    if (hasLocalIntentRef.current[key] && intendedValuesRef.current[key] !== value) {
                        flushSettingSave(key);
                    }
                },
            },
        );
    };

    const saveTextSetting = (key: string, value: string) => {
        if (value === intendedValuesRef.current[key]) return;

        intendedValuesRef.current[key] = value;
        hasLocalIntentRef.current[key] = true;
        flushSettingSave(key);
    };

    const saveBooleanSetting = (checked: boolean) => {
        const value = checked ? 'true' : 'false';
        setEnabled(checked);
        if (value === intendedValuesRef.current[SettingKey.SemanticCacheEnabled]) return;

        intendedValuesRef.current[SettingKey.SemanticCacheEnabled] = value;
        hasLocalIntentRef.current[SettingKey.SemanticCacheEnabled] = true;
        flushSettingSave(SettingKey.SemanticCacheEnabled);
    };

    return (
        <div className="waterhouse-island rounded-[2.1rem] border-border/35 bg-card/58 p-6 space-y-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
            <div className="space-y-1">
                <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                    <Database className="h-5 w-5" />
                    {t('semanticCache.title')}
                </h2>
                <p className="text-sm text-muted-foreground">
                    {t('semanticCache.description')}
                </p>
            </div>

            <div className="waterhouse-pod flex items-center justify-between gap-4 rounded-[1.55rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                <div className="flex items-center gap-3">
                    <Sparkles className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('semanticCache.enabled.label')}</span>
                </div>
                <Switch checked={enabled} onCheckedChange={saveBooleanSetting} />
            </div>

            <div className="waterhouse-pod space-y-2 rounded-[1.55rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                        <Clock3 className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.ttl.label')}</span>
                    </div>
                    <Input
                        type="number"
                        min="1"
                        value={ttl}
                        onChange={(event) => setTTL(event.target.value)}
                        onBlur={() => saveTextSetting(SettingKey.SemanticCacheTTL, ttl)}
                        placeholder={t('semanticCache.ttl.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>
                <p className="pl-8 text-xs text-muted-foreground">
                    {t('semanticCache.ttl.hint')}
                </p>
            </div>

            <div className="waterhouse-pod space-y-2 rounded-[1.55rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                        <Percent className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.threshold.label')}</span>
                    </div>
                    <Input
                        type="number"
                        min="0"
                        max="100"
                        value={threshold}
                        onChange={(event) => setThreshold(event.target.value)}
                        onBlur={() => saveTextSetting(SettingKey.SemanticCacheThreshold, threshold)}
                        placeholder={t('semanticCache.threshold.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>
                <p className="pl-8 text-xs text-muted-foreground">
                    {t('semanticCache.threshold.hint')}
                </p>
            </div>

            <div className="waterhouse-pod space-y-2 rounded-[1.55rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-3">
                        <HardDrive className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.maxEntries.label')}</span>
                    </div>
                    <Input
                        type="number"
                        min="1"
                        value={maxEntries}
                        onChange={(event) => setMaxEntries(event.target.value)}
                        onBlur={() => saveTextSetting(SettingKey.SemanticCacheMaxEntries, maxEntries)}
                        placeholder={t('semanticCache.maxEntries.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>
                <p className="pl-8 text-xs text-muted-foreground">
                    {t('semanticCache.maxEntries.hint')}
                </p>
            </div>

            <div className="waterhouse-pod space-y-5 rounded-[1.8rem] border-border/30 bg-background/30 p-5 shadow-waterhouse-soft">
                <div className="space-y-1">
                    <p className="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                        {t('semanticCache.embedding.title')}
                    </p>
                    <p className="text-xs text-muted-foreground">
                        {t('semanticCache.embedding.description')}
                    </p>
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Link2 className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.baseUrl.label')}</span>
                    </div>
                    <Input
                        value={baseURL}
                        onChange={(event) => setBaseURL(event.target.value)}
                        onBlur={() =>
                            saveTextSetting(
                                SettingKey.SemanticCacheEmbeddingBaseURL,
                                baseURL,
                            )
                        }
                        placeholder={t('semanticCache.baseUrl.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <KeyRound className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.apiKey.label')}</span>
                    </div>
                    <Input
                        type="password"
                        value={apiKey}
                        onChange={(event) => setAPIKey(event.target.value)}
                        onBlur={() =>
                            saveTextSetting(
                                SettingKey.SemanticCacheEmbeddingAPIKey,
                                apiKey,
                            )
                        }
                        placeholder={t('semanticCache.apiKey.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>

                <div className="flex items-center justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <Bot className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('semanticCache.model.label')}</span>
                    </div>
                    <Input
                        value={model}
                        onChange={(event) => setModel(event.target.value)}
                        onBlur={() =>
                            saveTextSetting(
                                SettingKey.SemanticCacheEmbeddingModel,
                                model,
                            )
                        }
                        placeholder={t('semanticCache.model.placeholder')}
                        className="w-72 rounded-xl"
                    />
                </div>

                <div className="space-y-2">
                    <div className="flex items-center justify-between gap-4">
                        <div className="flex items-center gap-3">
                            <Clock3 className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('semanticCache.timeoutSeconds.label')}</span>
                        </div>
                        <Input
                            type="number"
                            min="1"
                            value={timeoutSeconds}
                            onChange={(event) => setTimeoutSeconds(event.target.value)}
                            onBlur={() =>
                                saveTextSetting(
                                    SettingKey.SemanticCacheEmbeddingTimeoutSeconds,
                                    timeoutSeconds,
                                )
                            }
                            placeholder={t('semanticCache.timeoutSeconds.placeholder')}
                            className="w-72 rounded-xl"
                        />
                    </div>
                    <p className="pl-8 text-xs text-muted-foreground">
                        {t('semanticCache.timeoutSeconds.hint')}
                    </p>
                </div>
            </div>
        </div>
    );
}
