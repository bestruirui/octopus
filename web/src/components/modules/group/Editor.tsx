'use client';

import { useCallback, useMemo, useState, type FormEvent } from 'react';
import { create } from 'zustand';
import { Check, ChevronDownIcon, Plus, Sparkles, Tag, Trash2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { Button } from '@/components/ui/button';
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Accordion, AccordionContent, AccordionItem } from '@/components/ui/accordion';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';
import type { GroupMode } from '@/api/endpoints/group';
import type { SelectedMember } from './ItemList';
import { MemberList } from './ItemList';
import { matchesGroupName, memberKey, normalizeKey, MODE_LABELS } from './utils';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { HelpCircle } from 'lucide-react';

const EMPTY_TAGS: string[] = [];

const useTagFilterStore = create<{
    tagsByGroup: Record<string, string[]>;
    setSelectedTags: (groupId: string, tags: string[]) => void;
    toggleTag: (groupId: string, tag: string) => void;
}>((set) => ({
    tagsByGroup: {},
    setSelectedTags: (groupId, tags) => set((state) => ({
        tagsByGroup: { ...state.tagsByGroup, [groupId]: tags },
    })),
    toggleTag: (groupId, tag) => set((state) => {
        const current = state.tagsByGroup[groupId] ?? [];
        return {
            tagsByGroup: {
                ...state.tagsByGroup,
                [groupId]: current.includes(tag)
                    ? current.filter((t) => t !== tag)
                    : [...current, tag],
            },
        };
    }),
}));

