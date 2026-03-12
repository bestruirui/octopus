import { AutoGroupType, ChannelType, type Channel, useFetchModel } from '@/api/endpoints/channel';
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
import { useEffect, useRef, useState } from 'react';
import { RefreshCw, X, Plus, ChevronLeft, ChevronRight, MoreVertical, Download } from 'lucide-react';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog';

export const DEFAULT_BASE_URLS: Partial<Record<ChannelType, string>> = {
    [ChannelType.OpenAIChat]: 'https://api.openai.com/v1',
    [ChannelType.OpenAIResponse]: 'https://api.openai.com/v1',
    [ChannelType.Anthropic]: 'https://api.anthropic.com/v1',
    [ChannelType.Gemini]: 'https://generativelanguage.googleapis.com/v1beta',
    [ChannelType.Volcengine]: 'https://ark.cn-beijing.volces.com/api/v3',
    [ChannelType.OpenAIEmbedding]: 'https://api.openai.com/v1',
};

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
    keys: ChannelKeyFormItem[];
    model: string;
    custom_model: string;
    enabled: boolean;
    proxy: boolean;
    auto_sync: boolean;
    auto_group: AutoGroupType;
    match_regex: string;
    enable_multi_key_retry: boolean;
    retry_count: number;
    key_load_balance_mode: string;
    auto_ban_key_failures: number;
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
    channelId?: number;
}

