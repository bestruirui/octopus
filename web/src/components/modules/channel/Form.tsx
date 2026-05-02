import {
    AutoGroupType,
    ChannelType,
    RequestRewriteProfile,
    SystemMessageStrategy,
    ToolRoleStrategy,
    type Channel,
    type RequestRewriteConfig,
    useFetchModel,
    useTestChannel,
    type TestChannelSummary,
} from '@/api/endpoints/channel';
import { channelTemplates } from './templates';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { toast } from '@/components/common/Toast';
import { useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef, useState } from 'react';
import { RefreshCw, X, Plus, FlaskConical, CheckCircle2, AlertTriangle, Sparkles, Orbit, Layers3, KeyRound, Cable, SlidersHorizontal } from 'lucide-react';

export interface ChannelKeyFormItem {
    id?: number;
    enabled: boolean;
    channel_key: string;
    status_code?: number;
    last_use_time_stamp?: number;
    total_cost?: number;
    remark?: string;
}

export interface ChannelFormData {
    name: string;
    type: ChannelType;
    base_urls: Channel['base_urls'];
    custom_header: Channel['custom_header'];
    channel_proxy: string;
    param_override: string;
    request_rewrite: RequestRewriteConfig;
    keys: ChannelKeyFormItem[];
    model: string;
    custom_model: string;
    enabled: boolean;
    proxy: boolean;
    auto_sync: boolean;
    auto_group: AutoGroupType;
    match_regex: string;
}

export function createDefaultRequestRewriteFormData(): RequestRewriteConfig {
    return {
        enabled: false,
        profile: RequestRewriteProfile.OpenAIChatCompat,
        tool_role_strategy: ToolRoleStrategy.Keep,
        system_message_strategy: SystemMessageStrategy.Keep,
    };
}

export function normalizeRequestRewriteFormData(config?: RequestRewriteConfig | null): RequestRewriteConfig {
    return {
        enabled: config?.enabled ?? false,
        profile: config?.profile ?? RequestRewriteProfile.OpenAIChatCompat,
        tool_role_strategy: config?.tool_role_strategy ?? ToolRoleStrategy.Keep,
        system_message_strategy: config?.system_message_strategy ?? SystemMessageStrategy.Keep,
    };
}

export function isRequestRewriteSupportedChannelType(channelType: ChannelType): boolean {
    return channelType === ChannelType.OpenAIChat || channelType === ChannelType.Mimo;
}

export function getEffectiveRequestRewriteFormData(channelType: ChannelType, config?: RequestRewriteConfig | null): RequestRewriteConfig {
    const normalized = normalizeRequestRewriteFormData(config);
    if (isRequestRewriteSupportedChannelType(channelType)) {
        return normalized;
    }

    return {
        ...normalized,
        enabled: false,
    };
}

export interface ChannelFormProps {
    formData: ChannelFormData;
    onFormDataChange: (data: ChannelFormData) => void;
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
    isPending: boolean;
    submitText: string;
    pendingText: string;
    onCancel?: () => void;
    cancelText?: string;
    idPrefix?: string;
}

import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";

function SectionHeader({
    icon: Icon,
    title,
    hint,
}: {
    icon: typeof Sparkles;
    title: string;
    hint?: string;
}) {
    return (
        <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="space-y-2">
                <div className="inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-primary shadow-waterhouse-soft">
                    <Icon className="size-3.5" />
                    {title}
                </div>
                {hint ? <p className="text-xs leading-5 text-muted-foreground">{hint}</p> : null}
            </div>
        </div>
    );
}