export type GroupEditorValues = {
    name: string;
    match_regex: string;
    mode: GroupMode;
    first_token_time_out: number;
    session_keep_time: number;
    members: SelectedMember[];
};

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

    const selectedKeys = useMemo(() => new Set(selectedMembers.map(memberKey)), [selectedMembers]);

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

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0">
            <div className="flex items-center justify-between px-3 py-2 border-b border-border/30 bg-muted/50">
                <span className="text-sm font-medium text-foreground">
                    {t('form.addItem')}
                    <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                        ({selectedMembers.length})
                    </span>
                </span>

                <button
                    type="button"
                    onClick={onAutoAdd}
                    className={cn(
                        'flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
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
                    {channels.map((channel) => {
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

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0">
            <div className="flex items-center justify-between px-3 py-2 border-b border-border/30 bg-muted/50">
                <span className="text-sm font-medium text-foreground">
                    {t('form.items')}
                    {members.length > 0 && (
                        <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                            ({members.length})
                        </span>
                    )}
                </span>
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

            <div className="flex-1 min-h-0">
                <MemberList
                    members={members}
                    onReorder={onReorder}
                    onRemove={onRemove}
                    onWeightChange={onWeightChange}
                    removingIds={removingIds}
                    showWeight={showWeight}
                    showConfirmDelete={false}
                />
            </div>
        </div>
    );
}

export function GroupEditor({
    groupId,
    initial,
    submitText,
    submittingText,
    isSubmitting,
    onSubmit,
    onCancel,
}: {
    groupId?: number;
    initial?: Partial<GroupEditorValues>;
    submitText: string;
    submittingText: string;
    isSubmitting: boolean;
    onSubmit: (values: GroupEditorValues) => void;
    onCancel?: () => void;
}) {
    const t = useTranslations('group');
    const { data: modelChannels = [] } = useModelChannelList();

    const [groupName, setGroupName] = useState(initial?.name ?? '');
    const [matchRegex, setMatchRegex] = useState(initial?.match_regex ?? '');
    const [mode, setMode] = useState<GroupMode>((initial?.mode ?? 1) as GroupMode);
    const [firstTokenTimeOut, setFirstTokenTimeOut] = useState<number>(initial?.first_token_time_out ?? 0);
    const [sessionKeepTime, setSessionKeepTime] = useState<number>(initial?.session_keep_time ?? 0);
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>(initial?.members ?? []);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

    // Per-group tag filter: use store for edit (has groupId), local state for create
    const storeKey = groupId != null ? String(groupId) : '';
    const storeSelectedTags = useTagFilterStore((s) => s.tagsByGroup[storeKey]) ?? EMPTY_TAGS;
    const storeSetSelectedTags = useTagFilterStore((s) => s.setSelectedTags);
    const storeToggleTag = useTagFilterStore((s) => s.toggleTag);
    const [localSelectedTags, setLocalSelectedTags] = useState<string[]>([]);

    const selectedTags = groupId != null ? storeSelectedTags : localSelectedTags;
    const setSelectedTags = useCallback((tags: string[]) => {
        if (groupId != null) storeSetSelectedTags(storeKey, tags);
        else setLocalSelectedTags(tags);
    }, [groupId, storeKey, storeSetSelectedTags]);
    const toggleTag = useCallback((tag: string) => {
        if (groupId != null) storeToggleTag(storeKey, tag);
        else setLocalSelectedTags((prev) =>
            prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]
        );
    }, [groupId, storeKey, storeToggleTag]);

    const allTags = useMemo(() => {
        const tagSet = new Set<string>();
        modelChannels.forEach((mc) => mc.tags?.forEach((t) => tagSet.add(t)));
        return Array.from(tagSet).sort();
    }, [modelChannels]);

    // Derive effective tags: prune any selectedTags that no longer exist in
    // allTags. This avoids an empty model list when persisted tags become stale
    // (e.g., all channel tags removed), without needing a useEffect + setState.
    const effectiveTags = useMemo(() => {
        if (selectedTags.length === 0) return selectedTags;
        const validSet = new Set(allTags);
        return selectedTags.filter((t) => validSet.has(t));
    }, [selectedTags, allTags]);

    const filteredModelChannels = useMemo(() => {
        if (effectiveTags.length === 0) return modelChannels;
        return modelChannels.filter((mc) =>
            mc.tags?.some((t) => effectiveTags.includes(t))
        );
    }, [modelChannels, effectiveTags]);

    const groupKey = normalizeKey(groupName);
    const regexKey = matchRegex.trim();

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
                return { matchedModelChannels: filteredModelChannels.filter((mc) => re.test(mc.name)), regexError: '' };
            } catch (e) {
                return { matchedModelChannels: [], regexError: (e as Error)?.message ?? 'Invalid regex' };
            }
        }
        if (!groupKey) return { matchedModelChannels: [], regexError: '' };
        return { matchedModelChannels: filteredModelChannels.filter((mc) => matchesGroupName(mc.name, groupKey)), regexError: '' };
    }, [groupKey, regexKey, filteredModelChannels]);

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
                                value={String(firstTokenTimeOut)}
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
                                value={String(sessionKeepTime)}
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
                        </Field>
                    </div>

                    {/* Mode */}
                    <div className="flex gap-1">
                        {([1, 2, 3, 4] as const).map((m) => (
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

                    {/* Tag filter */}
                    {allTags.length > 0 && (
                        <div className="flex items-center gap-2 flex-wrap">
                            <span className="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
                                <Tag className="size-3" />
                                {t('form.filterByTag')}
                            </span>
                            {allTags.map((tag) => (
                                <button
                                    key={tag}
                                    type="button"
                                    onClick={() => toggleTag(tag)}
                                    className={cn(
                                        'px-2 py-0.5 text-xs rounded-md transition-colors',
                                        selectedTags.includes(tag)
                                            ? 'bg-primary text-primary-foreground'
                                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                                    )}
                                >
                                    {tag}
                                </button>
                            ))}
                            {selectedTags.length > 0 && (
                                <button
                                    type="button"
                                    onClick={() => setSelectedTags([])}
                                    className="px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
                                >
                                    {t('form.allChannels')}
                                </button>
                            )}
                        </div>
                    )}

                    <div className="flex-1 min-h-0">
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 h-full min-h-0">
                            <ModelPickerSection
                                modelChannels={filteredModelChannels}
                                selectedMembers={selectedMembers}
                                onAdd={handleAddMember}
                                onAutoAdd={handleAutoAdd}
                                autoAddDisabled={autoAddDisabled}
                            />
                            <SortSection
                                members={selectedMembers}
                                onReorder={setSelectedMembers}
                                onRemove={handleRemoveMember}
                                onWeightChange={handleWeightChange}
                                removingIds={removingIds}
                                showWeight={mode === 4}
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


