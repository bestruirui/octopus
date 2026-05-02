'use client';

import { useEffect, useRef, useState, type MutableRefObject } from 'react';
import { Bot, Clock3, KeyRound, Link2, Sparkles } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/input';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { useGroupList } from '@/api/endpoints/group';
import { SettingKey, useSetSetting, useSettingList } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';

export function SettingAIRoute() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const { data: groups = [] } = useGroupList();
    const setSetting = useSetSetting();

    const [groupID, setGroupID] = useState('0');
    const [baseURL, setBaseURL] = useState('');
    const [apiKey, setAPIKey] = useState('');
    const [model, setModel] = useState('');
    const [timeoutSeconds, setTimeoutSeconds] = useState('180');
    const [parallelism, setParallelism] = useState('3');
    const [servicesJSON, setServicesJSON] = useState('[]');

    const initialGroupID = useRef('0');
    const initialBaseURL = useRef('');
    const initialAPIKey = useRef('');
    const initialModel = useRef('');
    const initialTimeoutSeconds = useRef('180');
    const initialParallelism = useRef('3');
    const initialServicesJSON = useRef('[]');

    useEffect(() => {
        if (!settings) return;

        const groupSetting = settings.find((item) => item.key === SettingKey.AIRouteGroupID);
        const baseURLSetting = settings.find((item) => item.key === SettingKey.AIRouteBaseURL);
        const apiKeySetting = settings.find((item) => item.key === SettingKey.AIRouteAPIKey);
        const modelSetting = settings.find((item) => item.key === SettingKey.AIRouteModel);
        const timeoutSetting = settings.find((item) => item.key === SettingKey.AIRouteTimeoutSeconds);
        const parallelismSetting = settings.find((item) => item.key === SettingKey.AIRouteParallelism);
        const servicesSetting = settings.find((item) => item.key === SettingKey.AIRouteServices);

        if (groupSetting) {
            queueMicrotask(() => setGroupID(groupSetting.value || '0'));
            initialGroupID.current = groupSetting.value || '0';
        }
        if (baseURLSetting) {
            queueMicrotask(() => setBaseURL(baseURLSetting.value));
            initialBaseURL.current = baseURLSetting.value;
        }
        if (apiKeySetting) {
            queueMicrotask(() => setAPIKey(apiKeySetting.value));
            initialAPIKey.current = apiKeySetting.value;
        }
        if (modelSetting) {
            queueMicrotask(() => setModel(modelSetting.value));
            initialModel.current = modelSetting.value;
        }
        if (timeoutSetting) {
            queueMicrotask(() => setTimeoutSeconds(timeoutSetting.value || '180'));
            initialTimeoutSeconds.current = timeoutSetting.value || '180';
        }
        if (parallelismSetting) {
            queueMicrotask(() => setParallelism(parallelismSetting.value || '3'));
            initialParallelism.current = parallelismSetting.value || '3';
        }
        if (servicesSetting) {
            const nextValue = servicesSetting.value || '[]';
            queueMicrotask(() => setServicesJSON(nextValue));
            initialServicesJSON.current = nextValue;
        }
    }, [settings]);

    const saveSetting = (key: string, value: string, initialRef: MutableRefObject<string>) => {
        if (value === initialRef.current) return;

        setSetting.mutate(
            { key, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                    initialRef.current = value;
                },
            },
        );
    };

    const saveServicesSetting = () => {
        const normalizedValue = servicesJSON.trim() === '' ? '[]' : servicesJSON;
        if (normalizedValue === initialServicesJSON.current) {
            if (normalizedValue !== servicesJSON) {
                setServicesJSON(normalizedValue);
            }
            return;
        }

        try {
            const parsed = JSON.parse(normalizedValue);
            if (!Array.isArray(parsed)) {
                throw new Error('not-array');
            }
        } catch {
            toast.error(t('aiRoute.services.invalid'));
            return;
        }

        if (normalizedValue !== servicesJSON) {
            setServicesJSON(normalizedValue);
        }
        saveSetting(SettingKey.AIRouteServices, normalizedValue, initialServicesJSON);
    };

    return (
        <div className="waterhouse-island relative overflow-hidden rounded-[2.25rem] border-border/35 bg-card/62 p-6 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
            <div className="space-y-5">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div className="space-y-1.5">
                        <h2 className="flex items-center gap-2 text-lg font-bold text-card-foreground">
                            <Bot className="h-5 w-5" />
                            {t('aiRoute.title')}
                        </h2>
                        <p className="text-sm text-muted-foreground">{t('aiRoute.services.hint')}</p>
                    </div>
                    <div className="waterhouse-pod w-fit rounded-full border-border/25 bg-background/36 px-3 py-1.5 text-xs font-medium text-muted-foreground shadow-waterhouse-soft">
                        Command Habitat
                    </div>
                </div>

                <div className="grid gap-4 xl:grid-cols-2">
                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/38 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <Sparkles className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.group.label')}</span>
                        </div>
                        <Select
                            value={groupID}
                            onValueChange={(value) => {
                                setGroupID(value);
                                saveSetting(SettingKey.AIRouteGroupID, value, initialGroupID);
                            }}
                        >
                            <SelectTrigger className="w-full rounded-[1.2rem]">
                                <SelectValue placeholder={t('aiRoute.group.placeholder')} />
                            </SelectTrigger>
                            <SelectContent className="rounded-[1.2rem]">
                                <SelectItem value="0">{t('aiRoute.group.placeholder')}</SelectItem>
                                {groups.map((group) => (
                                    <SelectItem key={group.id} value={String(group.id)}>
                                        {group.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <p className="text-xs text-muted-foreground">{t('aiRoute.group.hint')}</p>
                    </div>

                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <Bot className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.model.label')}</span>
                        </div>
                        <Input
                            value={model}
                            onChange={(event) => setModel(event.target.value)}
                            onBlur={() => saveSetting(SettingKey.AIRouteModel, model, initialModel)}
                            placeholder={t('aiRoute.model.placeholder')}
                            className="w-full rounded-[1.2rem]"
                        />
                    </div>

                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <Link2 className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.baseUrl.label')}</span>
                        </div>
                        <Input
                            value={baseURL}
                            onChange={(event) => setBaseURL(event.target.value)}
                            onBlur={() => saveSetting(SettingKey.AIRouteBaseURL, baseURL, initialBaseURL)}
                            placeholder={t('aiRoute.baseUrl.placeholder')}
                            className="w-full rounded-[1.2rem]"
                        />
                    </div>

                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/34 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <KeyRound className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.apiKey.label')}</span>
                        </div>
                        <Input
                            type="password"
                            value={apiKey}
                            onChange={(event) => setAPIKey(event.target.value)}
                            onBlur={() => saveSetting(SettingKey.AIRouteAPIKey, apiKey, initialAPIKey)}
                            placeholder={t('aiRoute.apiKey.placeholder')}
                            className="w-full rounded-[1.2rem]"
                        />
                    </div>

                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/38 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <Clock3 className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.timeoutSeconds.label')}</span>
                        </div>
                        <Input
                            type="number"
                            min="1"
                            value={timeoutSeconds}
                            onChange={(event) => setTimeoutSeconds(event.target.value)}
                            onBlur={() =>
                                saveSetting(
                                    SettingKey.AIRouteTimeoutSeconds,
                                    timeoutSeconds,
                                    initialTimeoutSeconds,
                                )
                            }
                            placeholder={t('aiRoute.timeoutSeconds.placeholder')}
                            className="w-full rounded-[1.2rem]"
                        />
                        <p className="text-xs text-muted-foreground">{t('aiRoute.timeoutSeconds.hint')}</p>
                    </div>

                    <div className="waterhouse-pod space-y-3 rounded-[1.85rem] border-border/30 bg-background/38 p-4 shadow-waterhouse-soft">
                        <div className="flex items-center gap-3">
                            <Sparkles className="h-5 w-5 text-muted-foreground" />
                            <span className="text-sm font-medium">{t('aiRoute.parallelism.label')}</span>
                        </div>
                        <Input
                            type="number"
                            min="1"
                            value={parallelism}
                            onChange={(event) => setParallelism(event.target.value)}
                            onBlur={() => saveSetting(SettingKey.AIRouteParallelism, parallelism, initialParallelism)}
                            placeholder={t('aiRoute.parallelism.placeholder')}
                            className="w-full rounded-[1.2rem]"
                        />
                        <p className="text-xs text-muted-foreground">{t('aiRoute.parallelism.hint')}</p>
                    </div>
                </div>

                <div className="waterhouse-pod space-y-3 rounded-[1.95rem] border-border/30 bg-background/36 p-4 shadow-waterhouse-soft">
                    <div className="flex items-center gap-3">
                        <Link2 className="h-5 w-5 text-muted-foreground" />
                        <span className="text-sm font-medium">{t('aiRoute.services.label')}</span>
                    </div>
                    <textarea
                        value={servicesJSON}
                        onChange={(event) => setServicesJSON(event.target.value)}
                        onBlur={saveServicesSetting}
                        placeholder={t('aiRoute.services.placeholder')}
                        className="waterhouse-liquid-field min-h-44 w-full rounded-[1.4rem] border border-border/35 bg-background/62 px-4 py-3 font-mono text-sm text-foreground shadow-inner outline-none transition-[border-color,box-shadow] duration-300 focus-visible:border-ring focus-visible:ring-4 focus-visible:ring-ring/20"
                    />
                    <p className="text-xs text-muted-foreground">{t('aiRoute.services.hint')}</p>
                </div>
            </div>
        </div>
    );
}
