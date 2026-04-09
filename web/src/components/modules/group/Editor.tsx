'use client';

import { useCallback, useMemo, useState, type FormEvent } from 'react';
import { Check, ChevronDownIcon, HelpCircle, Plus, Search, Sparkles, Trash2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { SettingKey, useSettingList } from '@/api/endpoints/setting';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Accordion, AccordionContent, AccordionItem } from '@/components/ui/accordion';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';
import type { GroupMode } from '@/api/endpoints/group';
import type { SelectedMember } from './ItemList';
import { MemberList } from './ItemList';
import { matchesGroupName, memberKey, normalizeKey, MODE_LABELS } from './utils';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

export type GroupEditorValues = {
    name: string;
    match_regex: string;
    mode: GroupMode;
    first_token_time_out: number;
    session_keep_time: number;
    members: SelectedMember[];
};

type SelectedMemberFilter = 'all' | 'duplicates';
type SelectedMemberSort = 'manual' | 'model-asc' | 'channel-asc' | 'weight-desc';

function ModelPickerSection({
    modelChannels,
    selectedMembers,
    onAdd,
    onAutoAdd,
    autoAddDisabled,
}: {
    modelChannels: LLMChannel[];
    selectedMembers: SelectedMember[];
    onAdd: (channel: LLMChannel) => void;
    onAutoAdd: () => void;
    autoAddDisabled: boolean;
}) {
    const t = useTranslations('group');
    const [searchKeyword, setSearchKeyword] = useState('');

    const selectedKeys = useMemo(() => new Set(selectedMembers.map(memberKey)), [selectedMembers]);
    const normalizedSearch = searchKeyword.trim().toLowerCase();

    const channels = useMemo(() => {
        const byId = new Map<number, { id: number; name: string; models: LLMChannel[] }>();
        modelChannels.forEach((mc) => {
            const existing = byId.get(mc.channel_id);
            if (existing) existing.models.push(mc);
            else byId.set(mc.channel_id, { id: mc.channel_id, name: mc.channel_name, models: [mc] });
        });

        return Array.from(byId.values())
            .map((c) => ({ ...c, models: [...c.models].sort((a, b) => a.name.localeCompare(b.name)) }))
            .sort((a, b) => a.id - b.id);
    }, [modelChannels]);

    const filteredChannels = useMemo(() => {
        if (!normalizedSearch) return channels;
        return channels.reduce<typeof channels>((acc, channel) => {
            if (channel.name.toLowerCase().includes(normalizedSearch)) {
                acc.push(channel);
                return acc;
            }

            const models = channel.models.filter((model) => model.name.toLowerCase().includes(normalizedSearch));
            if (models.length > 0) acc.push({ ...channel, models });
            return acc;
        }, []);
    }, [channels, normalizedSearch]);

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0">
            <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 px-3 py-2 border-b border-border/30 bg-muted/50">
                <span className="min-w-0 justify-self-start text-sm font-medium text-foreground">
                    {t('form.addItem')}
                </span>

                <div className="relative justify-self-center w-30">
                    <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        value={searchKeyword}
                        onChange={(event) => setSearchKeyword(event.target.value)}
                        className="h-6 rounded-lg border-border/60 bg-background/70 pl-7 pr-2 text-xs shadow-none focus-visible:border-border/60 focus-visible:ring-0"
                        aria-label="search"
                    />
                </div>

                <button
                    type="button"
                    onClick={onAutoAdd}
                    className={cn(
                        'justify-self-end shrink-0 flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
                        autoAddDisabled
                            ? 'text-muted-foreground/50 cursor-not-allowed'
                            : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                    )}
                    disabled={autoAddDisabled}
                    title={t('form.autoAdd')}
                >
                    <Sparkles className="size-3.5" />
                    <span>{t('form.autoAdd')}</span>
                </button>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto p-2">
                <Accordion type="multiple" className="w-full space-y-2">
                    {filteredChannels.map((channel) => {
                        const total = channel.models.length;
                        const selectedCount = channel.models.reduce(
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
                                        {channel.models.map((m) => {
                                            const isSelected = selectedKeys.has(memberKey(m));
                                            const { Avatar } = getModelIcon(m.name);
                                            return (
                                                <button
                                                    key={memberKey(m)}
                                                    type="button"
                                                    onClick={() => !isSelected && onAdd(m)}
                                                    disabled={isSelected}
                                                    className={cn(
                                                        'w-full flex items-center justify-between gap-2 rounded-lg border border-border/50 bg-background px-2.5 py-2 text-left transition-colors',
                                                        isSelected ? 'opacity-60 cursor-not-allowed' : 'hover:bg-muted'
                                                    )}
                                                >
                                                    <span className="flex items-center gap-2 min-w-0">
                                                        <Avatar size={16} />
                                                        <span className="text-sm font-medium truncate">{m.name}</span>
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
                                </AccordionContent>
                            </AccordionItem>
                        );
                    })}
                </Accordion>
            </div>
        </div>
    );
}

function SortSection({
    members,
    onReorder,
    onRemove,
    onWeightChange,
    removingIds,
    showWeight,
    onClear,
}: {
    members: SelectedMember[];
    onReorder: (members: SelectedMember[]) => void;
    onRemove: (id: string) => void;
    onWeightChange: (id: string, weight: number) => void;
    removingIds: Set<string>;
    showWeight: boolean;
    onClear: () => void;
}) {
    const t = useTranslations('group');
    const [searchKeyword, setSearchKeyword] = useState('');
    const [filter, setFilter] = useState<SelectedMemberFilter>('all');
    const [sort, setSort] = useState<SelectedMemberSort>('manual');

    const normalizedSearch = normalizeKey(searchKeyword);

    const duplicateSignatures = useMemo(() => {
        const counts = new Map<string, number>();
        members.forEach((member) => {
            const baseUrl = normalizeKey(member.base_url);
            if (!baseUrl) return;
            const signature = `${baseUrl}::${normalizeKey(member.name)}`;
            counts.set(signature, (counts.get(signature) ?? 0) + 1);
        });
        return counts;
    }, [members]);

    const duplicateIds = useMemo(() => {
        const ids = new Set<string>();
        members.forEach((member) => {
            const baseUrl = normalizeKey(member.base_url);
            if (!baseUrl) return;
            const signature = `${baseUrl}::${normalizeKey(member.name)}`;
            if ((duplicateSignatures.get(signature) ?? 0) > 1) ids.add(member.id);
        });
        return ids;
    }, [duplicateSignatures, members]);

    const filteredMembers = useMemo(() => {
        const next = members.filter((member) => {
            const matchesSearch = !normalizedSearch || [
                member.name,
                member.channel_name,
                member.base_url,
            ].some((value) => normalizeKey(value).includes(normalizedSearch));

            if (!matchesSearch) return false;
            if (filter === 'duplicates') return duplicateIds.has(member.id);
            return true;
        });

        if (sort === 'manual') return next;

        return [...next].sort((left, right) => {
            if (sort === 'model-asc') return left.name.localeCompare(right.name);
            if (sort === 'channel-asc') {
                const channelCompare = left.channel_name.localeCompare(right.channel_name);
                if (channelCompare !== 0) return channelCompare;
                return left.name.localeCompare(right.name);
            }

            const weightDiff = (right.weight ?? 1) - (left.weight ?? 1);
            if (weightDiff !== 0) return weightDiff;
            return left.name.localeCompare(right.name);
        });
    }, [duplicateIds, filter, members, normalizedSearch, sort]);

    const dragDisabled = normalizedSearch.length > 0 || filter !== 'all' || sort !== 'manual';

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0">
            <div className="border-b border-border/30 bg-muted/50">
                <div className="flex items-center justify-between px-3 py-2">
                    <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-foreground">
                            {t('form.items')}
                            {members.length > 0 && (
                                <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                                    ({members.length})
                                </span>
                            )}
                        </span>
                        {duplicateIds.size > 0 ? (
                            <Badge variant="secondary" className="rounded-full bg-amber-500/12 text-amber-700 dark:text-amber-300">
                                {t('form.duplicateBadge', { count: duplicateIds.size })}
                            </Badge>
                        ) : null}
                    </div>
                    <button
                        type="button"
                        onClick={onClear}
                        disabled={members.length === 0}
                        className={cn(
                            'flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
                            members.length === 0
                                ? 'text-muted-foreground/50 cursor-not-allowed'
                                : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                        )}
                        title={t('form.clear')}
                    >
                        <Trash2 className="size-3.5" />
                        <span>{t('form.clear')}</span>
                    </button>
                </div>

                <div className="grid gap-2 px-3 pb-3 md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)]">
                    <div className="relative">
                        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                        <Input
                            value={searchKeyword}
                            onChange={(event) => setSearchKeyword(event.target.value)}
                            placeholder={t('form.selectedSearchPlaceholder')}
                            className="rounded-xl pl-9"
                            aria-label={t('form.selectedSearchPlaceholder')}
                        />
                    </div>

                    <Select value={filter} onValueChange={(value) => setFilter(value as SelectedMemberFilter)}>
                        <SelectTrigger className="w-full rounded-xl">
                            <SelectValue placeholder={t('form.selectedFilter')} />
                        </SelectTrigger>
                        <SelectContent className="rounded-xl">
                            <SelectItem value="all">{t('form.filters.all')}</SelectItem>
                            <SelectItem value="duplicates">{t('form.filters.duplicates')}</SelectItem>
                        </SelectContent>
                    </Select>

                    <Select value={sort} onValueChange={(value) => setSort(value as SelectedMemberSort)}>
                        <SelectTrigger className="w-full rounded-xl">
                            <SelectValue placeholder={t('form.selectedSort')} />
                        </SelectTrigger>
                        <SelectContent className="rounded-xl">
                            <SelectItem value="manual">{t('form.sorts.manual')}</SelectItem>
                            <SelectItem value="model-asc">{t('form.sorts.modelAsc')}</SelectItem>
                            <SelectItem value="channel-asc">{t('form.sorts.channelAsc')}</SelectItem>
                            <SelectItem value="weight-desc">{t('form.sorts.weightDesc')}</SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                <div className="px-3 pb-3">
                    <p className="text-[11px] text-muted-foreground">
                        {dragDisabled ? t('form.reorderLockedHint') : t('form.duplicateHint')}
                    </p>
                </div>
            </div>

            <div className="flex-1 min-h-0">
                <MemberList
                    members={filteredMembers}
                    onReorder={onReorder}
                    onRemove={onRemove}
                    onWeightChange={onWeightChange}
                    removingIds={removingIds}
                    showWeight={showWeight}
                    showConfirmDelete={false}
                    dragDisabled={dragDisabled}
                />
            </div>
        </div>
    );
}

export function GroupEditor({
    initial,
    submitText,
    submittingText,
    isSubmitting,
    onSubmit,
    onCancel,
}: {
    initial?: Partial<GroupEditorValues>;
    submitText: string;
    submittingText: string;
    isSubmitting: boolean;
    onSubmit: (values: GroupEditorValues) => void;
    onCancel?: () => void;
}) {
    const t = useTranslations('group');
    const { data: modelChannels = [] } = useModelChannelList();
    const { data: settings = [] } = useSettingList();

    const [groupName, setGroupName] = useState(initial?.name ?? '');
    const [matchRegex, setMatchRegex] = useState(initial?.match_regex ?? '');
    const [mode, setMode] = useState<GroupMode>((initial?.mode ?? 1) as GroupMode);
    const [firstTokenTimeOut, setFirstTokenTimeOut] = useState<number>(initial?.first_token_time_out ?? 0);
    const [sessionKeepTime, setSessionKeepTime] = useState<number>(initial?.session_keep_time ?? 0);
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>(initial?.members ?? []);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

    const groupKey = normalizeKey(groupName);
    const regexKey = matchRegex.trim();
    const modelChannelByKey = useMemo(() => {
        const map = new Map<string, LLMChannel>();
        modelChannels.forEach((channel) => {
            map.set(memberKey(channel), channel);
        });
        return map;
    }, [modelChannels]);
    const hydratedSelectedMembers = useMemo(() => (
        selectedMembers.map((member) => {
            const latest = modelChannelByKey.get(member.id || memberKey(member));
            if (!latest) {
                return {
                    ...member,
                    base_url: member.base_url ?? '',
                    key_count: member.key_count ?? 0,
                };
            }

            return {
                ...latest,
                ...member,
                id: member.id || memberKey(latest),
                channel_name: member.channel_name || latest.channel_name,
                base_url: member.base_url ?? latest.base_url ?? '',
                key_count: member.key_count ?? latest.key_count ?? 0,
                enabled: member.enabled ?? latest.enabled,
            };
        })
    ), [modelChannelByKey, selectedMembers]);
    const defaultFirstTokenTimeOut = useMemo(() => {
        const value = settings.find((item) => item.key === SettingKey.GroupDefaultFirstTokenTimeOut)?.value;
        const parsed = Number.parseInt(value ?? '0', 10);
        return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
    }, [settings]);
    const defaultSessionKeepTime = useMemo(() => {
        const value = settings.find((item) => item.key === SettingKey.GroupDefaultSessionKeepTime)?.value;
        const parsed = Number.parseInt(value ?? '0', 10);
        return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
    }, [settings]);

    const { matchedModelChannels, regexError } = useMemo(() => {
        const parseRegex = (input: string): RegExp => {
            const inlineMatch = input.match(/^\(\?([ism]+)\)(.+)$/);
            if (inlineMatch) {
                const flagMap: Record<string, string> = { i: 'i', s: 's', m: 'm' };
                const flags = inlineMatch[1].split('').map(f => flagMap[f] || '').join('');
                return new RegExp(inlineMatch[2], flags);
            }

            return new RegExp(input);
        };

        if (regexKey) {
            try {
                const re = parseRegex(regexKey);
                return { matchedModelChannels: modelChannels.filter((mc) => re.test(mc.name)), regexError: '' };
            } catch (e) {
                return { matchedModelChannels: [], regexError: (e as Error)?.message ?? 'Invalid regex' };
            }
        }
        if (!groupKey) return { matchedModelChannels: [], regexError: '' };
        return { matchedModelChannels: modelChannels.filter((mc) => matchesGroupName(mc.name, groupKey)), regexError: '' };
    }, [groupKey, regexKey, modelChannels]);

    const handleAddMember = useCallback((channel: LLMChannel) => {
        const key = memberKey(channel);
        setSelectedMembers((prev) => {
            if (prev.some((m) => m.id === key)) return prev;
            return [...prev, { ...channel, id: key, weight: 1 }];
        });
    }, []);

    const autoAddDisabled = useMemo(() => {
        if ((!regexKey && !groupKey) || regexError || matchedModelChannels.length === 0) return true;
        const existing = new Set(selectedMembers.map((m) => m.id));
        return matchedModelChannels.every((mc) => existing.has(memberKey(mc)));
    }, [groupKey, regexKey, regexError, matchedModelChannels, selectedMembers]);

    const handleAutoAdd = useCallback(() => {
        if (matchedModelChannels.length === 0) return;
        setSelectedMembers((prev) => {
            const existing = new Set(prev.map((m) => m.id));
            const toAdd = matchedModelChannels
                .filter((mc) => !existing.has(memberKey(mc)))
                .map((mc) => ({ ...mc, id: memberKey(mc), weight: 1 }));
            return toAdd.length ? [...prev, ...toAdd] : prev;
        });
    }, [matchedModelChannels]);

    const handleWeightChange = useCallback((id: string, weight: number) => {
        setSelectedMembers((prev) => prev.map((m) => m.id === id ? { ...m, weight } : m));
    }, []);

    const handleRemoveMember = useCallback((id: string) => {
        setRemovingIds((prev) => new Set(prev).add(id));
        setTimeout(() => {
            setSelectedMembers((prev) => prev.filter((m) => m.id !== id));
            setRemovingIds((prev) => { const n = new Set(prev); n.delete(id); return n; });
        }, 200);
    }, []);

    const handleClearMembers = useCallback(() => {
        setSelectedMembers([]);
        setRemovingIds(new Set());
    }, []);

    const isValid = groupKey.length > 0 && selectedMembers.length > 0 && !regexError;

    const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!isValid) return;
        onSubmit({
            name: groupName,
            match_regex: regexKey,
            mode,
            first_token_time_out: firstTokenTimeOut,
            session_keep_time: sessionKeepTime,
            members: selectedMembers,
        });
    };


    return (
        <form onSubmit={handleSubmit} className="flex flex-col h-full min-h-0 ">
            <div className="flex-1 min-h-0 overflow-hidden pr-1">
                <FieldGroup className="gap-4 flex flex-col min-h-0 h-full">
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                        <Field>
                            <FieldLabel htmlFor="group-name">{t('form.name')}</FieldLabel>
                            <Input
                                id="group-name"
                                value={groupName}
                                onChange={(e) => setGroupName(e.target.value)}
                                className="rounded-xl"
                            />
                        </Field>
                        <Field>
                            <FieldLabel htmlFor="group-match-regex">{t('form.matchRegex')}</FieldLabel>
                            <Input
                                id="group-match-regex"
                                value={matchRegex}
                                onChange={(e) => setMatchRegex(e.target.value)}
                                className="rounded-xl"
                                placeholder={t('form.matchRegexPlaceholder')}
                            />
                            {regexError && (
                                <p className="mt-1 text-xs text-destructive">
                                    {t('form.matchRegexInvalid')}: {regexError}
                                </p>
                            )}
                        </Field>

                        <Field>
                            <FieldLabel htmlFor="group-first-token-time-out">
                                {t('form.firstTokenTimeOut')}
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger asChild>
                                            <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                                        </TooltipTrigger>
                                        <TooltipContent>
                                            {t('form.firstTokenTimeOutHint')}
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FieldLabel>
                            <Input
                                id="group-first-token-time-out"
                                type="number"
                                inputMode="numeric"
                                min={0}
                                step={1}
                                value={firstTokenTimeOut > 0 ? String(firstTokenTimeOut) : ''}
                                placeholder={defaultFirstTokenTimeOut > 0 ? String(defaultFirstTokenTimeOut) : '0'}
                                onChange={(e) => {
                                    const raw = e.target.value;
                                    if (raw.trim() === '') {
                                        setFirstTokenTimeOut(0);
                                        return;
                                    }
                                    const n = Number.parseInt(raw, 10);
                                    setFirstTokenTimeOut(Number.isFinite(n) && n > 0 ? n : 0);
                                }}
                                className="rounded-xl"
                            />
                            <p className="mt-1 text-xs text-muted-foreground">
                                {defaultFirstTokenTimeOut > 0
                                    ? t('form.firstTokenTimeOutDefaultHint', { value: defaultFirstTokenTimeOut })
                                    : t('form.firstTokenTimeOutDisabledHint')}
                            </p>
                        </Field>

                        <Field>
                            <FieldLabel htmlFor="group-session-keep-time">
                                {t('form.sessionKeepTime')}
                                <TooltipProvider>
                                    <Tooltip>
                                        <TooltipTrigger asChild>
                                            <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                                        </TooltipTrigger>
                                        <TooltipContent>
                                            {t('form.sessionKeepTimeHint')}
                                        </TooltipContent>
                                    </Tooltip>
                                </TooltipProvider>
                            </FieldLabel>
                            <Input
                                id="group-session-keep-time"
                                type="number"
                                inputMode="numeric"
                                min={0}
                                step={1}
                                value={sessionKeepTime > 0 ? String(sessionKeepTime) : ''}
                                placeholder={defaultSessionKeepTime > 0 ? String(defaultSessionKeepTime) : '0'}
                                onChange={(e) => {
                                    const raw = e.target.value;
                                    if (raw.trim() === '') {
                                        setSessionKeepTime(0);
                                        return;
                                    }
                                    const n = Number.parseInt(raw, 10);
                                    setSessionKeepTime(Number.isFinite(n) && n > 0 ? n : 0);
                                }}
                                className="rounded-xl"
                            />
                            <p className="mt-1 text-xs text-muted-foreground">
                                {defaultSessionKeepTime > 0
                                    ? t('form.sessionKeepTimeDefaultHint', { value: defaultSessionKeepTime })
                                    : t('form.sessionKeepTimeDisabledHint')}
                            </p>
                        </Field>
                    </div>

                    {/* Mode */}
                    <div className="flex gap-1">
                        {([1, 2, 3, 4, 5] as const).map((m) => (
                            <button
                                key={m}
                                type="button"
                                onClick={() => setMode(m)}
                                className={cn(
                                    'flex-1 py-1 text-xs rounded-lg transition-colors',
                                    mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'
                                )}
                            >
                                {t(`mode.${MODE_LABELS[m]}`)}
                            </button>
                        ))}
                    </div>

                    <div className="flex-1 min-h-0">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 h-full min-h-0">
                            <ModelPickerSection
                                modelChannels={modelChannels}
                                selectedMembers={hydratedSelectedMembers}
                                onAdd={handleAddMember}
                                onAutoAdd={handleAutoAdd}
                                autoAddDisabled={autoAddDisabled}
                            />
                            <SortSection
                                members={hydratedSelectedMembers}
                                onReorder={setSelectedMembers}
                                onRemove={handleRemoveMember}
                                onWeightChange={handleWeightChange}
                                removingIds={removingIds}
                                showWeight={mode === 4 || mode === 5}
                                onClear={handleClearMembers}
                            />
                        </div>
                    </div>
                </FieldGroup>
            </div>

            <div className="pt-4 mt-auto shrink-0">
                <div className="flex gap-2">
                    {onCancel && (
                        <Button type="button" variant="secondary" className="flex-1 rounded-xl h-11" onClick={onCancel}>
                            {t('detail.actions.cancel')}
                        </Button>
                    )}
                    <Button
                        type="submit"
                        disabled={!isValid || isSubmitting}
                        className="flex-1 rounded-xl h-11"
                    >
                        {isSubmitting ? submittingText : submitText}
                    </Button>
                </div>
            </div>
        </form>
    );
}
