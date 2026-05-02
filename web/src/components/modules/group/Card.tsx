'use client';

import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Trash2, X, Pencil, Activity, Loader2, CircleCheck, CircleX, Clock3, Layers, Waves, Orbit, TestTubeDiagonal } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { type Group, useDeleteGroup, useUpdateGroup, useTestGroup, useGroupTestProgress, type GroupTestResult } from '@/api/endpoints/group';
import { useModelChannelList } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { toast } from '@/components/common/Toast';
import { CopyIconButton } from '@/components/common/CopyButton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import type { SelectedMember } from './ItemList';
import { MemberList } from './ItemList';
import { GroupEditor, type GroupEditorValues } from './Editor';
import { AIRouteButton } from './AIRouteButton';
import { buildChannelNameByModelKey, modelChannelKey, MODE_LABELS, inferGroupCapabilities, CAPABILITY_LABEL_KEYS, CAPABILITY_COLORS, endpointTypeLabelKey, normalizeEndpointType } from './utils';
import { GroupMode, type GroupUpdateRequest } from '@/api/endpoints/group';
import {
    MorphingDialog,
    MorphingDialogClose,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogDescription,
    MorphingDialogTitle,
    MorphingDialogTrigger,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { Progress } from '@/components/ui/progress';

interface EditDialogContentProps {
    group: Group;
    displayMembers: SelectedMember[];
    isSubmitting: boolean;
    onSubmit: (values: GroupEditorValues, onDone?: () => void) => void;
}

function EditDialogContent({ group, displayMembers, isSubmitting, onSubmit }: EditDialogContentProps) {
    const { setIsOpen } = useMorphingDialog();
    const t = useTranslations('group');
    return (
        <>
            <MorphingDialogTitle className="shrink-0">
                <header className="relative mb-4 flex items-start justify-between gap-4">
                    <div className="space-y-3">
                        <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                            <Waves className="size-3.5" />
                            {t('detail.actions.edit')}
                        </div>
                        <div className="space-y-1">
                            <h2 className="text-2xl font-bold text-card-foreground">
                                {t('detail.actions.edit')}
                            </h2>
                            <p className="text-sm text-muted-foreground">{group.name}</p>
                        </div>
                    </div>
                    <div className="flex items-center gap-2">
                        {group.id ? (
                            <AIRouteButton
                                scope="group"
                                groupId={group.id}
                                variant="default"
                                className="h-10 rounded-[1.2rem] px-3"
                                onSuccess={() => setIsOpen(false)}
                            />
                        ) : null}
                        <MorphingDialogClose className="relative right-0 top-0" />
                    </div>
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription className="flex-1 min-h-0 overflow-hidden">
                <GroupEditor
                    key={`edit-group-${group.id}`}
                    initial={{
                        name: group.name,
                        endpoint_type: normalizeEndpointType(group.endpoint_type),
                        match_regex: group.match_regex ?? '',
                        condition: group.condition ?? '',
                        mode: group.mode,
                        first_token_time_out: group.first_token_time_out ?? 0,
                        session_keep_time: group.session_keep_time ?? 0,
                        members: displayMembers,
                    }}
                    submitText={t('detail.actions.save')}
                    submittingText={t('create.submitting')}
                    isSubmitting={isSubmitting}
                    onCancel={() => setIsOpen(false)}
                    onSubmit={(v) => onSubmit(v, () => setIsOpen(false))}
                />
            </MorphingDialogDescription>
        </>
    );
}

export function GroupCard({ group }: { group: Group }) {
    const t = useTranslations('group');
    const updateGroup = useUpdateGroup();
    const deleteGroup = useDeleteGroup();
    const testGroup = useTestGroup();
    const { data: modelChannels = [] } = useModelChannelList();

    const channelNameByKey = useMemo(() => buildChannelNameByModelKey(modelChannels), [modelChannels]);
    const enabledByKey = useMemo(() => {
        const map = new Map<string, boolean>();
        modelChannels.forEach((mc) => {
            map.set(modelChannelKey(mc.channel_id, mc.name), mc.enabled);
        });
        return map;
    }, [modelChannels]);

    const displayMembers = useMemo((): SelectedMember[] =>
        [...(group.items || [])]
            .sort((a, b) => a.priority - b.priority)
            .map((item) => ({
                id: modelChannelKey(item.channel_id, item.model_name),
                name: item.model_name,
                enabled: enabledByKey.get(modelChannelKey(item.channel_id, item.model_name)) ?? true,
                channel_id: item.channel_id,
                channel_name: channelNameByKey.get(modelChannelKey(item.channel_id, item.model_name)) ?? t('aiRoute.progress.channelFallbackName', { id: item.channel_id }),
                item_id: item.id,
                weight: item.weight,
            })),
        [group.items, channelNameByKey, enabledByKey, t]
    );

    const [confirmDelete, setConfirmDelete] = useState(false);
    const [members, setMembers] = useState<SelectedMember[]>(displayMembers);
    const [isDragging, setIsDragging] = useState(false);
    const [currentTestId, setCurrentTestId] = useState<string | null>(null);
    const weightTimerRef = useRef<NodeJS.Timeout | null>(null);
    const membersRef = useRef<SelectedMember[]>([]);
    const lastDisplayMembersRef = useRef(displayMembers);
    const handledTestCompletionRef = useRef<string | null>(null);
    const testProgressQuery = useGroupTestProgress(currentTestId);
    const testProgress = testProgressQuery.data;

    useEffect(() => {
        membersRef.current = members;
    }, [members]);

    useEffect(() => {
        if (isDragging || lastDisplayMembersRef.current === displayMembers) {
            return;
        }

        lastDisplayMembersRef.current = displayMembers;
        const frameId = window.requestAnimationFrame(() => {
            setMembers(displayMembers);
        });

        return () => window.cancelAnimationFrame(frameId);
    }, [displayMembers, isDragging]);

    useEffect(() => {
        return () => { if (weightTimerRef.current) clearTimeout(weightTimerRef.current); };
    }, []);

    const onSuccess = useCallback(() => toast.success(t('toast.updated')), [t]);
    const onError = useCallback((error: Error) => toast.error(t('toast.updateFailed'), { description: error.message }), [t]);
    const getRemovableSuggestion = useCallback((results: GroupTestResult[]) => results
        .filter((result) => !result.passed)
        .map((result) => `${result.model_name} @ ${result.channel_name}`), []);

    useEffect(() => {
        if (!testProgress?.done || !testProgress.id || handledTestCompletionRef.current === testProgress.id) {
            return;
        }

        handledTestCompletionRef.current = testProgress.id;

        if (testProgress.message) {
            toast.error(t('toast.testRequestFailed'), { description: testProgress.message });
            return;
        }

        const failedResults = (testProgress.results ?? []).filter((result) => !result.passed);
        if (failedResults.length === 0) {
            toast.success(t('toast.testAllPassed'));
            return;
        }

        const removable = getRemovableSuggestion(failedResults);
        toast.warning(t('toast.testPartialFailed'), {
            description: removable.length > 0
                ? `${t('toast.removableSuggestion')}: ${removable.join(', ')}`
                : t('toast.testPartialFailedDescription'),
            duration: 8000,
        });
    }, [getRemovableSuggestion, t, testProgress]);

    const handleTestGroup = useCallback(() => {
        if (!group.id) return;
        setCurrentTestId(null);
        handledTestCompletionRef.current = null;
        testGroup.mutate(group.id, {
            onSuccess: (progress) => {
                setCurrentTestId(progress.id);
            },
            onError: (error: Error) => {
                toast.error(t('toast.testRequestFailed'), { description: error.message });
            },
        });
    }, [group.id, t, testGroup]);

    // Avoid UI flicker: drag-reorder also uses the same mutation, so only "mode switch" should lock mode buttons.
    const isUpdatingMode = (() => {
        if (!updateGroup.isPending) return false;
        const v = updateGroup.variables;
        if (typeof v !== 'object' || v === null) return false;
        return 'mode' in v && typeof (v as { mode?: unknown }).mode === 'number';
    })();

    const priorityByItemId = useMemo(() => {
        const map = new Map<number, number>();
        (group.items || []).forEach((item) => {
            if (item.id !== undefined) map.set(item.id, item.priority);
        });
        return map;
    }, [group.items]);

    const handleDragStart = useCallback(() => { setIsDragging(true); }, []);
    const handleDragFinish = useCallback(() => { setIsDragging(false); }, []);

    const handleDropReorder = useCallback((nextMembers: SelectedMember[]) => {
        const itemsToUpdate = nextMembers
            .map((m, i) => ({ member: m, newPriority: i + 1 }))
            .filter(({ member, newPriority }) => {
                if (!member.item_id) return false;
                const origPriority = priorityByItemId.get(member.item_id);
                return origPriority !== undefined && origPriority !== newPriority;
            })
            .map(({ member, newPriority }) => ({ id: member.item_id!, priority: newPriority, weight: member.weight ?? 1 }));
        if (itemsToUpdate.length > 0) updateGroup.mutate({ id: group.id!, items_to_update: itemsToUpdate }, { onSuccess, onError });
    }, [group.id, priorityByItemId, updateGroup, onSuccess, onError]);

    const handleRemoveMember = useCallback((id: string) => {
        const member = members.find((m) => m.id === id);
        if (member?.item_id !== undefined) updateGroup.mutate({ id: group.id!, items_to_delete: [member.item_id] }, { onSuccess, onError });
    }, [members, group.id, updateGroup, onSuccess, onError]);

    const handleRemoveFailedMembers = useCallback(() => {
        if (!group.id) return;

        const failedIds = Array.from(new Set(
            (testProgress?.results ?? [])
                .filter((result) => !result.passed)
                .map((result) => result.item_id)
                .filter((itemId): itemId is number => typeof itemId === 'number')
        ));

        if (failedIds.length === 0) {
            return;
        }

        updateGroup.mutate(
            { id: group.id, items_to_delete: failedIds },
            {
                onSuccess: () => {
                    setCurrentTestId(null);
                    handledTestCompletionRef.current = null;
                    onSuccess();
                    toast.success(t('toast.removedFailedModels'));
                },
                onError,
            }
        );
    }, [group.id, onError, onSuccess, t, testProgress?.results, updateGroup]);

    const handleWeightChange = useCallback((id: string, weight: number) => {
        setMembers((prev) => prev.map((m) => m.id === id ? { ...m, weight } : m));
        if (weightTimerRef.current) clearTimeout(weightTimerRef.current);
        weightTimerRef.current = setTimeout(() => {
            const member = membersRef.current.find((m) => m.id === id);
            if (!member?.item_id) return;
            const priority = priorityByItemId.get(member.item_id);
            if (!priority) return;
            updateGroup.mutate(
                { id: group.id!, items_to_update: [{ id: member.item_id, priority, weight }] },
                { onSuccess, onError }
            );
        }, 500);
    }, [group.id, priorityByItemId, updateGroup, onSuccess, onError]);

    const handleSubmitEdit = useCallback((values: GroupEditorValues, onDone?: () => void) => {
        if (!group.id) return;

        const originalItems = [...(group.items || [])].sort((a, b) => a.priority - b.priority);
        const originalById = new Map<number, { priority: number; weight: number }>();
        const originalIds = new Set<number>();
        originalItems.forEach((it) => {
            if (typeof it.id === 'number') {
                originalIds.add(it.id);
                originalById.set(it.id, { priority: it.priority, weight: it.weight });
            }
        });

        const newIds = new Set<number>();
        values.members.forEach((m) => { if (typeof m.item_id === 'number') newIds.add(m.item_id); });

        const items_to_delete = Array.from(originalIds).filter((id) => !newIds.has(id));

        const items_to_add = values.members
            .map((m, idx) => ({ m, priority: idx + 1 }))
            .filter(({ m }) => typeof m.item_id !== 'number')
            .map(({ m, priority }) => ({
                channel_id: m.channel_id,
                model_name: m.name,
                priority,
                weight: m.weight ?? 1,
            }));

        const items_to_update = values.members
            .map((m, idx) => ({ m, priority: idx + 1 }))
            .filter(({ m }) => typeof m.item_id === 'number')
            .map(({ m, priority }) => {
                const id = m.item_id!;
                const orig = originalById.get(id);
                const weight = m.weight ?? 1;
                if (!orig) return null;
                if (orig.priority === priority && orig.weight === weight) return null;
                return { id, priority, weight };
            })
            .filter((x): x is { id: number; priority: number; weight: number } => x !== null);

        const payload: GroupUpdateRequest = { id: group.id };
        const nextName = values.name.trim();
        const nextEndpointType = normalizeEndpointType(values.endpoint_type);
        const nextRegex = (values.match_regex ?? '').trim();
        const nextCondition = values.condition.trim();
        const nextFirstTokenTimeOut = values.first_token_time_out ?? 0;
        const nextSessionKeepTime = values.session_keep_time ?? 0;

        if (nextName && nextName !== group.name) payload.name = nextName;
        if (nextEndpointType !== normalizeEndpointType(group.endpoint_type)) payload.endpoint_type = nextEndpointType;
        if (values.mode !== group.mode) payload.mode = values.mode;
        if (nextRegex !== (group.match_regex ?? '')) payload.match_regex = nextRegex;
        if (nextCondition !== (group.condition ?? '')) payload.condition = nextCondition;
        if (nextFirstTokenTimeOut !== (group.first_token_time_out ?? 0)) payload.first_token_time_out = nextFirstTokenTimeOut;
        if (nextSessionKeepTime !== (group.session_keep_time ?? 0)) payload.session_keep_time = nextSessionKeepTime;
        if (items_to_add.length) payload.items_to_add = items_to_add;
        if (items_to_update.length) payload.items_to_update = items_to_update;
        if (items_to_delete.length) payload.items_to_delete = items_to_delete;

        if (Object.keys(payload).length === 1) {
            onDone?.();
            return;
        }

        updateGroup.mutate(payload, {
            onSuccess: () => {
                onSuccess();
                onDone?.();
            },
            onError,
        });
    }, [group.condition, group.endpoint_type, group.first_token_time_out, group.session_keep_time, group.id, group.items, group.match_regex, group.mode, group.name, onSuccess, onError, updateGroup]);

    const failedTestResults = useMemo(
        () => (testProgress?.done ? (testProgress.results ?? []).filter((result) => !result.passed) : []),
        [testProgress]
    );

    const resultByItemId = useMemo(() => {
        const map = new Map<number, GroupTestResult>();
        const activeResults = testProgress?.results ?? [];
        activeResults.forEach((result) => {
            map.set(result.item_id, result);
        });
        return map;
    }, [testProgress]);

    const isTesting = testGroup.isPending || (!!currentTestId && !testProgress?.done);
    const completedCount = testProgress?.completed ?? 0;
    const totalCount = testProgress?.total ?? group.items?.length ?? 0;
    const progressValue = totalCount > 0 ? (completedCount / totalCount) * 100 : 0;

    return (
        <article className="waterhouse-island group relative flex flex-col overflow-hidden rounded-[2.1rem] border border-border/35 bg-card/60 p-4 text-card-foreground shadow-waterhouse-soft md:bg-card/58 md:shadow-waterhouse-deep md:backdrop-blur-[var(--waterhouse-shell-blur)]">
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_14%,color-mix(in_oklch,var(--waterhouse-highlight)_16%,transparent)_0%,transparent_26%),linear-gradient(150deg,color-mix(in_oklch,white_10%,transparent),transparent_48%,color-mix(in_oklch,var(--primary)_8%,transparent))]" />
            <header className="relative mb-4 overflow-visible rounded-[1.7rem] border border-border/25 bg-background/36 px-4 py-4 shadow-none md:shadow-waterhouse-soft">
                <div className="flex items-start justify-between gap-3">
                <div className="relative mr-2 min-w-0 flex-1 group/title">
                    <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-primary/10 bg-background/44 px-2.5 py-1 text-[0.64rem] font-semibold uppercase tracking-[0.18em] text-primary shadow-waterhouse-soft">
                        <Orbit className="size-3.5" />
                        {t('card.endpointType', {
                            value: t(endpointTypeLabelKey(group.endpoint_type) ?? 'form.endpointType.options.all'),
                        })}
                    </div>
                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger asChild>
                            <h3 className="truncate text-xl font-bold tracking-tight">{group.name}</h3>
                        </TooltipTrigger>
                        <TooltipContent key={group.name}>{group.name}</TooltipContent>
                    </Tooltip>
                </div>

                <div className="flex shrink-0 items-center gap-1">
                    <MorphingDialog>
                        <MorphingDialogTrigger className="rounded-[1rem] p-2 text-muted-foreground transition-colors hover:bg-background/70 hover:text-foreground">
                            <Tooltip side="top" sideOffset={10} align="center">
                                <TooltipTrigger asChild>
                                    <Pencil className="size-4" />
                                </TooltipTrigger>
                                <TooltipContent>{t('detail.actions.edit')}</TooltipContent>
                            </Tooltip>
                        </MorphingDialogTrigger>

                        <MorphingDialogContainer>
                            <MorphingDialogContent className="relative flex h-[calc(100dvh-2rem)] w-[min(100vw-2rem,92rem)] max-w-full flex-col overflow-hidden rounded-[2.4rem] border border-border/35 bg-background/80 px-4 py-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:h-[calc(100dvh-3rem)] md:px-6">
                                <EditDialogContent
                                    group={group}
                                    displayMembers={displayMembers}
                                    isSubmitting={updateGroup.isPending}
                                    onSubmit={handleSubmitEdit}
                                />
                            </MorphingDialogContent>
                        </MorphingDialogContainer>
                    </MorphingDialog>

                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger asChild>
                            <button
                                type="button"
                                onClick={handleTestGroup}
                                disabled={isTesting || !group.id}
                                className="rounded-[1rem] p-2 text-muted-foreground transition-colors hover:bg-background/70 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {isTesting ? <Loader2 className="size-4 animate-spin" /> : <Activity className="size-4" />}
                            </button>
                        </TooltipTrigger>
                        <TooltipContent>{t('detail.actions.testAvailability')}</TooltipContent>
                    </Tooltip>

                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger>
                            <CopyIconButton
                                text={group.name}
                                className="rounded-[1rem] p-2 text-muted-foreground transition-colors hover:bg-background/70 hover:text-foreground"
                                copyIconClassName="size-4"
                                checkIconClassName="size-4 text-primary"
                            />
                        </TooltipTrigger>
                        <TooltipContent>{t('detail.actions.copyName')}</TooltipContent>
                    </Tooltip>
                    {!confirmDelete && (
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger>
                                <motion.button layoutId={`delete-btn-group-${group.id}`} type="button" onClick={() => setConfirmDelete(true)} className="rounded-[1rem] p-2 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive">
                                    <Trash2 className="size-4" />
                                </motion.button>
                            </TooltipTrigger>
                            <TooltipContent>{t('detail.actions.delete')}</TooltipContent>
                        </Tooltip>
                    )}
                </div>

                <AnimatePresence>
                    {confirmDelete && (
                        <motion.div layoutId={`delete-btn-group-${group.id}`} className="absolute inset-0 flex items-center justify-center gap-2 rounded-[1.7rem] bg-destructive p-2" transition={{ type: 'spring', stiffness: 400, damping: 30 }}>
                            <button type="button" onClick={() => setConfirmDelete(false)} className="flex h-7 w-7 items-center justify-center rounded-lg bg-destructive-foreground/20 text-destructive-foreground transition-all hover:bg-destructive-foreground/30 active:scale-95">
                                <X className="size-4" />
                            </button>
                            <button type="button" onClick={() => group.id && deleteGroup.mutate(group.id, { onSuccess: () => toast.success(t('toast.deleted')) })} disabled={deleteGroup.isPending} className="flex-1 h-7 flex items-center justify-center gap-2 rounded-lg bg-destructive-foreground text-destructive text-sm font-semibold transition-all hover:bg-destructive-foreground/90 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed">
                                <Trash2 className="size-3.5" />
                                {t('detail.actions.confirmDelete')}
                            </button>
                        </motion.div>
                    )}
                </AnimatePresence>
                </div>

                {(() => {
                    const modelNames = (group.items || []).map((item) => item.model_name);
                    const capabilities = inferGroupCapabilities(modelNames);
                    const modelCount = modelNames.length;
                    return (
                        <div className="mt-4 flex flex-wrap items-center gap-2">
                            {capabilities.map((cap) => (
                                <span
                                    key={cap}
                                    className={cn(
                                        'inline-flex items-center rounded-full px-2.5 py-1 text-[10px] font-medium shadow-waterhouse-soft',
                                        CAPABILITY_COLORS[cap]
                                    )}
                                >
                                    {t(CAPABILITY_LABEL_KEYS[cap])}
                                </span>
                            ))}
                            <span className="ml-auto inline-flex items-center gap-1 rounded-full border border-border/20 bg-background/48 px-2.5 py-1 text-[10px] text-muted-foreground shadow-waterhouse-soft">
                                <Layers className="size-3" />
                                {t('card.modelCount', { count: modelCount })}
                            </span>
                        </div>
                    );
                })()}
            </header>

            <section className="relative mb-4 rounded-[1.7rem] border border-border/25 bg-background/34 p-3 shadow-none md:shadow-waterhouse-soft">
                <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/44 px-2.5 py-1 text-[0.64rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground shadow-waterhouse-soft">
                    <Waves className="size-3.5" />
                    {t(`mode.${MODE_LABELS[group.mode]}`)}
                </div>
                <div className="grid grid-cols-2 gap-2 md:grid-cols-5">
                {([GroupMode.RoundRobin, GroupMode.Random, GroupMode.Failover, GroupMode.Weighted, GroupMode.Auto] as const).map((m) => (
                    <button
                        key={m}
                        type="button"
                        aria-disabled={isUpdatingMode || !group.id}
                        onClick={() => {
                            if (isUpdatingMode || !group.id) return;
                            if (m === group.mode) return;
                            updateGroup.mutate({ id: group.id!, mode: m }, { onSuccess, onError });
                        }}
                        className={cn(
                            'rounded-[1rem] px-3 py-2 text-xs font-medium transition-[transform,border-color,background-color,box-shadow] duration-300',
                            group.mode === m
                                ? 'border border-primary/20 bg-primary text-primary-foreground shadow-waterhouse-soft'
                                : 'border border-border/25 bg-background/48 text-foreground shadow-waterhouse-soft hover:-translate-y-0.5 hover:border-primary/16 hover:bg-background/64',
                            // Keep visuals stable (no opacity/disabled flicker) while still preventing double-submit via onClick guard.
                            (!group.id) && 'cursor-not-allowed opacity-50'
                        )}
                    >
                        {t(`mode.${MODE_LABELS[m]}`)}
                    </button>
                ))}
                </div>
            </section>

            <section className="relative min-h-[25.25rem] overflow-hidden rounded-[1.8rem] border border-border/25 bg-background/32 shadow-none md:shadow-waterhouse-soft">
                <MemberList
                    members={members}
                    onReorder={setMembers}
                    onRemove={handleRemoveMember}
                    onWeightChange={handleWeightChange}
                    onDragStart={handleDragStart}
                    onDrop={handleDropReorder}
                    onDragFinish={handleDragFinish}
                    autoScrollOnAdd={false}
                    showWeight={group.mode === GroupMode.Weighted || group.mode === GroupMode.Auto}
                    layoutScope={`card-${group.id ?? 'unknown'}`}
                />
            </section>

            {(isTesting || resultByItemId.size > 0) && (
                <section className="mt-4 rounded-[1.7rem] border border-border/25 bg-background/40 p-4 shadow-none md:shadow-waterhouse-soft">
                    <div className="flex items-center justify-between gap-3">
                        <div className="space-y-1">
                            <div className="inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/44 px-2.5 py-1 text-[0.64rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground shadow-waterhouse-soft">
                                <TestTubeDiagonal className="size-3.5" />
                                {t('card.testProgressTitle')}
                            </div>
                            <p className="text-xs text-muted-foreground">
                                {t('card.testProgressCount', { completed: completedCount, total: totalCount })}
                            </p>
                        </div>
                        {isTesting && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
                    </div>
                    <Progress value={progressValue} className="mt-3 h-2" />
                    <ul className="mt-3 space-y-2">
                        {displayMembers.map((member) => {
                            const result = member.item_id !== undefined ? resultByItemId.get(member.item_id) : undefined;
                            const status = !result ? 'pending' : result.passed ? 'passed' : 'failed';

                            return (
                                <li key={`test-status-${member.id}`} className="flex items-center justify-between gap-3 rounded-[1.2rem] border border-border/25 bg-background/44 px-3 py-2.5 shadow-waterhouse-soft">
                                    <div className="min-w-0">
                                        <div className="truncate text-sm font-medium text-foreground">{member.name}</div>
                                        <div className="truncate text-xs text-muted-foreground">{member.channel_name}</div>
                                    </div>
                                    <div className="flex items-center gap-2 text-xs">
                                        {status === 'pending' && (
                                            <>
                                                <Clock3 className="size-3.5 text-muted-foreground" />
                                                <span className="text-muted-foreground">{t('card.testPending')}</span>
                                            </>
                                        )}
                                        {status === 'passed' && (
                                            <>
                                                <CircleCheck className="size-3.5 text-emerald-500" />
                                                <span className="text-emerald-600 dark:text-emerald-400">{t('card.testPassed')}</span>
                                            </>
                                        )}
                                        {status === 'failed' && (
                                            <>
                                                <CircleX className="size-3.5 text-destructive" />
                                                <span className="max-w-48 truncate text-destructive" title={result?.message || t('card.testUnknownError')}>
                                                    {result?.message || t('card.testUnknownError')}
                                                </span>
                                            </>
                                        )}
                                    </div>
                                </li>
                            );
                        })}
                    </ul>
                </section>
            )}

            {failedTestResults.length > 0 && (
                <section className="mt-4 rounded-[1.7rem] border border-amber-500/28 bg-amber-500/6 p-4 shadow-waterhouse-soft">
                    <div className="flex items-start justify-between gap-3">
                        <div className="text-sm font-medium text-amber-700 dark:text-amber-300">
                            {t('card.testFailedTitle')}
                        </div>
                        <button
                            type="button"
                            onClick={handleRemoveFailedMembers}
                            disabled={updateGroup.isPending}
                            className="shrink-0 rounded-lg bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-500/25 disabled:cursor-not-allowed disabled:opacity-50 dark:text-amber-300"
                        >
                            {t('detail.actions.removeFailedModels')}
                        </button>
                    </div>
                    <ul className="mt-2 space-y-1 text-xs text-muted-foreground">
                        {failedTestResults.map((result) => (
                                <li key={`${result.item_id}-${result.channel_id}-${result.model_name}`}>
                                    {result.model_name} @ {result.channel_name} · {t('card.testAttempts', { count: result.attempts })} · {result.message || t('card.testUnknownError')}
                                </li>
                            ))}
                    </ul>
                    <p className="mt-2 text-xs text-amber-700/90 dark:text-amber-300/90">
                        {t('card.testRemovableHint')}
                    </p>
                </section>
            )}
        </article >
    );
}
