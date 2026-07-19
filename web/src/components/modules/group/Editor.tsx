'use client';

import { useCallback, useMemo, useState, type FormEvent } from 'react';
import { Check, ChevronDownIcon, Plus, Search, Sparkles, Trash2, ChevronUp, X } from 'lucide-react';
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
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0 h-full">
            <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 px-2 py-1.5 border-b border-border/30 bg-muted/50">
                <span className="min-w-0 justify-self-start text-xs font-medium text-foreground">
                    {t('form.addItem')}
                </span>

                <div className="relative justify-self-center w-28">
                    <Search className="pointer-events-none absolute left-1.5 top-1/2 size-3 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        value={searchKeyword}
                        onChange={(event) => setSearchKeyword(event.target.value)}
                        className="h-5 rounded-lg border-border/60 bg-background/70 pl-6 pr-1.5 text-[10px] shadow-none focus-visible:border-border/60 focus-visible:ring-0"
                        aria-label="search"
                    />
                </div>

                <button
                    type="button"
                    onClick={onAutoAdd}
                    className={cn(
                        'justify-self-end shrink-0 flex items-center gap-0.5 px-1.5 py-0.5 rounded-lg text-[10px] font-medium transition-colors',
                        autoAddDisabled
                            ? 'text-muted-foreground/50 cursor-not-allowed'
                            : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                    )}
                    disabled={autoAddDisabled}
                    title={t('form.autoAdd')}
                >
                    <Sparkles className="size-3" />
                    <span className="hidden sm:inline">{t('form.autoAdd')}</span>
                </button>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto p-1.5">
                <Accordion type="multiple" className="w-full space-y-1">
                    {filteredChannels.map((channel) => {
                        const total = channel.models.length;
                        const selectedCount = channel.models.reduce(
                            (acc, m) => acc + (selectedKeys.has(memberKey(m)) ? 1 : 0),
                            0
                        );
                        const available = total - selectedCount;

                        return (
                            <AccordionItem key={channel.id} value={`channel-${channel.id}`}>
                                <AccordionPrimitive.Header className="rounded-lg bg-muted sticky top-0 z-10 flex px-2 overflow-visible">
                                    <AccordionPrimitive.Trigger className="flex flex-1 min-w-0 items-center gap-3 py-2 text-left text-xs transition-all outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 [&[data-state=open]>svg]:rotate-180">
                                        <span className="truncate font-medium">{channel.name}</span>
                                        <span className="text-[10px] text-muted-foreground shrink-0">
                                            {available}/{total}
                                        </span>
                                        <ChevronDownIcon className="text-muted-foreground pointer-events-none size-3 shrink-0 transition-transform duration-200" />
                                    </AccordionPrimitive.Trigger>
                                </AccordionPrimitive.Header>
                                <AccordionContent className="px-1.5 pt-1">
                                    <div className="flex flex-col gap-1">
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
                                                        'w-full flex items-center justify-between gap-1.5 rounded-lg border border-border/50 bg-background px-2 py-1.5 text-left transition-colors',
                                                        isSelected ? 'opacity-60 cursor-not-allowed' : 'hover:bg-muted'
                                                    )}
                                                >
                                                    <span className="flex items-center gap-1.5 min-w-0">
                                                        <Avatar size={14} />
                                                        <span className="text-xs font-medium truncate">{m.name}</span>
                                                    </span>

                                                    <span className="shrink-0 text-muted-foreground">
                                                        {isSelected ? (
                                                            <Check className="size-3 text-primary" />
                                                        ) : (
                                                            <Plus className="size-3" />
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
        <div className="rounded-xl border border-border/50 bg-muted/30 flex flex-col min-h-0 h-full">
            <div className="flex items-center justify-between px-2 py-1 border-b border-border/30 bg-muted/50">
                <span className="text-xs font-medium text-foreground">
                    {t('form.items')}
                    {members.length > 0 && (
                        <span className="ml-1 text-[10px] text-muted-foreground font-normal">
                            ({members.length})
                        </span>
                    )}
                </span>
                <button
                    type="button"
                    onClick={onClear}
                    disabled={members.length === 0}
                    className={cn(
                        'flex items-center gap-0.5 px-1.5 py-0.5 rounded-lg text-[10px] font-medium transition-colors',
                        members.length === 0
                            ? 'text-muted-foreground/50 cursor-not-allowed'
                            : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                    )}
                    title={t('form.clear')}
                >
                    <Trash2 className="size-3" />
                    <span className="hidden sm:inline">{t('form.clear')}</span>
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
    const [showForm, setShowForm] = useState(false);
    const [showSort, setShowSort] = useState(false);

    const [groupName, setGroupName] = useState(initial?.name ?? '');
    const [matchRegex, setMatchRegex] = useState(initial?.match_regex ?? '');
    const [mode, setMode] = useState<GroupMode>((initial?.mode ?? 1) as GroupMode);
    const [firstTokenTimeOut, setFirstTokenTimeOut] = useState<number>(initial?.first_token_time_out ?? 0);
    const [sessionKeepTime, setSessionKeepTime] = useState<number>(initial?.session_keep_time ?? 0);
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>(initial?.members ?? []);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

    const groupKey = normalizeKey(groupName);
    const regexKey = matchRegex.trim();

    const { matchedModelChannels, regexError } = useMemo(() => {
        const parseRegex = (input: string): RegExp => {
            const inlineMatch = input.match(/^\\(\\?([ism]+)\\)(.+)$/);
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
        <form onSubmit={handleSubmit} className="flex flex-col h-full min-h-0 overflow-y-auto">
            {/* Mode + collapsible config bar */}
            <div className="shrink-0 flex items-center gap-1 mb-1">
                <button
                    type="button"
                    onClick={() => setShowForm(!showForm)}
                    className="shrink-0 p-1 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                    title={showForm ? t('form.hideConfig') : t('form.showConfig')}
                >
                    {showForm ? <ChevronUp className="size-3.5" /> : <ChevronDownIcon className="size-3.5" />}
                </button>
                <div className="flex gap-0.5 flex-1">
                    {([1, 2, 3, 4] as const).map((m) => (
                        <button
                            key={m}
                            type="button"
                            onClick={() => setMode(m)}
                            className={cn(
                                'flex-1 py-0.5 text-[10px] rounded-md transition-colors',
                                mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'
                            )}
                        >
                            {t(`mode.${MODE_LABELS[m]}`)}
                        </button>
                    ))}
                </div>
            </div>

            {/* Collapsible form fields */}
            {showForm && (
                <div className="shrink-0 mb-1 p-1.5 rounded-lg bg-muted/40 border border-border/30 space-y-1">
                    <div className="grid grid-cols-2 gap-1">
                        <div className="flex flex-col gap-0.5">
                            <label className="text-[10px] text-muted-foreground leading-none px-0.5">{t('form.name')}</label>
                            <Input
                                value={groupName}
                                onChange={(e) => setGroupName(e.target.value)}
                                className="h-6 rounded-lg text-[10px] px-1.5"
                            />
                        </div>
                        <div className="flex flex-col gap-0.5">
                            <label className="text-[10px] text-muted-foreground leading-none px-0.5">{t('form.matchRegex')}</label>
                            <Input
                                value={matchRegex}
                                onChange={(e) => setMatchRegex(e.target.value)}
                                className="h-6 rounded-lg text-[10px] px-1.5"
                            />
                        </div>
                        <div className="flex flex-col gap-0.5">
                            <label className="text-[10px] text-muted-foreground leading-none px-0.5">{t('form.firstTokenTimeOut')}</label>
                            <Input
                                type="number"
                                inputMode="numeric"
                                min={0}
                                step={1}
                                value={String(firstTokenTimeOut)}
                                onChange={(e) => {
                                    const raw = e.target.value;
                                    if (raw.trim() === '') { setFirstTokenTimeOut(0); return; }
                                    const n = Number.parseInt(raw, 10);
                                    setFirstTokenTimeOut(Number.isFinite(n) && n > 0 ? n : 0);
                                }}
                                className="h-6 rounded-lg text-[10px] px-1.5"
                            />
                        </div>
                        <div className="flex flex-col gap-0.5">
                            <label className="text-[10px] text-muted-foreground leading-none px-0.5">{t('form.sessionKeepTime')}</label>
                            <Input
                                type="number"
                                inputMode="numeric"
                                min={0}
                                step={1}
                                value={String(sessionKeepTime)}
                                onChange={(e) => {
                                    const raw = e.target.value;
                                    if (raw.trim() === '') { setSessionKeepTime(0); return; }
                                    const n = Number.parseInt(raw, 10);
                                    setSessionKeepTime(Number.isFinite(n) && n > 0 ? n : 0);
                                }}
                                className="h-6 rounded-lg text-[10px] px-1.5"
                            />
                        </div>
                    </div>
                    {regexError && (
                        <p className="text-[10px] text-destructive leading-tight">{t('form.matchRegexInvalid')}: {regexError}</p>
                    )}
                </div>
            )}

            {/* Main content: ModelList fills all space */}
            <div className="flex-1 min-h-0 flex flex-col gap-1">
                <div className="flex-1 min-h-0">
                    <ModelPickerSection
                        modelChannels={modelChannels}
                        selectedMembers={selectedMembers}
                        onAdd={handleAddMember}
                        onAutoAdd={handleAutoAdd}
                        autoAddDisabled={autoAddDisabled}
                    />
                </div>
            </div>

            {/* Bottom bar: selected models chips + expand */}
            <div className="shrink-0 border-t border-border/30 pt-1 mt-1">
                {selectedMembers.length > 0 && (
                    <div className="flex flex-col">
                        {/* Chips row */}
                        <div className="flex items-center gap-1 flex-wrap mb-1">
                            <button
                                type="button"
                                onClick={() => setShowSort(!showSort)}
                                className="shrink-0 flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-primary/10 text-primary text-[10px] font-medium hover:bg-primary/20 transition-colors"
                            >
                                {selectedMembers.length}
                                {showSort ? <ChevronDownIcon className="size-2.5" /> : <ChevronUp className="size-2.5" />}
                            </button>
                            {selectedMembers.slice(0, 3).map((m) => (
                                <span key={m.id} className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-md bg-muted text-[10px] text-muted-foreground truncate max-w-24">
                                    <span className="truncate">{m.name}</span>
                                    <button
                                        type="button"
                                        onClick={(e) => { e.stopPropagation(); handleRemoveMember(m.id); }}
                                        className="shrink-0 hover:text-foreground"
                                    >
                                        <X className="size-2.5" />
                                    </button>
                                </span>
                            ))}
                            {selectedMembers.length > 3 && (
                                <span className="text-[10px] text-muted-foreground">+{selectedMembers.length - 3}</span>
                            )}
                        </div>

                        {/* Expanded sort list */}
                        {showSort && (
                            <div className="max-h-32 overflow-y-auto rounded-lg border border-border/30 bg-muted/20 p-1">
                                <div className="flex items-center justify-between px-1 mb-1">
                                    <span className="text-[10px] font-medium text-muted-foreground">{t('form.items')} ({selectedMembers.length})</span>
                                    <button
                                        type="button"
                                        onClick={handleClearMembers}
                                        disabled={selectedMembers.length === 0}
                                        className="flex items-center gap-0.5 px-1 py-0.5 rounded text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted disabled:opacity-50"
                                    >
                                        <Trash2 className="size-2.5" />
                                        {t('form.clear')}
                                    </button>
                                </div>
                                <MemberList
                                    members={selectedMembers}
                                    onReorder={setSelectedMembers}
                                    onRemove={handleRemoveMember}
                                    onWeightChange={handleWeightChange}
                                    removingIds={removingIds}
                                    showWeight={mode === 4}
                                    showConfirmDelete={false}
                                />
                            </div>
                        )}
                    </div>
                )}
                {/* Submit buttons */}
                <div className="flex gap-2">
                    {onCancel && (
                        <Button type="button" variant="secondary" className="flex-1 rounded-xl h-8 text-xs" onClick={onCancel}>
                            {t('detail.actions.cancel')}
                        </Button>
                    )}
                    <Button
                        type="submit"
                        disabled={!isValid || isSubmitting}
                        className="flex-1 rounded-xl h-8 text-xs"
                    >
                        {isSubmitting ? submittingText : submitText}
                    </Button>
                </div>
            </div>
        </form>
    );
}