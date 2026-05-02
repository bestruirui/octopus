'use client';

import { useCallback, useMemo, useState, type FormEvent } from 'react';
import { Check, ChevronDownIcon, Plus, Search, Sparkles, Trash2, Waves, Orbit, SlidersHorizontal, FlaskConical } from 'lucide-react';
import { useTranslations } from 'next-intl';
import * as AccordionPrimitive from '@radix-ui/react-accordion';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { Button } from '@/components/ui/button';
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Accordion, AccordionContent, AccordionItem } from '@/components/ui/accordion';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';
import { GroupMode } from '@/api/endpoints/group';
import type { SelectedMember } from './ItemList';
import { MemberList } from './ItemList';
import { matchesGroupName, memberKey, normalizeKey, MODE_LABELS, ENDPOINT_TYPE_OPTIONS, normalizeEndpointType } from './utils';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { HelpCircle } from 'lucide-react';



export type GroupEditorValues = {
    name: string;
    endpoint_type: string;
    match_regex: string;
    condition: string;
    mode: GroupMode;
    first_token_time_out: number;
    session_keep_time: number;
    members: SelectedMember[];
};

function dedupeSelectedMembers(members: SelectedMember[]) {
    const seen = new Set<string>();
    const deduped: SelectedMember[] = [];

    for (const member of members) {
        const key = memberKey(member);
        if (seen.has(key)) {
            continue;
        }
        seen.add(key);
        deduped.push({ ...member, id: key, weight: member.weight ?? 1 });
    }

    return deduped;
}

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
        <div className="waterhouse-pod flex min-h-[22rem] flex-col rounded-[1.85rem] border border-border/30 bg-background/38 shadow-waterhouse-soft lg:min-h-0">
            <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-b border-border/20 px-4 py-3">
                <div className="min-w-0">
                    <div className="inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/40 px-2.5 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground shadow-waterhouse-soft">
                        <Orbit className="size-3.5" />
                        {t('form.addItem')}
                    </div>
                </div>

                <button
                    type="button"
                    onClick={onAutoAdd}
                    className={cn(
                        'justify-self-end shrink-0 flex items-center gap-1 rounded-[1rem] px-3 py-1.5 text-xs font-medium transition-colors',
                        autoAddDisabled
                            ? 'cursor-not-allowed text-muted-foreground/50'
                            : 'bg-background/44 text-muted-foreground shadow-waterhouse-soft hover:bg-background/62 hover:text-foreground'
                    )}
                    disabled={autoAddDisabled}
                    title={t('form.autoAdd')}
                >
                    <Sparkles className="size-3.5" />
                    <span>{t('form.autoAdd')}</span>
                </button>
            </div>

            <div className="border-b border-border/15 px-4 py-3">
                <div className="relative w-full">
                    <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        value={searchKeyword}
                        onChange={(event) => setSearchKeyword(event.target.value)}
                        className="h-9 rounded-[1.15rem] border-border/35 bg-background/58 pl-8 pr-3 text-sm shadow-waterhouse-soft focus-visible:border-ring focus-visible:ring-4 focus-visible:ring-ring/20"
                        aria-label={t('form.searchAriaLabel')}
                    />
                </div>
            </div>

            <div className="flex-1 min-h-0 overflow-y-auto p-3">
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
                                <AccordionPrimitive.Header className="sticky top-0 z-10 flex overflow-hidden rounded-[1.25rem] border border-border/25 bg-background/54 px-3 shadow-waterhouse-soft backdrop-blur-md">
                                    <AccordionPrimitive.Trigger className="flex min-w-0 flex-1 items-center gap-4 py-3.5 text-left text-sm transition-all outline-none focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 [&[data-state=open]>svg]:rotate-180">
                                        <span className="truncate">{channel.name}</span>
                                        <span className="text-xs text-muted-foreground shrink-0">
                                            {available}/{total}
                                        </span>
                                        <ChevronDownIcon className="text-muted-foreground pointer-events-none size-4 shrink-0 transition-transform duration-200" />
                                    </AccordionPrimitive.Trigger>
                                </AccordionPrimitive.Header>
                                <AccordionContent className="px-1 pt-2">
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
                                                        'w-full flex items-center justify-between gap-2 rounded-[1.15rem] border border-border/30 bg-background/48 px-3 py-2.5 text-left transition-[transform,border-color,background-color,box-shadow] duration-300',
                                                        isSelected ? 'cursor-not-allowed opacity-60' : 'shadow-waterhouse-soft hover:-translate-y-0.5 hover:border-primary/18 hover:bg-background/68'
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
        <div className="waterhouse-pod flex min-h-[28rem] flex-col rounded-[1.85rem] border border-border/30 bg-background/34 shadow-waterhouse-soft lg:min-h-0">
            <div className="flex items-center justify-between border-b border-border/20 px-4 py-3">
                <span className="inline-flex items-center gap-2 text-sm font-medium text-foreground">
                    <FlaskConical className="size-4 text-primary" />
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
                        'flex items-center gap-1 rounded-[1rem] px-3 py-1.5 text-xs font-medium transition-colors',
                        members.length === 0
                            ? 'cursor-not-allowed text-muted-foreground/50'
                            : 'bg-background/44 text-muted-foreground shadow-waterhouse-soft hover:bg-background/62 hover:text-foreground'
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
    const conditionPlaceholder = '[{"key":"model","op":"contains","value":"gpt-4"}]';

    const [groupName, setGroupName] = useState(initial?.name ?? '');
    const [endpointType, setEndpointType] = useState(normalizeEndpointType(initial?.endpoint_type));
    const [matchRegex, setMatchRegex] = useState(initial?.match_regex ?? '');
    const [mode, setMode] = useState<GroupMode>((initial?.mode ?? GroupMode.Auto) as GroupMode);
    const [firstTokenTimeOut, setFirstTokenTimeOut] = useState<number>(initial?.first_token_time_out ?? 0);
    const [sessionKeepTime, setSessionKeepTime] = useState<number>(initial?.session_keep_time ?? 0);
    const [condition, setCondition] = useState(initial?.condition ?? '');
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>(dedupeSelectedMembers(initial?.members ?? []));
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

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
            return toAdd.length ? dedupeSelectedMembers([...prev, ...toAdd]) : prev;
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
            endpoint_type: endpointType,
            match_regex: regexKey,
            mode,
            first_token_time_out: firstTokenTimeOut,
            session_keep_time: sessionKeepTime,
            condition,
            members: dedupeSelectedMembers(selectedMembers),
        });
    };


    return (
        <form onSubmit={handleSubmit} className="flex h-full min-h-0 flex-col overflow-hidden">
            <div className="flex-1 min-h-0 overflow-y-auto pr-1">
                <FieldGroup className="flex min-h-full flex-col gap-4 lg:h-full">
                    <div className="grid min-h-full gap-4 xl:grid-cols-[minmax(21rem,0.9fr)_minmax(0,1.55fr)] xl:items-stretch">
                        <section className="waterhouse-pod flex flex-col gap-4 rounded-[2rem] border border-border/30 bg-background/38 p-4 shadow-waterhouse-soft md:p-5">
                            <div className="flex flex-wrap items-center justify-between gap-3">
                                <div className="space-y-2">
                                    <div className="inline-flex items-center gap-2 rounded-full border border-primary/14 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft">
                                        <Waves className="size-3.5" />
                                        {t('form.name')}
                                    </div>
                                    <p className="text-sm text-muted-foreground">{t('emptyState.description')}</p>
                                </div>
                                <div className="inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/42 px-3 py-1 text-xs text-muted-foreground shadow-waterhouse-soft">
                                    <SlidersHorizontal className="size-3.5" />
                                    {t('mode.auto')}
                                </div>
                            </div>

                            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                                <Field>
                                    <FieldLabel htmlFor="group-name">{t('form.name')}</FieldLabel>
                                    <Input
                                        id="group-name"
                                        value={groupName}
                                        onChange={(e) => setGroupName(e.target.value)}
                                        className="rounded-[1.2rem]"
                                    />
                                </Field>
                                <Field>
                                    <FieldLabel htmlFor="group-endpoint-type">{t('form.endpointType.label')}</FieldLabel>
                                    <select
                                        id="group-endpoint-type"
                                        value={endpointType}
                                        onChange={(e) => setEndpointType(normalizeEndpointType(e.target.value))}
                                        className="h-10 w-full rounded-[1.2rem] border border-border/40 bg-background/55 px-3 text-sm shadow-nature-organic backdrop-blur-md transition-[border-color,box-shadow,background-color] duration-300 outline-none hover:border-primary/15 hover:shadow-[var(--waterhouse-shadow-soft)] focus-visible:border-ring focus-visible:ring-4 focus-visible:ring-ring/20"
                                    >
                                        {ENDPOINT_TYPE_OPTIONS.map((option) => (
                                            <option key={option.value} value={option.value}>
                                                {t(option.labelKey)}
                                            </option>
                                        ))}
                                    </select>
                                </Field>
                                <Field className="md:col-span-2">
                                    <FieldLabel htmlFor="group-match-regex">{t('form.matchRegex')}</FieldLabel>
                                    <Input
                                        id="group-match-regex"
                                        value={matchRegex}
                                        onChange={(e) => setMatchRegex(e.target.value)}
                                        className="rounded-[1.2rem]"
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
                                                    <HelpCircle className="size-4 cursor-help text-muted-foreground" />
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
                                        className="rounded-[1.2rem]"
                                    />
                                </Field>
                                <Field>
                                    <FieldLabel htmlFor="group-session-keep-time">
                                        {t('form.sessionKeepTime')}
                                        <TooltipProvider>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <HelpCircle className="size-4 cursor-help text-muted-foreground" />
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
                                        className="rounded-[1.2rem]"
                                    />
                                </Field>
                                <Field className="md:col-span-2">
                                    <FieldLabel htmlFor="group-condition">
                                        {t('form.condition.label')}
                                        <TooltipProvider>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <HelpCircle className="size-4 cursor-help text-muted-foreground" />
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    {t('form.condition.hint')}<br />
                                                    {conditionPlaceholder}
                                                </TooltipContent>
                                            </Tooltip>
                                        </TooltipProvider>
                                    </FieldLabel>
                                    <Input
                                        id="group-condition"
                                        value={condition}
                                        onChange={(e) => setCondition(e.target.value)}
                                        className="rounded-[1.2rem] font-mono text-xs"
                                        placeholder={conditionPlaceholder}
                                    />
                                </Field>
                            </div>

                            <div className="space-y-2">
                                <div className="inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground shadow-waterhouse-soft">
                                    <Sparkles className="size-3.5" />
                                    {t('mode.auto')}
                                </div>
                                <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
                                    {([1, 2, 3, 4, 5] as const).map((m) => (
                                        <button
                                            key={m}
                                            type="button"
                                            onClick={() => setMode(m)}
                                            className={cn(
                                                'rounded-[1.1rem] px-3 py-2 text-xs font-medium transition-[transform,border-color,background-color,box-shadow] duration-300',
                                                mode === m
                                                    ? 'border border-primary/20 bg-primary text-primary-foreground shadow-waterhouse-soft'
                                                    : 'border border-border/30 bg-background/46 text-foreground shadow-waterhouse-soft hover:-translate-y-0.5 hover:border-primary/16 hover:bg-background/64'
                                            )}
                                        >
                                            {t(`mode.${MODE_LABELS[m]}`)}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </section>

                        <section className="waterhouse-pod flex min-h-[34rem] flex-col gap-4 rounded-[2rem] border border-border/30 bg-background/34 p-4 shadow-waterhouse-soft md:p-5 xl:min-h-0">
                            <div className="flex flex-wrap items-center justify-between gap-3">
                                <div className="space-y-2">
                                    <div className="inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-primary shadow-waterhouse-soft">
                                        <FlaskConical className="size-3.5" />
                                        {t('form.items')}
                                    </div>
                                    <p className="text-sm text-muted-foreground">{t('card.empty')}</p>
                                </div>
                                <div className="inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/42 px-3 py-1 text-xs text-muted-foreground shadow-waterhouse-soft">
                                    {selectedMembers.length}
                                </div>
                            </div>

                            <div className="grid grid-cols-1 gap-4 lg:flex-1 lg:grid-cols-[minmax(0,0.92fr)_minmax(0,1.18fr)] lg:min-h-0">
                                <ModelPickerSection
                                    modelChannels={modelChannels}
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
                                    showWeight={mode === 4 || mode === 5}
                                    onClear={handleClearMembers}
                                />
                            </div>
                        </section>
                    </div>
                </FieldGroup>
            </div>

            <div className="mt-auto shrink-0 pt-4">
                <div className="flex gap-2">
                    {onCancel && (
                        <Button type="button" variant="secondary" className="h-11 flex-1 rounded-[1.25rem]" onClick={onCancel}>
                            {t('detail.actions.cancel')}
                        </Button>
                    )}
                    <Button
                        type="submit"
                        disabled={!isValid || isSubmitting}
                        className="h-11 flex-1 rounded-[1.25rem]"
                    >
                        {isSubmitting ? submittingText : submitText}
                    </Button>
                </div>
            </div>
        </form>
    );
}