import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import { BatchImportModal } from './BatchImportModal';
import { Upload } from 'lucide-react';

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
    channelId,
}: ChannelFormProps) {
    const t = useTranslations('channel.form');
    
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
    const [batchImportOpen, setBatchImportOpen] = useState(false);
    
    // keys分页
    const [keyPage, setKeyPage] = useState(1);
    const [keyPageInput, setKeyPageInput] = useState('1');
    const keyPageSize = 10;
    const totalKeyPages = Math.ceil((formData.keys?.length || 0) / keyPageSize);
    const currentKeyPage = Math.min(Math.max(1, keyPage), Math.max(1, totalKeyPages));
    const paginatedKeys = (formData.keys ?? []).slice((currentKeyPage - 1) * keyPageSize, currentKeyPage * keyPageSize);

    // 确认对话框状态
    const [confirmDialog, setConfirmDialog] = useState<{
        open: boolean;
        title: string;
        description: string;
        onConfirm: () => void;
    }>({
        open: false,
        title: '',
        description: '',
        onConfirm: () => {},
    });

    const fetchModel = useFetchModel();

    // 同步 keyPage 和 keyPageInput
    useEffect(() => {
        setKeyPageInput(currentKeyPage.toString());
    }, [currentKeyPage]);

    const handleKeyPageInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setKeyPageInput(e.target.value);
    };

    const handleKeyPageInputBlur = () => {
        const pageNum = parseInt(keyPageInput, 10);
        if (!isNaN(pageNum) && pageNum >= 1 && pageNum <= totalKeyPages) {
            setKeyPage(pageNum);
        } else {
            setKeyPageInput(currentKeyPage.toString());
        }
    };

    const handleKeyPageInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            e.stopPropagation();
            handleKeyPageInputBlur();
        }
    };

    const effectiveKey =
        formData.keys.find((k) => k.enabled && k.channel_key.trim())?.channel_key.trim() || '';

    // 更新模型列表的辅助函数
    const updateModels = (nextAuto: string[], nextCustom: string[]) => {
        const model = nextAuto.join(',');
        const custom_model = nextCustom.join(',');
        if (formData.model === model && formData.custom_model === custom_model) return;
        onFormDataChange({ ...formData, model, custom_model });
    };

    // 处理刷新模型列表
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
                custom_header: formData.custom_header?.filter((h) => h.header_key.trim()) || [],
            },
            {
                onSuccess: (data) => {
                    if (data && data.length > 0) {
                        const nextAuto = Array.from(new Set([...autoModels, ...data].map((m) => m.trim()).filter(Boolean)));
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
        const nextKeys = [...formData.keys, { enabled: true, channel_key: '' }];
        onFormDataChange({
            ...formData,
            keys: nextKeys,
        });
        setKeyPage(Math.ceil(nextKeys.length / keyPageSize));
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

    // 处理导入的密钥
    const handleKeysImported = (newKeys: string[]) => {
        const keysToAdd = newKeys.map(k => ({
            enabled: true,
            channel_key: k,
            remark: ''
        }));
        // 若列表中仅有一个空key，则将其过滤掉
        let currentKeys = formData.keys;
        if (currentKeys.length === 1 && !currentKeys[0].channel_key.trim()) {
            currentKeys = [];
        }
        
        const nextKeys = [...currentKeys, ...keysToAdd];
        onFormDataChange({
            ...formData,
            keys: nextKeys,
        });
        setKeyPage(Math.ceil(nextKeys.length / keyPageSize));
    };

    // 批量操作：启用所有key
    const handleEnableAllKeys = (e: Event) => {
        const nextKeys = formData.keys.map(k => ({ ...k, enabled: true }));
        onFormDataChange({ ...formData, keys: nextKeys });
        toast.success(t('keyBulkEnableSuccess'));
    };

    // 批量操作：禁用所有key
    const handleDisableAllKeys = (e: Event) => {
        const nextKeys = formData.keys.map(k => ({ ...k, enabled: false }));
        onFormDataChange({ ...formData, keys: nextKeys });
        toast.success(t('keyBulkDisableSuccess'));
    };

    // 批量操作：移除所有key
    const handleRemoveAllKeys = (e: Event) => {
        setConfirmDialog({
            open: true,
            title: t('keyBulkRemoveConfirmTitle'),
            description: t('keyBulkRemoveConfirmDesc'),
            onConfirm: () => {
                onFormDataChange({ ...formData, keys: [{ enabled: true, channel_key: '' }] });
                setKeyPage(1);
                toast.success(t('keyBulkRemoveSuccess'));
                setConfirmDialog(prev => ({ ...prev, open: false }));
            },
        });
    };

    // 批量操作：移除已禁用的key
    const handleRemoveDisabledKeys = (e: Event) => {
        const disabledCount = formData.keys.filter(k => !k.enabled).length;
        if (disabledCount === 0) {
            toast.info(t('keyBulkRemoveDisabledNone'));
            return;
        }
        
        setConfirmDialog({
            open: true,
            title: t('keyBulkRemoveDisabledConfirmTitle'),
            description: t('keyBulkRemoveDisabledConfirmDesc', { count: disabledCount }),
            onConfirm: () => {
                const nextKeys = formData.keys.filter(k => k.enabled);
                if (nextKeys.length === 0) {
                    onFormDataChange({ ...formData, keys: [{ enabled: true, channel_key: '' }] });
                } else {
                    onFormDataChange({ ...formData, keys: nextKeys });
                }
                setKeyPage(1);
                toast.success(t('keyBulkRemoveDisabledSuccess'));
                setConfirmDialog(prev => ({ ...prev, open: false }));
            },
        });
    };

    // 导出密钥功能
    const exportKeys = (keys: ChannelKeyFormItem[], filename: string) => {
        const keysText = keys
            .map(k => k.channel_key.trim())
            .filter(Boolean)
            .join('\n');
        
        if (!keysText) {
            toast.warning(t('keyExportEmpty'));
            return;
        }

        const blob = new Blob([keysText], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        toast.success(t('keyExportSuccess'));
    };

    // 导出所有密钥
    const handleExportAllKeys = (e: Event) => {
        exportKeys(formData.keys, `keys-all-${Date.now()}.txt`);
    };

    // 导出已启用的密钥
    const handleExportEnabledKeys = (e: Event) => {
        const enabledKeys = formData.keys.filter(k => k.enabled);
        exportKeys(enabledKeys, `keys-enabled-${Date.now()}.txt`);
    };

    // 导出已禁用的密钥
    const handleExportDisabledKeys = (e: Event) => {
        const disabledKeys = formData.keys.filter(k => !k.enabled);
        exportKeys(disabledKeys, `keys-disabled-${Date.now()}.txt`);
    };

    return (
        <form onSubmit={onSubmit} className="space-y-4 px-1">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                    <label htmlFor={`${idPrefix}-name`} className="text-sm font-medium text-card-foreground">
                        {t('name')}
                    </label>
                    <Input
                        className='rounded-xl'
                        id={`${idPrefix}-name`}
                        type="text"
                        value={formData.name}
                        onChange={(event) => onFormDataChange({ ...formData, name: event.target.value })}
                        required
                    />
                </div>

                <div className="space-y-2">
                    <label htmlFor={`${idPrefix}-type`} className="text-sm font-medium text-card-foreground">
                        {t('type')}
                    </label>
                    <Select
                        value={String(formData.type)}
                        onValueChange={(value) => {
                            const newType = Number(value) as ChannelType;
                            const currentBaseUrl = formData.base_urls?.[0]?.url;
                            const oldDefault = DEFAULT_BASE_URLS[formData.type];

                            // Only auto-fill if the user hasn't customized the URL (empty or matches previous default)
                            let nextBaseUrls = formData.base_urls;
                            if (!currentBaseUrl || currentBaseUrl === oldDefault) {
                                const newDefault = DEFAULT_BASE_URLS[newType] || '';
                                if (nextBaseUrls && nextBaseUrls.length > 0) {
                                    nextBaseUrls = [{ ...nextBaseUrls[0], url: newDefault }, ...nextBaseUrls.slice(1)];
                                } else {
                                    nextBaseUrls = [{ url: newDefault, delay: 0 }];
                                }
                            }

                            onFormDataChange({ ...formData, type: newType, base_urls: nextBaseUrls });
                        }}
                    >
                        <SelectTrigger id={`${idPrefix}-type`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent className='rounded-xl'>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIChat)}>{t('typeOpenAIChat')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIResponse)}>{t('typeOpenAIResponse')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Anthropic)}>{t('typeAnthropic')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Gemini)}>{t('typeGemini')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.Volcengine)}>{t('typeVolcengine')}</SelectItem>
                            <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIEmbedding)}>{t('typeOpenAIEmbedding')}</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">
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
                        <div key={`baseurl-${idx}`} className="flex items-center gap-2">
                            <Input
                                id={`${idPrefix}-base-${idx}`}
                                type="url"
                                value={u.url}
                                onChange={(e) => handleUpdateBaseUrl(idx, { url: e.target.value })}
                                placeholder={t('baseUrlUrl')}
                                required={idx === 0}
                                className="rounded-xl flex-1"
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveBaseUrl(idx)}
                                disabled={(formData.base_urls ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive disabled:opacity-40 hover:bg-transparent"
                                title="Remove"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </div>

            <BatchImportModal 
                open={batchImportOpen} 
                onOpenChange={setBatchImportOpen}
                onKeysImported={handleKeysImported}
            />

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">
                        {t('apiKey')} {formData.keys.length > 0 ? `(${formData.keys.length})` : ''}
                    </label>
                    <div className="flex gap-2">
                        {totalKeyPages > 1 && (
                            <div className="flex items-center gap-1 mr-2">
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6"
                                    onClick={() => setKeyPage(p => Math.max(1, p - 1))}
                                    disabled={currentKeyPage === 1}
                                >
                                    <ChevronLeft className="h-3 w-3" />
                                </Button>
                                <div className="flex items-center gap-1">
                                    <input
                                        type="text"
                                        value={keyPageInput}
                                        onChange={handleKeyPageInputChange}
                                        onBlur={handleKeyPageInputBlur}
                                        onKeyDown={handleKeyPageInputKeyDown}
                                        className="w-8 h-6 text-center text-xs border rounded px-1 bg-background"
                                    />
                                    <span className="text-xs text-muted-foreground">/ {totalKeyPages}</span>
                                </div>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6"
                                    onClick={() => setKeyPage(p => Math.min(totalKeyPages, p + 1))}
                                    disabled={currentKeyPage === totalKeyPages}
                                >
                                    <ChevronRight className="h-3 w-3" />
                                </Button>
                            </div>
                        )}
                        <DropdownMenu modal={false}>
                            <DropdownMenuTrigger asChild>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="h-6 w-6 p-0 text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                                >
                                    <MoreVertical className="h-3 w-3" />
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent 
                                align="end" 
                                className="rounded-xl" 
                                onCloseAutoFocus={(e) => e.preventDefault()}
                                onInteractOutside={(e) => e.stopPropagation()}
                            >
                                <DropdownMenuItem 
                                    onSelect={handleEnableAllKeys}
                                    className="rounded-lg cursor-pointer"
                                >
                                    {t('keyBulkEnable')}
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                    onSelect={handleDisableAllKeys}
                                    className="rounded-lg cursor-pointer"
                                >
                                    {t('keyBulkDisable')}
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem 
                                    onSelect={handleExportAllKeys}
                                    className="rounded-lg cursor-pointer"
                                >
                                    <Download className="h-3 w-3 mr-2" />
                                    {t('keyExportAll')}
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                    onSelect={handleExportEnabledKeys}
                                    className="rounded-lg cursor-pointer"
                                >
                                    <Download className="h-3 w-3 mr-2" />
                                    {t('keyExportEnabled')}
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                    onSelect={handleExportDisabledKeys}
                                    className="rounded-lg cursor-pointer"
                                >
                                    <Download className="h-3 w-3 mr-2" />
                                    {t('keyExportDisabled')}
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem 
                                    onSelect={handleRemoveAllKeys}
                                    className="rounded-lg cursor-pointer text-destructive focus:text-destructive"
                                >
                                    {t('keyBulkRemove')}
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                    onSelect={handleRemoveDisabledKeys}
                                    className="rounded-lg cursor-pointer text-destructive focus:text-destructive"
                                >
                                    {t('keyBulkRemoveDisabled')}
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={() => setBatchImportOpen(true)}
                            className="h-6 px-2 text-xs text-muted-foreground/70 hover:text-muted-foreground hover:bg-transparent"
                        >
                            <Upload className="h-3 w-3 mr-1" />
                            {t('batchImport')}
                        </Button>
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
                </div>
                <div className="space-y-2">
                    {paginatedKeys.map((k, idx) => {
                        const globalIdx = (currentKeyPage - 1) * keyPageSize + idx;
                        return (
                        <div key={k.id ?? `new-${globalIdx}`} className="flex items-center gap-2">
                            <Input
                                type="text"
                                value={k.channel_key}
                                onChange={(e) => handleUpdateKey(globalIdx, { channel_key: e.target.value })}
                                placeholder={t('apiKey')}
                                required={globalIdx === 0}
                                className="rounded-xl flex-1"
                            />
                            <Input
                                type="text"
                                value={k.remark ?? ''}
                                onChange={(e) => handleUpdateKey(globalIdx, { remark: e.target.value })}
                                placeholder={t('remark')}
                                className="rounded-xl w-32"
                            />
                            <Switch
                                checked={k.enabled}
                                onCheckedChange={(checked) => handleUpdateKey(globalIdx, { enabled: checked })}
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => handleRemoveKey(globalIdx)}
                                disabled={(formData.keys ?? []).length <= 1}
                                className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                title="Remove"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    )})}
                </div>
            </div>

            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">{t('model')}</label>
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
                <input type="hidden" value={formData.model} required />

                <div className="relative">
                    <Input
                        ref={inputRef}
                        id={`${idPrefix}-model-custom`}
                        type="text"
                        value={inputValue}
                        onChange={(e) => setInputValue(e.target.value)}
                        onKeyDown={handleInputKeyDown}
                        placeholder={t('modelCustomPlaceholder')}
                        className="pr-10 rounded-xl"
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
                    <div className="rounded-xl border border-border bg-muted/30 p-2.5 max-h-40 min-h-12 overflow-y-auto">
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
            </div>

            <Accordion type="single" collapsible className="w-full border rounded-xl bg-card">
                <AccordionItem value="advanced" className="border-none">
                    <AccordionTrigger className="text-sm font-medium text-card-foreground py-3 px-4 hover:no-underline hover:bg-muted/30 rounded-xl transition-colors">
                        {t('advanced')}
                    </AccordionTrigger>
                    <AccordionContent className="pt-4 px-4 pb-4 space-y-4 border-t">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <label htmlFor={`${idPrefix}-auto-group`} className="text-sm font-medium text-card-foreground">
                                    {t('autoGroup')}
                                </label>
                                <Select
                                    value={String(formData.auto_group)}
                                    onValueChange={(value) => onFormDataChange({ ...formData, auto_group: Number(value) as AutoGroupType })}
                                >
                                    <SelectTrigger id={`${idPrefix}-auto-group`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent className='rounded-xl'>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.None)}>{t('autoGroupNone')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Fuzzy)}>{t('autoGroupFuzzy')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Exact)}>{t('autoGroupExact')}</SelectItem>
                                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Regex)}>{t('autoGroupRegex')}</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>

                            <div className="space-y-2 col-span-1 md:col-span-2 border rounded-xl p-4 bg-muted/20">
                                <div className="flex items-center justify-between">
                                    <div className="space-y-0.5">
                                        <label className="text-sm font-medium text-card-foreground">
                                            {t('enableMultiKeyRetry')}
                                        </label>
                                        <p className="text-xs text-muted-foreground">
                                            {t('enableMultiKeyRetryDesc')}
                                        </p>
                                    </div>
                                    <Switch
                                        checked={formData.enable_multi_key_retry}
                                        onCheckedChange={(checked) => onFormDataChange({ ...formData, enable_multi_key_retry: checked })}
                                    />
                                </div>

                                {formData.enable_multi_key_retry && (
                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-4">
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium text-card-foreground">
                                                {t('retryCount')}
                                            </label>
                                            <Input
                                                type="number"
                                                min={1}
                                                value={formData.retry_count}
                                                onChange={(e) => {
                                                    const val = parseInt(e.target.value);
                                                    if (!isNaN(val)) {
                                                        onFormDataChange({ ...formData, retry_count: val });
                                                    }
                                                }}
                                                className="rounded-xl"
                                            />
                                            <p className="text-xs text-muted-foreground">{t('retryCountDesc')}</p>
                                        </div>
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium text-card-foreground">
                                                {t('keyLoadBalanceMode')}
                                            </label>
                                            <Select
                                                value={formData.key_load_balance_mode}
                                                onValueChange={(value) => onFormDataChange({ ...formData, key_load_balance_mode: value })}
                                            >
                                                <SelectTrigger className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                                                    <SelectValue />
                                                </SelectTrigger>
                                                <SelectContent className="rounded-xl">
                                                    <SelectItem className='rounded-xl' value="round_robin">{t('loadBalanceRoundRobin')}</SelectItem>
                                                    <SelectItem className='rounded-xl' value="random">{t('loadBalanceRandom')}</SelectItem>
                                                </SelectContent>
                                            </Select>
                                            <p className="text-xs text-muted-foreground">{t('keyLoadBalanceModeDesc')}</p>
                                        </div>
                                        <div className="space-y-2">
                                            <label className="text-sm font-medium text-card-foreground">
                                                {t('autoBanKeyFailures')}
                                            </label>
                                            <Input
                                                type="number"
                                                min={0}
                                                value={formData.auto_ban_key_failures}
                                                onChange={(e) => {
                                                    const val = parseInt(e.target.value);
                                                    if (!isNaN(val)) {
                                                        onFormDataChange({ ...formData, auto_ban_key_failures: val });
                                                    }
                                                }}
                                                className="rounded-xl"
                                            />
                                            <p className="text-xs text-muted-foreground">{t('autoBanKeyFailuresDesc')}</p>
                                        </div>
                                    </div>
                                )}
                            </div>

                            <div className="space-y-2">
                                <label htmlFor={`${idPrefix}-channel-proxy`} className="text-sm font-medium text-card-foreground">
                                    {t('channelProxy')}
                                </label>
                                <Input
                                    id={`${idPrefix}-channel-proxy`}
                                    type="text"
                                    value={formData.channel_proxy}
                                    onChange={(e) => onFormDataChange({ ...formData, channel_proxy: e.target.value })}
                                    placeholder={t('channelProxyPlaceholder')}
                                    className="rounded-xl"
                                />
                            </div>
                        </div>

                        <div className="space-y-2">
                            <div className="flex items-center justify-between">
                                <label className="text-sm font-medium text-card-foreground">
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
                                    <div key={`hdr-${idx}`} className="flex items-center gap-2">
                                        <Input
                                            type="text"
                                            value={h.header_key}
                                            onChange={(e) => handleUpdateHeader(idx, { header_key: e.target.value })}
                                            placeholder={t('customHeaderKey')}
                                            className="rounded-xl flex-1"
                                        />
                                        <Input
                                            type="text"
                                            value={h.header_value}
                                            onChange={(e) => handleUpdateHeader(idx, { header_value: e.target.value })}
                                            placeholder={t('customHeaderValue')}
                                            className="rounded-xl flex-1"
                                        />
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => handleRemoveHeader(idx)}
                                            disabled={(formData.custom_header ?? []).length <= 1}
                                            className="h-8 w-8 p-0 rounded-xl text-muted-foreground hover:text-destructive hover:bg-transparent disabled:opacity-40"
                                            title="Remove"
                                        >
                                            <X className="h-4 w-4" />
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        </div>

                        <div className="space-y-2">
                            <label htmlFor={`${idPrefix}-match-regex`} className="text-sm font-medium text-card-foreground">
                                {t('matchRegex')}
                            </label>
                            <Input
                                id={`${idPrefix}-match-regex`}
                                type="text"
                                value={formData.match_regex}
                                onChange={(e) => onFormDataChange({ ...formData, match_regex: e.target.value })}
                                placeholder={t('matchRegexPlaceholder')}
                                className="rounded-xl"
                            />
                        </div>

                        <div className="space-y-2">
                            <label htmlFor={`${idPrefix}-param-override`} className="text-sm font-medium text-card-foreground">
                                {t('paramOverride')}
                            </label>
                            <textarea
                                id={`${idPrefix}-param-override`}
                                value={formData.param_override}
                                onChange={(e) => onFormDataChange({ ...formData, param_override: e.target.value })}
                                placeholder={t('paramOverridePlaceholder')}
                                className="min-h-28 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            />
                        </div>
                    </AccordionContent>
                </AccordionItem>
            </Accordion>

            <div className="flex flex-wrap items-center justify-between gap-4 p-4 rounded-xl bg-muted/20 border border-border/50">
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
            </div>

            <div className={`flex flex-col gap-3 pt-2 ${onCancel ? 'sm:flex-row' : ''}`}>
                {onCancel && cancelText && (
                    <Button
                        type="button"
                        variant="secondary"
                        onClick={onCancel}
                        className="w-full sm:flex-1 rounded-2xl h-12"
                    >
                        {cancelText}
                    </Button>
                )}
                <Button
                    type="submit"
                    disabled={isPending}
                    className="w-full sm:flex-1 rounded-2xl h-12"
                >
                    {isPending ? pendingText : submitText}
                </Button>
            </div>

            <AlertDialog open={confirmDialog.open} onOpenChange={(open) => setConfirmDialog(prev => ({ ...prev, open }))}>
                <AlertDialogContent className="rounded-xl">
                    <AlertDialogHeader>
                        <AlertDialogTitle>{confirmDialog.title}</AlertDialogTitle>
                        <AlertDialogDescription>{confirmDialog.description}</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel className="rounded-xl">{t('cancel')}</AlertDialogCancel>
                        <AlertDialogAction 
                            className="rounded-xl bg-destructive text-destructive-foreground hover:bg-destructive/90"
                            onClick={confirmDialog.onConfirm}
                        >
                            {t('confirm')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </form>
    );
}
