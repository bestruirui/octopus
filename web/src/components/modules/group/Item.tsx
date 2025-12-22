'use client';

import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Trash2, Layers, X, Plus, Check, Copy, Loader2, Wand2 } from 'lucide-react';
import { Reorder, motion, AnimatePresence } from 'motion/react';
import { type Group, useAutoAddGroupItem, useDeleteGroup, useUpdateGroup } from '@/api/endpoints/group';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { MemberItem, AddMemberRow, type SelectedMember } from './components';
import { buildChannelNameByModelKey, modelChannelKey } from './utils';

export function GroupCard({ group }: { group: Group }) {
    const t = useTranslations('group');
    const updateGroup = useUpdateGroup();
    const deleteGroup = useDeleteGroup();
    const autoAddGroupItem = useAutoAddGroupItem();
    const { data: modelChannels = [] } = useModelChannelList();

    const [members, setMembers] = useState<SelectedMember[]>([]);
    const [isAdding, setIsAdding] = useState(false);
    const [confirmDelete, setConfirmDelete] = useState(false);
    const [copied, setCopied] = useState(false);
    const isDragging = useRef(false);
    const weightTimerRef = useRef<NodeJS.Timeout | null>(null);
    const membersRef = useRef<SelectedMember[]>([]);

    const channelNameByKey = useMemo(() => buildChannelNameByModelKey(modelChannels), [modelChannels]);

    const displayMembers = useMemo(() =>
        [...(group.items || [])]
            .sort((a, b) => a.priority - b.priority)
            .map((item) => ({
                id: `${item.channel_id}-${item.model_name}-${item.id || 0}`,
                name: item.model_name,
                channel_id: item.channel_id,
                channel_name: channelNameByKey.get(modelChannelKey(item.channel_id, item.model_name)) ?? `Channel ${item.channel_id}`,
                item_id: item.id,
                weight: item.weight,
            })),
        [group.items, channelNameByKey]
    );

    useEffect(() => {
        if (!isDragging.current) setMembers([...displayMembers]);
    }, [displayMembers]);

    useEffect(() => {
        membersRef.current = members;
    }, [members]);

    useEffect(() => {
        return () => { if (weightTimerRef.current) clearTimeout(weightTimerRef.current); };
    }, []);

    const isEmpty = members.length === 0 && !isAdding;
    const onSuccess = useCallback(() => toast.success(t('toast.updated')), [t]);

    const priorityByItemId = useMemo(() => {
        const map = new Map<number, number>();
        (group.items || []).forEach((item) => {
            if (item.id !== undefined) map.set(item.id, item.priority);
        });
        return map;
    }, [group.items]);

    const handleAddMember = useCallback((channel: LLMChannel) => {
        setMembers((prev) => [...prev, { ...channel, id: `${channel.channel_id}-${channel.name}-${Date.now()}`, weight: 1 }]);
        setIsAdding(false);
        updateGroup.mutate({ id: group.id!, items_to_add: [{ channel_id: channel.channel_id, model_name: channel.name, priority: (group.items?.length ?? 0) + 1, weight: 1 }] }, { onSuccess });
    }, [group.id, group.items?.length, updateGroup, onSuccess]);

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
                { onSuccess }
            );
        }, 500);
    }, [group.id, priorityByItemId, updateGroup, onSuccess]);

    const handleRemoveMember = useCallback((id: string) => {
        const member = members.find((m) => m.id === id);
        if (member?.item_id !== undefined) updateGroup.mutate({ id: group.id!, items_to_delete: [member.item_id] }, { onSuccess });
    }, [members, group.id, updateGroup, onSuccess]);

    const handleNameChange = useCallback((name: string) => {
        if (name && name !== group.name) updateGroup.mutate({ id: group.id!, name }, { onSuccess });
    }, [group.id, group.name, updateGroup, onSuccess]);

    const handleDragStart = useCallback(() => { isDragging.current = true; }, []);

    const handleDragEnd = useCallback(() => {
        isDragging.current = false;
        const itemsToUpdate = members
            .map((m, i) => ({ member: m, newPriority: i + 1 }))
            .filter(({ member, newPriority }) => {
                if (!member.item_id) return false;
                const origPriority = priorityByItemId.get(member.item_id);
                return origPriority !== undefined && origPriority !== newPriority;
            })
            .map(({ member, newPriority }) => ({ id: member.item_id!, priority: newPriority, weight: member.weight ?? 1 }));
        if (itemsToUpdate.length > 0) updateGroup.mutate({ id: group.id!, items_to_update: itemsToUpdate }, { onSuccess });
    }, [members, group.id, priorityByItemId, updateGroup, onSuccess]);

    const handleCopy = async () => {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(group.name);
            } else {
                const textArea = document.createElement('textarea');
                textArea.value = group.name;
                textArea.style.position = 'fixed';
                textArea.style.left = '-9999px';
                document.body.appendChild(textArea);
                textArea.select();
                document.execCommand('copy');
                document.body.removeChild(textArea);
            }
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    };

    return (
        <article className="flex flex-col rounded-3xl border border-border bg-card text-card-foreground p-4 custom-shadow">
            <header className="flex items-start justify-between mb-3 relative overflow-visible rounded-xl -mx-1 px-1 -my-1 py-1">
                <div className="relative flex-1 mr-2 min-w-0 group/title">
                    {/* Placeholder to maintain layout */}
                    <span className="text-lg font-bold truncate block invisible" aria-hidden="true">{group.name}</span>
                    {/* Actual editable title with hover expansion */}
                    <h3
                        contentEditable
                        suppressContentEditableWarning
                        onBlur={(e) => handleNameChange(e.currentTarget.textContent || '')}
                        className={cn(
                            "text-lg font-bold outline-none truncate absolute inset-0",
                            "transition-all duration-200 ease-out origin-left",
                            "group-hover/title:whitespace-normal group-hover/title:wrap-break-word group-hover/title:overflow-visible group-hover/title:bottom-auto",
                            "group-hover/title:bg-popover group-hover/title:backdrop-blur-md group-hover/title:shadow-xl group-hover/title:rounded-2xl",
                            "group-hover/title:px-3 group-hover/title:py-2 group-hover/title:-mx-3 group-hover/title:-my-2",
                            "group-hover/title:z-50 group-hover/title:scale-105 group-hover/title:border group-hover/title:border-border/50",
                            "focus:whitespace-normal focus:wrap-break-word focus:overflow-visible focus:bottom-auto",
                            "focus:bg-popover focus:backdrop-blur-md focus:shadow-xl focus:rounded-2xl",
                            "focus:px-3 focus:py-2 focus:-mx-3 focus:-my-2",
                            "focus:z-50 focus:scale-105 focus:border focus:border-primary/50"
                        )}
                    >
                        {group.name}
                    </h3>
                </div>

                <div className="flex items-center gap-1 shrink-0">
                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger>
                            <button
                                type="button"
                                onClick={() => setIsAdding(true)}
                                disabled={isAdding}
                                className={cn(
                                    'p-1.5 rounded-lg transition-colors',
                                    isAdding ? 'bg-primary/10 text-primary' : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                                )}
                            >
                                <Plus className="size-4" />
                            </button>
                        </TooltipTrigger>
                        <TooltipContent>{t('form.addItem')}</TooltipContent>
                    </Tooltip>

                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger>
                            <button
                                type="button"
                                onClick={() => group.id && autoAddGroupItem.mutate(group.id, {
                                    onSuccess,
                                    onError: (error: any) => toast.error(t('toast.autoAddFailed'), { description: error?.message }),
                                })}
                                disabled={isAdding || autoAddGroupItem.isPending}
                                className={cn(
                                    'p-1.5 rounded-lg transition-colors',
                                    autoAddGroupItem.isPending
                                        ? 'bg-primary/10 text-primary'
                                        : 'hover:bg-muted text-muted-foreground hover:text-foreground',
                                    (isAdding || autoAddGroupItem.isPending) && 'disabled:opacity-50 disabled:cursor-not-allowed'
                                )}
                            >
                                {autoAddGroupItem.isPending ? (
                                    <Loader2 className="size-4 animate-spin" />
                                ) : (
                                    <Wand2 className="size-4" />
                                )}
                            </button>
                        </TooltipTrigger>
                        <TooltipContent>{t('form.autoAdd')}</TooltipContent>
                    </Tooltip>

                    {updateGroup.isPending ? (
                        <div className="p-1.5 text-primary"><Loader2 className="size-4 animate-spin" /></div>
                    ) : (
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger>
                                <button type="button" onClick={handleCopy} className="p-1.5 rounded-lg transition-colors hover:bg-muted text-muted-foreground hover:text-foreground">
                                    <AnimatePresence mode="wait">
                                        <motion.div key={copied ? 'check' : 'copy'} initial={{ scale: 0 }} animate={{ scale: 1 }} exit={{ scale: 0 }}>
                                            {copied ? <Check className="size-4 text-primary" /> : <Copy className="size-4" />}
                                        </motion.div>
                                    </AnimatePresence>
                                </button>
                            </TooltipTrigger>
                            <TooltipContent>{t('detail.actions.copyName')}</TooltipContent>
                        </Tooltip>
                    )}
                    {!confirmDelete && (
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger>
                                <motion.button layoutId={`delete-btn-group-${group.id}`} type="button" onClick={() => setConfirmDelete(true)} className="p-1.5 rounded-lg hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors">
                                    <Trash2 className="size-4" />
                                </motion.button>
                            </TooltipTrigger>
                            <TooltipContent>{t('detail.actions.delete')}</TooltipContent>
                        </Tooltip>
                    )}
                </div>

                <AnimatePresence>
                    {confirmDelete && (
                        <motion.div layoutId={`delete-btn-group-${group.id}`} className="absolute inset-0 flex items-center justify-center gap-2 bg-destructive p-2 rounded-xl" transition={{ type: 'spring', stiffness: 400, damping: 30 }}>
                            <button type="button" onClick={() => setConfirmDelete(false)} className="flex h-7 w-7 items-center justify-center rounded-lg bg-destructive-foreground/20 text-destructive-foreground transition-all hover:bg-destructive-foreground/30 active:scale-95">
                                <X className="h-4 w-4" />
                            </button>
                            <button type="button" onClick={() => group.id && deleteGroup.mutate(group.id, { onSuccess: () => toast.success(t('toast.deleted')) })} disabled={deleteGroup.isPending} className="flex-1 h-7 flex items-center justify-center gap-2 rounded-lg bg-destructive-foreground text-destructive text-sm font-semibold transition-all hover:bg-destructive-foreground/90 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed">
                                <Trash2 className="h-3.5 w-3.5" />
                                {t('detail.actions.confirmDelete')}
                            </button>
                        </motion.div>
                    )}
                </AnimatePresence>
            </header>

            <div className="flex gap-1 mb-3">
                {([1, 2, 3, 4] as const).map((m) => (
                    <button key={m} type="button" onClick={() => m !== group.mode && updateGroup.mutate({ id: group.id!, mode: m }, { onSuccess })} className={cn('flex-1 py-1 text-xs rounded-lg transition-colors', group.mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80')}>
                        {t(`mode.${m === 1 ? 'roundRobin' : m === 2 ? 'random' : m === 3 ? 'failover' : 'weighted'}`)}
                    </button>
                ))}
            </div>

            {/* Member list: fixed height with internal scroll */}
            <section className="h-96 rounded-xl border border-border/50 bg-muted/30 overflow-hidden relative">
                <div className={cn('absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground transition-opacity duration-200', isEmpty ? 'opacity-100' : 'opacity-0 pointer-events-none')}>
                    <Layers className="size-8 opacity-40" />
                    <span className="text-xs">{t('card.empty')}</span>
                </div>

                <div className={cn('h-full overflow-y-auto transition-opacity duration-200', isEmpty && 'opacity-0')}>
                    <div className="p-2 flex flex-col gap-1.5">
                        <Reorder.Group axis="y" values={members} onReorder={setMembers} className="flex flex-col gap-1.5">
                            {members.map((m, i) => (
                                <MemberItem key={m.id} member={m} onRemove={handleRemoveMember} onWeightChange={handleWeightChange} onDragStart={handleDragStart} onDragEnd={handleDragEnd} index={i} editable showWeight={group.mode === 4} />
                            ))}
                        </Reorder.Group>
                        {isAdding && <AddMemberRow index={members.length} selectedMembers={members} onConfirm={handleAddMember} onCancel={() => setIsAdding(false)} />}
                    </div>
                </div>
            </section>
        </article>
    );
}
