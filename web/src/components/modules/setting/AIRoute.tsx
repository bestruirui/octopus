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

    const initialGroupID = useRef('0');
    const initialBaseURL = useRef('');
    const initialAPIKey = useRef('');
    const initialModel = useRef('');
    const initialTimeoutSeconds = useRef('180');

    useEffect(() => {
        if (!settings) return;

        const groupSetting = settings.find((item) => item.key === SettingKey.AIRouteGroupID);
        const baseURLSetting = settings.find((item) => item.key === SettingKey.AIRouteBaseURL);
        const apiKeySetting = settings.find((item) => item.key === SettingKey.AIRouteAPIKey);
        const modelSetting = settings.find((item) => item.key === SettingKey.AIRouteModel);
        const timeoutSetting = settings.find((item) => item.key === SettingKey.AIRouteTimeoutSeconds);

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

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Bot className="h-5 w-5" />
                {t('aiRoute.title')}
            </h2>

            <div className="space-y-2">
                <div className="flex items-center justify-between gap-4">
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
                        <SelectTrigger className="w-72 rounded-xl">
                            <SelectValue placeholder={t('aiRoute.group.placeholder')} />
                        </SelectTrigger>
                        <SelectContent className="rounded-xl">
                            <SelectItem value="0">{t('aiRoute.group.placeholder')}</SelectItem>
                            {groups.map((group) => (
                                <SelectItem key={group.id} value={String(group.id)}>
                                    {group.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
                <p className="pl-8 text-xs text-muted-foreground">
                    {t('aiRoute.group.hint')}
                </p>
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Link2 className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('aiRoute.baseUrl.label')}</span>
                </div>
                <Input
                    value={baseURL}
                    onChange={(event) => setBaseURL(event.target.value)}
                    onBlur={() => saveSetting(SettingKey.AIRouteBaseURL, baseURL, initialBaseURL)}
                    placeholder={t('aiRoute.baseUrl.placeholder')}
                    className="w-72 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
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
                    className="w-72 rounded-xl"
                />
            </div>

            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Bot className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('aiRoute.model.label')}</span>
                </div>
                <Input
                    value={model}
                    onChange={(event) => setModel(event.target.value)}
                    onBlur={() => saveSetting(SettingKey.AIRouteModel, model, initialModel)}
                    placeholder={t('aiRoute.model.placeholder')}
                    className="w-72 rounded-xl"
                />
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between gap-4">
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
                        className="w-72 rounded-xl"
                    />
                </div>
                <p className="pl-8 text-xs text-muted-foreground">
                    {t('aiRoute.timeoutSeconds.hint')}
                </p>
            </div>
        </div>
    );
}