export function ChannelForm({
    formData,
    onFormDataChange,
    onSubmit,
    isPending,
    submitText,
    pendingText,
    onCancel,
    cancelText,
    idPrefix = 'channel',
}: ChannelFormProps) {
    const t = useTranslations('channel.form');
    const requestRewriteSupported = isRequestRewriteSupportedChannelType(formData.type);
    const sectionClassName = 'waterhouse-pod space-y-4 rounded-[1.8rem] border border-border/30 bg-background/34 p-4 shadow-waterhouse-soft md:p-5';
    const labelClassName = 'text-sm font-medium text-card-foreground';
    const fieldGroupClassName = 'space-y-2';

    // Ensure the form always shows at least 1 row for base_urls / keys / custom_header.
    // This avoids "empty list" UI and also keeps URL + APIKEY layout consistent.
    useEffect(() => {
        if (!formData.base_urls || formData.base_urls.length === 0) {
            onFormDataChange({ ...formData, base_urls: [{ url: '', delay: 0 }] });
            return;
        }
        if (!formData.keys || formData.keys.length === 0) {
            onFormDataChange({ ...formData, keys: [{ enabled: true, channel_key: '' }] });
            return;
        }
        if (!formData.custom_header || formData.custom_header.length === 0) {
            onFormDataChange({ ...formData, custom_header: [{ header_key: '', header_value: '' }] });
        }
    }, [formData, onFormDataChange]);

    const autoModels = formData.model
        ? formData.model.split(',').map((m) => m.trim()).filter(Boolean)
        : [];
    const customModels = formData.custom_model
        ? formData.custom_model.split(',').map((m) => m.trim()).filter(Boolean)
        : [];
    const [inputValue, setInputValue] = useState('');
    const inputRef = useRef<HTMLInputElement>(null);

    const fetchModel = useFetchModel();
    const testChannel = useTestChannel();
    const [testSummary, setTestSummary] = useState<TestChannelSummary | null>(null);

    const effectiveKey =
        formData.keys.find((k) => k.enabled && k.channel_key.trim())?.channel_key.trim() || '';

    const updateModels = (nextAuto: string[], nextCustom: string[]) => {
        const model = nextAuto.join(',');
        const custom_model = nextCustom.join(',');
        if (formData.model === model && formData.custom_model === custom_model) return;
        onFormDataChange({ ...formData, model, custom_model });
    };

    const normalizeFetchedModels = (data: unknown): string[] => {
        if (!Array.isArray(data)) return [];

        return Array.from(new Set(
            data
                .map((item) => {
                    if (typeof item === 'string') return item.trim();
                    if (!item || typeof item !== 'object') return '';

                    const candidate =
                        ('id' in item && typeof item.id === 'string' && item.id) ||
                        ('name' in item && typeof item.name === 'string' && item.name) ||
                        ('display_name' in item && typeof item.display_name === 'string' && item.display_name) ||
                        ('displayName' in item && typeof item.displayName === 'string' && item.displayName) ||
                        '';

                    return candidate.trim();
                })
                .filter(Boolean)
        ));
    };

    const normalizedHeaders = useMemo(() =>
        (formData.custom_header ?? [])
            .map((h) => ({ header_key: h.header_key.trim(), header_value: h.header_value }))
            .filter((h) => h.header_key && h.header_value !== ''),
        [formData.custom_header]
    );

    const buildTestPayload = () => ({
        type: formData.type,
        base_urls: (formData.base_urls ?? []).filter((u) => u.url.trim()).map((u) => ({
            url: u.url.trim(),
            delay: Number(u.delay || 0),
        })),
        keys: formData.keys
            .filter((k) => k.channel_key.trim())
            .map((k) => ({ enabled: k.enabled, channel_key: k.channel_key.trim(), remark: k.remark ?? '' })),
        proxy: formData.proxy,
        channel_proxy: formData.channel_proxy?.trim() || null,
        match_regex: formData.match_regex.trim() || null,
        custom_header: normalizedHeaders,
        model: formData.model,
        custom_model: formData.custom_model,
        name: formData.name,
        enabled: formData.enabled,
        auto_sync: formData.auto_sync,
        auto_group: formData.auto_group,
        param_override: formData.param_override.trim() || null,
    });

    const handleTestChannel = () => {
        setTestSummary(null);
        testChannel.mutate(buildTestPayload(), {
            onSuccess: (data) => {
                setTestSummary(data);
                toast.success(data.passed ? t('test.success') : t('test.partialSuccess'));
            },
            onError: (error) => {
                const errorMessage = error instanceof Error ? error.message : String(error);
                toast.error(t('test.failed'), { description: errorMessage });
            },
        });
    };

    const handleRefreshModels = async () => {
        if (!formData.base_urls?.[0]?.url || !effectiveKey) return;
        fetchModel.mutate(
            {
                type: formData.type,
                base_urls: formData.base_urls,
                keys: formData.keys
                    .filter((k) => k.channel_key.trim())
                    .map((k) => ({ enabled: k.enabled, channel_key: k.channel_key.trim() })),
                proxy: formData.proxy,
                channel_proxy: formData.channel_proxy?.trim() || null,
                match_regex: formData.match_regex.trim() || null,
                custom_header: normalizedHeaders,
            },
            {
                onSuccess: (data) => {
                    const normalizedModels = normalizeFetchedModels(data);
                    if (normalizedModels.length > 0) {
                        const nextAuto = Array.from(new Set([...autoModels, ...normalizedModels]));
                        updateModels(nextAuto, customModels);
                        toast.success(t('modelRefreshSuccess'));
                    } else {
                        toast.warning(t('modelRefreshEmpty'));
                    }
                },
                onError: (error) => {
                    const errorMessage = error instanceof Error ? error.message : String(error);
                    toast.error(t('modelRefreshFailed'), { description: errorMessage });
                },
            }
        );
    };

    const handleAddModel = (model: string) => {
        const trimmedModel = model.trim();
        if (trimmedModel && !customModels.includes(trimmedModel) && !autoModels.includes(trimmedModel)) {
            updateModels(autoModels, [...customModels, trimmedModel]);
        }
        setInputValue('');
    };

    const handleRemoveAutoModel = (model: string) => {
        updateModels(autoModels.filter(m => m !== model), customModels);
    };

    const handleRemoveCustomModel = (model: string) => {
        updateModels(autoModels, customModels.filter(m => m !== model));
    };

    const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (inputValue.trim()) handleAddModel(inputValue);
        }
    };

    const handleAddKey = () => {
        onFormDataChange({
            ...formData,
            keys: [...formData.keys, { enabled: true, channel_key: '' }],
        });
    };

    const handleUpdateKey = (idx: number, patch: Partial<ChannelKeyFormItem>) => {
        const next = formData.keys.map((k, i) => (i === idx ? { ...k, ...patch } : k));
        onFormDataChange({ ...formData, keys: next });
    };

    const handleRemoveKey = (idx: number) => {
        const curr = formData.keys ?? [];
        if (curr.length <= 1) return;
        const next = curr.filter((_, i) => i !== idx);
        onFormDataChange({ ...formData, keys: next });
    };

    const handleAddBaseUrl = () => {
        onFormDataChange({
            ...formData,
            base_urls: [...(formData.base_urls ?? []), { url: '', delay: 0 }],
        });
    };

    const handleUpdateBaseUrl = (idx: number, patch: Partial<Channel['base_urls'][number]>) => {
        const next = (formData.base_urls ?? []).map((u, i) => (i === idx ? { ...u, ...patch } : u));
        onFormDataChange({ ...formData, base_urls: next });
    };

    const handleRemoveBaseUrl = (idx: number) => {
        const curr = formData.base_urls ?? [];
        if (curr.length <= 1) return;
        onFormDataChange({ ...formData, base_urls: curr.filter((_, i) => i !== idx) });
    };

    const handleAddHeader = () => {
        onFormDataChange({
            ...formData,
            custom_header: [...(formData.custom_header ?? []), { header_key: '', header_value: '' }],
        });
    };

    const handleUpdateHeader = (idx: number, patch: Partial<Channel['custom_header'][number]>) => {
        const next = (formData.custom_header ?? []).map((h, i) => (i === idx ? { ...h, ...patch } : h));
        onFormDataChange({ ...formData, custom_header: next });
    };

    const handleRemoveHeader = (idx: number) => {
        const curr = formData.custom_header ?? [];
        if (curr.length <= 1) return;
        onFormDataChange({ ...formData, custom_header: curr.filter((_, i) => i !== idx) });
    };

    const handleApplyTemplate = (templateKey: string) => {
        const template = channelTemplates.find((item) => item.key === templateKey);
        if (!template) return;
        onFormDataChange(template.apply(formData));
        setTestSummary(null);
    };

    return (
        <form onSubmit={onSubmit} className="flex h-full min-h-0 flex-col">
            <div className="flex-1 min-h-0 overflow-y-auto px-1">
            <div className="space-y-4 pb-2">
            <section className={sectionClassName}>
                <SectionHeader icon={Sparkles} title={t('template.label')} hint={t('template.hint')} />
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
                    {channelTemplates.map((template) => (
                        <Button
                            key={template.key}
                            type="button"
                            variant="outline"
                            onClick={() => handleApplyTemplate(template.key)}
                            className="h-auto min-h-24 flex-col items-start gap-1 rounded-[1.55rem] border-border/30 bg-background/42 px-4 py-3 text-left whitespace-normal shadow-waterhouse-soft hover:bg-background/54"
                        >
                            <span className="text-sm font-semibold">{template.name}</span>
                            <span className="text-xs text-muted-foreground">{t(template.descriptionKey)}</span>
                        </Button>
                    ))}
                </div>
            </section>

            <section className={sectionClassName}>
                <SectionHeader icon={Orbit} title={t('name')} hint={t('type')} />
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div className={fieldGroupClassName}>
                        <label htmlFor={`${idPrefix}-name`} className={labelClassName}>
                        {t('name')}
                        </label>
                        <Input
                            className="rounded-[1.25rem]"
                            id={`${idPrefix}-name`}
                            type="text"
                            value={formData.name}
                            onChange={(event) => onFormDataChange({ ...formData, name: event.target.value })}
                            required
                        />
                    </div>

                    <div className={fieldGroupClassName}>
                        <label htmlFor={`${idPrefix}-type`} className={labelClassName}>
                        {t('type')}
                        </label>
                        <Select
                            value={String(formData.type)}
                            onValueChange={(value) => onFormDataChange({ ...formData, type: Number(value) as ChannelType })}
                        >
                            <SelectTrigger id={`${idPrefix}-type`} className="w-full rounded-[1.25rem] border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent className="rounded-[1.25rem]">
                                <SelectItem className="rounded-xl" value={String(ChannelType.OpenAIChat)}>{t('typeOpenAIChat')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.OpenAIResponse)}>{t('typeOpenAIResponse')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.Anthropic)}>{t('typeAnthropic')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.Gemini)}>{t('typeGemini')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.Volcengine)}>{t('typeVolcengine')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.OpenAIEmbedding)}>{t('typeOpenAIEmbedding')}</SelectItem>
                                <SelectItem className="rounded-xl" value={String(ChannelType.Mimo)}>{t('typeMimo')}</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            </section>

            <section className={sectionClassName}>
                <SectionHeader icon={Cable} title={t('baseUrls')} hint={t('baseUrlUrl')} />
                <div className="flex items-center justify-between">
                    <label className={labelClassName}>
                        {t('baseUrls')} {formData.base_urls.length > 0 ? `(${formData.base_urls.length})` : ''}
                    </label>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleAddBaseUrl}
                        className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                    >
                        <Plus className="h-3 w-3 mr-1" />
                        {t('add')}
                    </Button>
                </div>
                <div className="space-y-2">
                    {(formData.base_urls ?? []).map((u, idx) => (
                        <div key={`baseurl-${idx}`} className="waterhouse-pod flex items-center gap-2 rounded-[1.35rem] border border-border/25 bg-background/42 p-2 shadow-waterhouse-soft">
                            <Input
                                id={`${idPrefix}-base-${idx}`}
                                type="url"
                                value={u.url}
                                onChange={(e) => handleUpdateBaseUrl(idx, { url: e.target.value })}
                                placeholder={t('baseUrlUrl')}
                                required={idx === 0}
                                className="flex-1 rounded-[1.15rem]"
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveBaseUrl(idx)}
                                disabled={(formData.base_urls ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive disabled:opacity-40 hover:bg-transparent"
                                title={t('remove')}
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </section>

            <section className={sectionClassName}>
                <SectionHeader icon={KeyRound} title={t('apiKey')} hint={t('remark')} />
                <div className="flex items-center justify-between">
                    <label className={labelClassName}>
                        {t('apiKey')} {formData.keys.length > 0 ? `(${formData.keys.length})` : ''}
                    </label>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleAddKey}
                        className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                    >
                        <Plus className="h-3 w-3 mr-1" />
                        {t('add')}
                    </Button>
                </div>
                <div className="space-y-2">
                    {(formData.keys ?? []).map((k, idx) => (
                        <div key={k.id ?? `new-${idx}`} className="waterhouse-pod grid gap-2 rounded-[1.35rem] border border-border/25 bg-background/42 p-2 shadow-waterhouse-soft md:grid-cols-[minmax(0,1fr)_10rem_auto_auto] md:items-center">
                            <Input
                                type="text"
                                value={k.channel_key}
                                onChange={(e) => handleUpdateKey(idx, { channel_key: e.target.value })}
                                placeholder={t('apiKey')}
                                required={idx === 0}
                                className="rounded-[1.15rem]"
                            />
                            <Input
                                type="text"
                                value={k.remark ?? ''}
                                onChange={(e) => handleUpdateKey(idx, { remark: e.target.value })}
                                placeholder={t('remark')}
                                className="rounded-[1.15rem] md:w-40"
                            />
                            <label className="flex items-center gap-2 rounded-[1.1rem] border border-border/20 bg-background/48 px-3 py-2 text-sm text-card-foreground shadow-waterhouse-soft">
                                <Switch
                                    checked={k.enabled}
                                    onCheckedChange={(checked) => handleUpdateKey(idx, { enabled: checked })}
                                />
                                <span>{t('enabled')}</span>
                            </label>
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveKey(idx)}
                                disabled={(formData.keys ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                title={t('remove')}
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </section>

            <section className={sectionClassName}>
                <SectionHeader icon={Layers3} title={t('model')} hint={t('modelSelected')} />
                <div className="flex items-center justify-between gap-2">
                    <label className={labelClassName}>{t('model')}</label>
                    <div className="flex items-center gap-2">
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={handleTestChannel}
                            disabled={testChannel.isPending || !(formData.base_urls?.some((u) => u.url.trim()) && formData.keys?.some((k) => k.channel_key.trim()))}
                            className="h-6 px-2 text-xs text-muted-foreground/50 hover:text-muted-foreground hover:bg-transparent"
                        >
                            {testChannel.isPending ? (
                                <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                            ) : (
                                <FlaskConical className="h-3 w-3 mr-1" />
                            )}
                            {t('test.button')}
                        </Button>
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={handleRefreshModels}
                            disabled={!formData.base_urls?.[0]?.url || !effectiveKey || fetchModel.isPending}
                            className="h-6 px-2 text-xs text-muted-foreground/50 hover:text-muted-foreground hover:bg-transparent"
                        >
                            <RefreshCw className={`h-3 w-3 mr-1 ${fetchModel.isPending ? 'animate-spin' : ''}`} />
                            {t('modelRefresh')}
                        </Button>
                    </div>
                </div>
                <input type="hidden" value={formData.model} required />

                {testSummary && (
                    <div className="waterhouse-pod space-y-2 rounded-[1.45rem] border border-border/25 bg-background/42 p-3 shadow-waterhouse-soft">
                        <div className="flex items-center justify-between gap-2">
                            <div className="flex items-center gap-2 text-sm font-medium text-card-foreground">
                                {testSummary.passed ? (
                                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                                ) : (
                                    <AlertTriangle className="h-4 w-4 text-orange-500" />
                                )}
                                <span>{testSummary.passed ? t('test.success') : t('test.partialSuccess')}</span>
                            </div>
                            <Badge variant="secondary">{testSummary.results.length} {t('test.results')}</Badge>
                        </div>
                        <div className="space-y-2 max-h-48 overflow-y-auto">
                            {testSummary.results.map((result, idx) => (
                                <div key={`${result.base_url}-${result.key_masked}-${idx}`} className="rounded-[1.1rem] border border-border/30 bg-background/64 p-2.5 text-xs space-y-1">
                                    <div className="flex items-center justify-between gap-2">
                                        <span className="font-mono truncate">{result.base_url}</span>
                                        <div className="flex items-center gap-1 shrink-0">
                                            <Badge variant="secondary">{result.key_masked || '-'}</Badge>
                                            <Badge variant="secondary">{result.status_code}</Badge>
                                        </div>
                                    </div>
                                    <div className="flex items-center justify-between gap-2 text-muted-foreground">
                                        <span>{result.key_remark || t('test.noRemark')}</span>
                                        <span>{result.latency_ms}ms · {result.passed ? t('test.pass') : t('test.fail')}</span>
                                    </div>
                                    {result.message && <p className="break-all text-muted-foreground">{result.message}</p>}
                                </div>
                            ))}
                        </div>
                        <p className="text-xs text-muted-foreground">{t('test.hint')}</p>
                    </div>
                )}

                <div className="relative">
                    <Input
                        ref={inputRef}
                        id={`${idPrefix}-model-custom`}
                        type="text"
                        value={inputValue}
                        onChange={(e) => setInputValue(e.target.value)}
                        onKeyDown={handleInputKeyDown}
                        placeholder={t('modelCustomPlaceholder')}
                        className="rounded-[1.25rem] pr-10"
                    />
                    {inputValue.trim() && !customModels.includes(inputValue.trim()) && !autoModels.includes(inputValue.trim()) && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => handleAddModel(inputValue)}
                            className="absolute rounded-lg right-1 top-1/2 -translate-y-1/2 h-7 w-7 p-0 text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
                            title={t('modelAdd')}
                        >
                            <Plus className="size-4" />
                        </Button>
                    )}
                </div>

                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <label className="text-xs font-medium text-card-foreground">
                            {t('modelSelected')} {(autoModels.length + customModels.length) > 0 && `(${autoModels.length + customModels.length})`}
                        </label>
                        {(autoModels.length + customModels.length) > 0 && (
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    updateModels([], []);
                                }}
                                className="h-6 px-2 text-xs text-muted-foreground/50 hover:text-muted-foreground hover:bg-transparent"
                            >
                                {t('modelClearAll')}
                            </Button>
                        )}
                    </div>
                    <div className="waterhouse-pod max-h-40 min-h-12 overflow-y-auto rounded-[1.45rem] border border-border/25 bg-background/42 p-2.5 shadow-waterhouse-soft">
                        {(autoModels.length + customModels.length) > 0 ? (
                            <div className="flex flex-wrap gap-1.5">
                                {autoModels.map((model) => (
                                    <Badge key={model} variant="secondary" className="bg-muted hover:bg-muted/80">
                                        {model}
                                        <button
                                            type="button"
                                            onClick={() => handleRemoveAutoModel(model)}
                                            className="ml-1 rounded-sm opacity-70 hover:opacity-100 focus:outline-none focus:ring-1 focus:ring-ring"
                                        >
                                            <X className="h-3 w-3" />
                                        </button>
                                    </Badge>
                                ))}
                                {customModels.map((model) => (
                                    <Badge key={model} className="bg-primary hover:bg-primary/90">
                                        {model}
                                        <button
                                            type="button"
                                            onClick={() => handleRemoveCustomModel(model)}
                                            className="ml-1 rounded-sm opacity-70 hover:opacity-100 focus:outline-none focus:ring-1 focus:ring-ring"
                                        >
                                            <X className="h-3 w-3" />
                                        </button>
                                    </Badge>
                                ))}
                            </div>
                        ) : (
                            <div className="flex items-center justify-center h-8 text-xs text-muted-foreground">
                                {t('modelNoSelected')}
                            </div>
                        )}
                    </div>
                </div>
            </section>

            <Accordion type="single" collapsible className="w-full">
                <AccordionItem value="advanced" className="border-none">
                    <AccordionTrigger className="waterhouse-pod rounded-[1.8rem] border border-border/30 bg-background/34 px-4 py-4 text-sm font-medium text-card-foreground shadow-waterhouse-soft transition-colors hover:bg-background/44 hover:no-underline">
                        <span className="flex items-center gap-2">
                            <span className="h-2 w-2 rounded-full bg-primary/70" />
                            {t('advanced')}
                        </span>
                    </AccordionTrigger>
                    <AccordionContent className="pt-4">
                        <div className={sectionClassName}>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className={fieldGroupClassName}>
                                <label htmlFor={`${idPrefix}-auto-group`} className={labelClassName}>
                                    {t('autoGroup')}
                                </label>
                                <Select
                                    value={String(formData.auto_group)}
                                    onValueChange={(value) => onFormDataChange({ ...formData, auto_group: Number(value) as AutoGroupType })}
                                >
                                    <SelectTrigger id={`${idPrefix}-auto-group`} className="w-full rounded-[1.25rem] border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent className="rounded-[1.25rem]">
                                        <SelectItem className="rounded-xl" value={String(AutoGroupType.None)}>{t('autoGroupNone')}</SelectItem>
                                        <SelectItem className="rounded-xl" value={String(AutoGroupType.Fuzzy)}>{t('autoGroupFuzzy')}</SelectItem>
                                        <SelectItem className="rounded-xl" value={String(AutoGroupType.Exact)}>{t('autoGroupExact')}</SelectItem>
                                        <SelectItem className="rounded-xl" value={String(AutoGroupType.Regex)}>{t('autoGroupRegex')}</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className={fieldGroupClassName}>
                                <label htmlFor={`${idPrefix}-channel-proxy`} className={labelClassName}>
                                    {t('channelProxy')}
                                </label>
                                <Input
                                    id={`${idPrefix}-channel-proxy`}
                                    type="text"
                                    value={formData.channel_proxy}
                                    onChange={(e) => onFormDataChange({ ...formData, channel_proxy: e.target.value })}
                                    placeholder={t('channelProxyPlaceholder')}
                                    className="rounded-[1.25rem]"
                                />
                            </div>
                        </div>

                        <div className={fieldGroupClassName}>
                            <div className="flex items-center justify-between">
                                <label className={labelClassName}>
                                    {t('customHeader')} {formData.custom_header.length > 0 ? `(${formData.custom_header.length})` : ''}
                                </label>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={handleAddHeader}
                                    className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                                >
                                    <Plus className="h-3 w-3 mr-1" />
                                    {t('customHeaderAdd')}
                                </Button>
                            </div>
                            <div className="space-y-2">
                                {(formData.custom_header ?? []).map((h, idx) => (
                                    <div key={`hdr-${idx}`} className="waterhouse-pod grid gap-2 rounded-[1.35rem] border border-border/25 bg-background/42 p-2 shadow-waterhouse-soft md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] md:items-center">
                                        <Input
                                            type="text"
                                            value={h.header_key}
                                            onChange={(e) => handleUpdateHeader(idx, { header_key: e.target.value })}
                                            placeholder={t('customHeaderKey')}
                                            className="rounded-[1.15rem]"
                                        />
                                        <Input
                                            type="text"
                                            value={h.header_value}
                                            onChange={(e) => handleUpdateHeader(idx, { header_value: e.target.value })}
                                            placeholder={t('customHeaderValue')}
                                            className="rounded-[1.15rem]"
                                        />
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => handleRemoveHeader(idx)}
                                            disabled={(formData.custom_header ?? []).length <= 1}
                                            className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                            title={t('remove')}
                                        >
                                            <X className="h-4 w-4" />
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        </div>

                        <div className={fieldGroupClassName}>
                            <label htmlFor={`${idPrefix}-match-regex`} className={labelClassName}>
                                {t('matchRegex')}
                            </label>
                            <Input
                                id={`${idPrefix}-match-regex`}
                                type="text"
                                value={formData.match_regex}
                                onChange={(e) => onFormDataChange({ ...formData, match_regex: e.target.value })}
                                placeholder={t('matchRegexPlaceholder')}
                                className="rounded-[1.25rem]"
                            />
                        </div>

                        <div className={fieldGroupClassName}>
                            <label htmlFor={`${idPrefix}-param-override`} className={labelClassName}>
                                {t('paramOverride')}
                            </label>
                            <textarea
                                id={`${idPrefix}-param-override`}
                                value={formData.param_override}
                                onChange={(e) => onFormDataChange({ ...formData, param_override: e.target.value })}
                                placeholder={t('paramOverridePlaceholder')}
                                className="waterhouse-liquid-field min-h-28 w-full rounded-[1.35rem] border border-border/35 bg-background/62 px-3 py-2 text-sm text-foreground shadow-inner focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            />
                        </div>

                        <div className="space-y-4 rounded-[1.55rem] border border-border/30 bg-background/40 p-4 shadow-waterhouse-soft">
                            <div className="flex flex-wrap items-center justify-between gap-3">
                                <div>
                                    <p className="text-sm font-medium text-card-foreground">{t('requestRewrite')}</p>
                                    <p className="text-xs text-muted-foreground">{t('requestRewriteHint')}</p>
                                </div>
                                <label className="flex items-center gap-2 cursor-pointer">
                                    <Switch
                                        checked={requestRewriteSupported && formData.request_rewrite.enabled}
                                        onCheckedChange={(checked) => onFormDataChange({
                                            ...formData,
                                            request_rewrite: {
                                                ...formData.request_rewrite,
                                                enabled: checked,
                                            },
                                        })}
                                        disabled={!requestRewriteSupported}
                                    />
                                    <span className="text-sm text-card-foreground">{t('requestRewriteEnabled')}</span>
                                </label>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                <div className={fieldGroupClassName}>
                                    <label htmlFor={`${idPrefix}-request-rewrite-profile`} className={labelClassName}>
                                        {t('requestRewriteProfile')}
                                    </label>
                                    <Select
                                        value={formData.request_rewrite.profile ?? RequestRewriteProfile.OpenAIChatCompat}
                                        onValueChange={(value) => onFormDataChange({
                                            ...formData,
                                            request_rewrite: {
                                                ...formData.request_rewrite,
                                                profile: value as RequestRewriteProfile,
                                            },
                                        })}
                                        disabled={!requestRewriteSupported || !formData.request_rewrite.enabled}
                                    >
                                        <SelectTrigger id={`${idPrefix}-request-rewrite-profile`} className="w-full rounded-[1.25rem] border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className="rounded-[1.25rem]">
                                            <SelectItem className="rounded-xl" value={RequestRewriteProfile.OpenAIChatCompat}>{t('requestRewriteProfileOpenAIChatCompat')}</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>

                                <div className={fieldGroupClassName}>
                                    <label htmlFor={`${idPrefix}-request-rewrite-tool-role`} className={labelClassName}>
                                        {t('requestRewriteToolRoleStrategy')}
                                    </label>
                                    <Select
                                        value={formData.request_rewrite.tool_role_strategy ?? ToolRoleStrategy.Keep}
                                        onValueChange={(value) => onFormDataChange({
                                            ...formData,
                                            request_rewrite: {
                                                ...formData.request_rewrite,
                                                tool_role_strategy: value as ToolRoleStrategy,
                                            },
                                        })}
                                        disabled={!requestRewriteSupported || !formData.request_rewrite.enabled}
                                    >
                                        <SelectTrigger id={`${idPrefix}-request-rewrite-tool-role`} className="w-full rounded-[1.25rem] border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className="rounded-[1.25rem]">
                                            <SelectItem className="rounded-xl" value={ToolRoleStrategy.Keep}>{t('requestRewriteStrategyKeep')}</SelectItem>
                                            <SelectItem className="rounded-xl" value={ToolRoleStrategy.StringifyToUser}>{t('requestRewriteStrategyStringifyToUser')}</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>

                                <div className={fieldGroupClassName}>
                                    <label htmlFor={`${idPrefix}-request-rewrite-system`} className={labelClassName}>
                                        {t('requestRewriteSystemMessageStrategy')}
                                    </label>
                                    <Select
                                        value={formData.request_rewrite.system_message_strategy ?? SystemMessageStrategy.Keep}
                                        onValueChange={(value) => onFormDataChange({
                                            ...formData,
                                            request_rewrite: {
                                                ...formData.request_rewrite,
                                                system_message_strategy: value as SystemMessageStrategy,
                                            },
                                        })}
                                        disabled={!requestRewriteSupported || !formData.request_rewrite.enabled}
                                    >
                                        <SelectTrigger id={`${idPrefix}-request-rewrite-system`} className="w-full rounded-[1.25rem] border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent className="rounded-[1.25rem]">
                                            <SelectItem className="rounded-xl" value={SystemMessageStrategy.Keep}>{t('requestRewriteStrategyKeep')}</SelectItem>
                                            <SelectItem className="rounded-xl" value={SystemMessageStrategy.Merge}>{t('requestRewriteStrategyMerge')}</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>
                            </div>
                        </div>
                        </div>
                    </AccordionContent>
                </AccordionItem>
            </Accordion>
            </div>
            </div>

            <section className={`${sectionClassName} mt-4 flex shrink-0 flex-col gap-4 md:flex-row md:items-center md:justify-between`}>
                <SectionHeader icon={SlidersHorizontal} title={t('advanced')} />
                <label className="flex items-center gap-2 cursor-pointer">
                    <Switch
                        checked={formData.enabled}
                        onCheckedChange={(checked) => onFormDataChange({ ...formData, enabled: checked })}
                    />
                    <span className="text-sm font-medium text-card-foreground">{t('enabled')}</span>
                </label>
                <div className="flex items-center gap-6">
                    <label className="flex items-center gap-2 cursor-pointer">
                        <Switch
                            checked={formData.proxy}
                            onCheckedChange={(checked) => onFormDataChange({ ...formData, proxy: checked })}
                        />
                        <span className="text-sm text-card-foreground">{t('proxy')}</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                        <Switch
                            checked={formData.auto_sync}
                            onCheckedChange={(checked) => onFormDataChange({ ...formData, auto_sync: checked })}
                        />
                        <span className="text-sm text-card-foreground">{t('autoSync')}</span>
                    </label>
                </div>
            </section>

            <div className={`shrink-0 flex flex-col gap-3 pt-4 ${onCancel ? 'sm:flex-row' : ''}`}>
                {onCancel && cancelText && (
                    <Button
                        type="button"
                        variant="secondary"
                        onClick={onCancel}
                        className="h-12 w-full rounded-[1.6rem] sm:flex-1"
                    >
                        {cancelText}
                    </Button>
                )}
                <Button
                    type="submit"
                    disabled={isPending}
                    className="h-12 w-full rounded-[1.6rem] sm:flex-1"
                >
                    {isPending ? pendingText : submitText}
                </Button>
            </div>
        </form>
    );
}
