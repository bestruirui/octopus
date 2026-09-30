import { useId, useMemo, useState } from 'react';
import { Check, ChevronDownIcon, Plus, Sparkles, X } from 'lucide-react';
import { useTranslations } from 'use-intl';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { Protocol } from '@/api/channel';
import { SearchSuggestionInput } from '@/components/common/SearchSuggestionInput';
import { Accordion, AccordionContent, AccordionItem } from '@/components/ui/accordion';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';
import type { SelectedMember } from './ItemList';
import { buildModelSuggestions, filterModelMembers, getAutoAddMembers, memberKey, normalizeKey, rankModelSuggestions, type ModelSearch } from './utils';

// PROTOCOL_TAGS 是凭据行上的协议标识。
// 此处写全称: 凭据行只有名称一列, 横向有余量; 渠道表单的授权矩阵是三列复选框, 列宽紧张才用缩写。
const PROTOCOL_TAGS = [
    { bit: Protocol.OpenAIChatCompletion, label: 'Chat' },
    { bit: Protocol.OpenAIResponse, label: 'Response' },
    { bit: Protocol.AnthropicMessage, label: 'Message' },
];

export function ModelPickerSection({
    grantMembers,
    selectedMembers,
    groupName,
    onAddMembers,
    loading,
    loadError,
    retrying,
    onRetry,
    disabled,
}: {
    grantMembers: SelectedMember[];
    selectedMembers: SelectedMember[];
    groupName: string;
    onAddMembers: (members: SelectedMember[]) => void;
    loading: boolean;
    loadError: boolean;
    retrying: boolean;
    onRetry: () => void;
    disabled: boolean;
}) {
    const t = useTranslations('group');
    const previewId = useId();
    const [search, setSearch] = useState<ModelSearch>({ keyword: '', exactModel: null });
    const [onlyEnabled, setOnlyEnabled] = useState(true);
    const [expandedChannels, setExpandedChannels] = useState<string[] | null>(null);

    const selectedKeys = useMemo(() => new Set(selectedMembers.map(memberKey)), [selectedMembers]);
    const hasSearch = Boolean(normalizeKey(search.keyword)) || search.exactModel !== null;
    const filteredMembers = useMemo(() => filterModelMembers(grantMembers, search, onlyEnabled), [grantMembers, search, onlyEnabled]);
    const additions = useMemo(() => getAutoAddMembers(grantMembers, selectedMembers, search, onlyEnabled), [grantMembers, selectedMembers, search, onlyEnabled]);
    const suggestionIndex = useMemo(() => buildModelSuggestions(grantMembers, selectedMembers, onlyEnabled), [grantMembers, selectedMembers, onlyEnabled]);
    const suggestions = useMemo(() => rankModelSuggestions(suggestionIndex, search.keyword), [suggestionIndex, search.keyword]);
    const addDisabled = disabled || loading || loadError || additions.length === 0;

    const updateSearch = (next: ModelSearch) => {
        setSearch(next);
        setExpandedChannels(null);
    };

    const emptyMessage = loading ? t('form.modelSearch.loading')
        : loadError ? t('form.modelSearch.loadError')
            : grantMembers.length === 0 ? t('form.modelSearch.empty')
                : onlyEnabled && suggestionIndex.length === 0 ? t('form.modelSearch.noEnabled')
                    : t('form.modelSearch.noMatches');

    const suggestionHeading = hasSearch ? t('form.modelSearch.results')
        : suggestions.some((suggestion) => suggestion.inGroup) ? t('form.modelSearch.groupModelsFirst')
            : t(suggestions.some((suggestion) => suggestion.channelCount > 1) ? 'form.modelSearch.commonModels' : 'form.modelSearch.candidates');

    const disabledAdditions = additions.filter((member) => !member.enabled).length;
    const preview = !hasSearch ? t('form.modelSearch.emptySearch')
        : filteredMembers.length === 0 ? emptyMessage
            : additions.length === 0 ? t('form.modelSearch.allAdded')
                : t('form.modelSearch.preview', {
                    channels: new Set(additions.map((member) => member.channel_id)).size,
                    models: new Set(additions.map((member) => member.name)).size,
                    grants: additions.length,
                    skipped: new Set(filteredMembers.map(memberKey)).size - additions.length,
                });

    // 候选按渠道 -> 模型 -> 凭据三级组织: 一个模型可有多份凭据, 各自是独立授权, 需再展开一级才能分别选取。
    // 三级顺序沿用后端给出的候选顺序: Map 保留插入顺序, 后端已按渠道, 模型, 凭据排好, 此处无需再排。
    const channels = useMemo(() => {
        const byChannel = new Map<number, {
            id: number;
            name: string;
            models: Map<string, SelectedMember[]>;
        }>();
        filteredMembers.forEach((mc) => {
            let channel = byChannel.get(mc.channel_id);
            if (!channel) {
                channel = { id: mc.channel_id, name: mc.channel_name, models: new Map() };
                byChannel.set(mc.channel_id, channel);
            }
            const grants = channel.models.get(mc.name);
            if (grants) grants.push(mc);
            else channel.models.set(mc.name, [mc]);
        });

        return Array.from(byChannel.values()).map((channel) => ({
            id: channel.id,
            name: channel.name,
            models: Array.from(channel.models, ([name, grants]) => ({ name, grants })),
        }));
    }, [filteredMembers]);

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0">
            <div className="space-y-2 px-3 py-2 border-b border-border/30 bg-muted/50">
                <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="text-sm font-medium text-foreground">{t('form.addItem')}</span>
                    <button
                        type="button"
                        onClick={() => { if (!addDisabled) onAddMembers(additions); }}
                        className={cn(
                            'shrink-0 flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium transition-colors',
                            addDisabled ? 'text-muted-foreground/50 cursor-not-allowed' : 'hover:bg-muted text-muted-foreground hover:text-foreground',
                        )}
                        disabled={addDisabled}
                        aria-describedby={previewId}
                    >
                        <Sparkles aria-hidden="true" className="size-3.5" />
                        <span>{t('form.modelSearch.addMatches', { count: additions.length })}</span>
                    </button>
                </div>
                <SearchSuggestionInput
                    value={search.keyword}
                    onValueChange={(keyword) => updateSearch({ keyword, exactModel: null })}
                    onSelect={(name) => updateSearch({ keyword: name, exactModel: name })}
                    suggestions={loading || loadError ? [] : suggestions.map((suggestion) => ({
                        value: suggestion.name,
                        description: <>
                            {suggestion.inGroup && <><span className="font-medium text-primary">{t('form.modelSearch.inGroup')}</span>{' · '}</>}
                            {t('form.modelSearch.channels', { count: suggestion.channelCount })}
                            {' · '}
                            {suggestion.pendingCount ? t('form.modelSearch.pending', { count: suggestion.pendingCount }) : t('form.modelSearch.allAdded')}
                            {suggestion.newChannelCount > 0 && <> · {t('form.modelSearch.newChannels', { count: suggestion.newChannelCount })}</>}
                            {suggestion.disabledCount > 0 && <span className="block text-amber-600 dark:text-amber-400">{t('form.modelSearch.disabledGrants', { count: suggestion.disabledCount })}</span>}
                        </>,
                    }))}
                    label={t('form.modelSearch.placeholder')}
                    placeholder={t('form.modelSearch.placeholder')}
                    heading={suggestionHeading}
                    emptyMessage={emptyMessage}
                    hint={t('form.modelSearch.selectionHint')}
                    clearLabel={t('form.modelSearch.clear')}
                    disabled={disabled}
                    describedBy={previewId}
                />
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-xs text-muted-foreground">
                    <label className="flex items-center gap-1.5" title={t('form.modelSearch.enabledHint')}>
                        <Checkbox
                            checked={onlyEnabled}
                            disabled={disabled}
                            onCheckedChange={(checked) => {
                                setOnlyEnabled(checked === true);
                                setExpandedChannels(null);
                            }}
                        />
                        {t('form.modelSearch.onlyEnabled')}
                    </label>
                    <button
                        type="button"
                        disabled={disabled || !groupName.trim()}
                        onClick={() => updateSearch({ keyword: groupName.trim(), exactModel: null })}
                        className="hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {t('form.modelSearch.useGroupName')}
                    </button>
                    {search.exactModel !== null && (
                        <button
                            type="button"
                            disabled={disabled}
                            aria-label={t('form.modelSearch.clearExact')}
                            title={t('form.modelSearch.clearExact')}
                            onClick={() => updateSearch({ ...search, exactModel: null })}
                            className="flex items-center gap-1 rounded border border-primary/30 px-1.5 py-0.5 text-primary"
                        >
                            {t('form.modelSearch.exact')}
                            <X aria-hidden="true" className="size-3" />
                        </button>
                    )}
                </div>
                <p id={previewId} role="status" className="text-xs leading-relaxed text-muted-foreground">
                    {preview}
                    {disabledAdditions > 0 && <span className="block text-amber-600 dark:text-amber-400">{t('form.modelSearch.disabledGrants', { count: disabledAdditions })}</span>}
                </p>
                {loadError && (
                    <div role="alert" className="flex items-center gap-2 text-xs text-destructive">
                        <span>{t('form.modelSearch.loadError')}</span>
                        <button type="button" onClick={onRetry} disabled={retrying || disabled} className="shrink-0 underline disabled:opacity-50">{t('form.modelSearch.retry')}</button>
                    </div>
                )}
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto p-2">
                {channels.length === 0 && <p role="status" className="px-2 py-4 text-center text-xs text-muted-foreground">{emptyMessage}</p>}
                <Accordion
                    type="multiple"
                    value={expandedChannels ?? (hasSearch ? channels.map((channel) => `channel-${channel.id}`) : [])}
                    onValueChange={setExpandedChannels}
                    className="w-full space-y-2"
                >
                    {channels.map((channel) => {
                        // 计数按授权算而非按模型: 展开后每份凭据都是一个可选项。
                        const grants = channel.models.flatMap((model) => model.grants);
                        const total = grants.length;
                        const selectedCount = grants.reduce(
                            (acc, m) => acc + (selectedKeys.has(memberKey(m)) ? 1 : 0),
                            0
                        );
                        const available = total - selectedCount;

                        return (
                            <AccordionItem key={channel.id} value={`channel-${channel.id}`}>
                                <AccordionPrimitive.Header className="rounded-lg bg-muted sticky top-0 z-10 flex px-2 overflow-hidden">
                                    <AccordionPrimitive.Trigger className="flex flex-1 min-w-0 items-center gap-4 py-4 text-left text-sm transition-all outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 [&[data-state=open]>svg]:rotate-180">
                                        <span className="truncate">{channel.name}</span>
                                        <span className="text-xs text-muted-foreground shrink-0">
                                            {available}/{total}
                                        </span>
                                        <ChevronDownIcon className="text-muted-foreground pointer-events-none size-4 shrink-0 transition-transform duration-200" />
                                    </AccordionPrimitive.Trigger>
                                </AccordionPrimitive.Header>
                                <AccordionContent className="px-2 pt-2">
                                    <div className="flex flex-col gap-1.5">
                                        {channel.models.map((model) => {
                                            const { Icon, className: iconClassName } = getModelIcon(model.name);
                                            const modelSelected = model.grants.reduce(
                                                (acc, m) => acc + (selectedKeys.has(memberKey(m)) ? 1 : 0),
                                                0
                                            );
                                            return (
                                                <div key={model.name} className="rounded-lg border border-border/50 bg-background">
                                                    {/* 模型行只作分组标题, 不可点选: 可选的是它下面的凭据, 一份凭据一条授权。 */}
                                                    <div className="flex items-center justify-between gap-2 px-2.5 py-2">
                                                        <span className="flex items-center gap-2 min-w-0">
                                                            <Icon aria-hidden="true" className={iconClassName} width={16} height={16} />
                                                            <span className="text-sm font-medium truncate">{model.name}</span>
                                                        </span>
                                                        <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
                                                            {model.grants.length - modelSelected}/{model.grants.length}
                                                        </span>
                                                    </div>

                                                    <div className="flex flex-col border-t border-border/50">
                                                        {model.grants.map((m) => {
                                                            const isSelected = selectedKeys.has(memberKey(m));
                                                            return (
                                                                <button
                                                                    key={memberKey(m)}
                                                                    type="button"
                                                                    onClick={() => !isSelected && onAddMembers([m])}
                                                                    disabled={isSelected || disabled || loading || loadError}
                                                                    className={cn(
                                                                        'flex w-full items-center justify-between gap-2 px-2.5 py-1.5 pl-8 text-left transition-colors',
                                                                        isSelected ? 'opacity-60 cursor-not-allowed' : 'hover:bg-muted'
                                                                    )}
                                                                >
                                                                    <span className="flex min-w-0 items-center gap-2">
                                                                        <span className="truncate text-xs text-muted-foreground">{m.key_name}</span>
                                                                        {!m.enabled && <span className="shrink-0 text-[10px] text-amber-600 dark:text-amber-400">{t('form.modelSearch.disabled')}</span>}
                                                                        {/* 标出该凭据讲得通的协议: 同一模型的不同凭据可能只支持其中一部分, 选之前就要能看出来。 */}
                                                                        {PROTOCOL_TAGS.map(({ bit, label }) => (m.protocols & bit) !== 0 && (
                                                                            <span
                                                                                key={bit}
                                                                                className="shrink-0 rounded border border-border/60 px-1 text-[10px] leading-4 text-muted-foreground"
                                                                            >
                                                                                {label}
                                                                            </span>
                                                                        ))}
                                                                    </span>
                                                                    <span className="shrink-0 text-muted-foreground">
                                                                        {isSelected ? (
                                                                            <Check className="size-4 text-primary" />
                                                                        ) : (
                                                                            <Plus className="size-4" />
                                                                        )}
                                                                    </span>
                                                                </button>
                                                            );
                                                        })}
                                                    </div>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </AccordionContent>
                            </AccordionItem>
                        );
                    })}
                </Accordion>
            </div>
        </div>
    );
}
