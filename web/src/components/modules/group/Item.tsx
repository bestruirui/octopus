'use client';

import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { Trash2, Layers, X, Plus, Check, Copy, Loader2 } from 'lucide-react';
import { Reorder, motion, AnimatePresence } from 'motion/react';
import { type Group, type GroupUpdateRequest, useDeleteGroup, useUpdateGroup } from '@/api/endpoints/group';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { toast } from '@/components/common/Toast';
import { MemberItem, AddMemberRow, type SelectedMember } from './components';

export function GroupCard({ group }: { group: Group }) {
    const t = useTranslations('group');
    const updateGroup = useUpdateGroup();
    const deleteGroup = useDeleteGroup();
    const { data: modelChannels = [] } = useModelChannelList();

    const [editName, setEditName] = useState(group.name);
    const [editMembers, setEditMembers] = useState<SelectedMember[]>([]);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());
    const [isAdding, setIsAdding] = useState(false);
    const [confirmDelete, setConfirmDelete] = useState(false);
    const [copied, setCopied] = useState(false);

    // 用于防止初始化时触发自动保存
    const isInitialMount = useRef(true);
    const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const isDragging = useRef(false);

    // 合并计算渠道映射和列表
    const { channelMap, channels } = useMemo(() => {
        const map = new Map<number, string>();
        const list: { id: number; name: string }[] = [];
        modelChannels.forEach((mc) => {
            if (!map.has(mc.channel_id)) {
                map.set(mc.channel_id, mc.channel_name);
                list.push({ id: mc.channel_id, name: mc.channel_name });
            }
        });
        return { channelMap: map, channels: list };
    }, [modelChannels]);

    // 展示用成员列表
    const displayMembers = useMemo(() =>
        [...(group.items || [])]
            .sort((a, b) => a.priority - b.priority)
            .map((item) => ({
                id: `${item.channel_id}-${item.model_name}-${item.id || 0}`,
                name: item.model_name,
                channel_id: item.channel_id,
                channel_name: channelMap.get(item.channel_id) || `Channel ${item.channel_id}`,
                item_id: item.id,
                weight: item.weight,
            })),
        [group.items, channelMap]
    );

    // 同步 group 数据到编辑状态
    useEffect(() => {
        isInitialMount.current = true;
        setEditName(group.name);
        setEditMembers([...displayMembers]);
    }, [group.name, displayMembers]);

    // 计算派生状态
    const { hasChanges, isValid, isEmpty } = useMemo(() => {
        const changed = editName !== group.name ||
            editMembers.length !== displayMembers.length ||
            editMembers.some((m, i) => m.id !== displayMembers[i]?.id || m.weight !== displayMembers[i]?.weight);
        const valid = !!editName.trim() && editMembers.length > 0;
        const visibleCount = editMembers.filter((m) => !removingIds.has(m.id)).length;
        return { hasChanges: changed, isValid: valid, isEmpty: visibleCount === 0 && !isAdding };
    }, [editName, editMembers, group.name, displayMembers, removingIds, isAdding]);

    const handleAddMember = (channel: LLMChannel) => {
        setEditMembers((prev) => [...prev, { ...channel, id: `${channel.channel_id}-${channel.name}-${Date.now()}`, weight: 1 }]);
        setIsAdding(false);
    };

    const handleWeightChange = useCallback((id: string, weight: number) => {
        setEditMembers((prev) => prev.map((m) => m.id === id ? { ...m, weight } : m));
    }, []);

    const handleRemoveMember = useCallback((id: string) => {
        setRemovingIds((prev) => new Set(prev).add(id));
        setTimeout(() => {
            setEditMembers((prev) => prev.filter((m) => m.id !== id));
            setRemovingIds((prev) => { const n = new Set(prev); n.delete(id); return n; });
        }, 200);
    }, []);

    const handleSave = useCallback(() => {
        const originalItems = new Map((group.items || []).map((item) => [item.id, item]));
        const editItemIds = new Set(editMembers.filter((m) => m.item_id !== undefined).map((m) => m.item_id));

        const req: GroupUpdateRequest = { id: group.id! };
        if (editName !== group.name) req.name = editName;

        const itemsToDelete = [...originalItems.keys()].filter((id) => id !== undefined && !editItemIds.has(id)) as number[];
        if (itemsToDelete.length > 0) req.items_to_delete = itemsToDelete;

        const itemsToAdd = editMembers
            .map((m, i) => ({ m, priority: i + 1 }))
            .filter(({ m }) => m.item_id === undefined)
            .map(({ m, priority }) => ({ channel_id: m.channel_id, model_name: m.name, priority, weight: m.weight ?? 1 }));
        if (itemsToAdd.length > 0) req.items_to_add = itemsToAdd;

        const itemsToUpdate = editMembers
            .map((m, i) => ({ m, priority: i + 1 }))
            .filter(({ m, priority }) => {
                const orig = originalItems.get(m.item_id);
                return m.item_id !== undefined && (orig?.priority !== priority || orig?.weight !== m.weight);
            })
            .map(({ m, priority }) => ({ id: m.item_id!, priority, weight: m.weight ?? 1 }));
        if (itemsToUpdate.length > 0) req.items_to_update = itemsToUpdate;

        updateGroup.mutate(req, { onSuccess: () => toast.success(t('toast.updated')) });
    }, [group.id, group.name, group.items, editName, editMembers, updateGroup, t]);

    // 自动保存（防抖 500ms），拖拽中不触发
    useEffect(() => {
        // 跳过初始挂载
        if (isInitialMount.current) {
            isInitialMount.current = false;
            return;
        }

        // 拖拽中不触发自动保存
        if (isDragging.current) return;

        // 清除之前的定时器
        if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);

        // 只在有变更且数据有效时保存
        if (hasChanges && isValid && !updateGroup.isPending) {
            saveTimeoutRef.current = setTimeout(handleSave, 500);
        }

        return () => {
            if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
        };
    }, [editName, editMembers, handleSave, updateGroup.isPending]);

    // 拖拽开始
    const handleDragStart = useCallback(() => {
        isDragging.current = true;
        // 清除待保存的定时器
        if (saveTimeoutRef.current) {
            clearTimeout(saveTimeoutRef.current);
            saveTimeoutRef.current = null;
        }
    }, []);

    // 拖拽结束，触发保存
    const handleDragEnd = useCallback(() => {
        isDragging.current = false;
        // 延迟一帧确保状态已更新后保存
        requestAnimationFrame(() => {
            if (!updateGroup.isPending) handleSave();
        });
    }, [updateGroup.isPending, handleSave]);

    const scaleAnim = { initial: { scale: 0 }, animate: { scale: 1 }, exit: { scale: 0 } };

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
            <header className="flex items-start justify-between mb-3 relative overflow-hidden rounded-xl -mx-1 px-1 -my-1 py-1">
                <h3
                    contentEditable
                    suppressContentEditableWarning
                    onBlur={(e) => setEditName(e.currentTarget.textContent || '')}
                    className="text-lg font-bold truncate flex-1 mr-2 outline-none"
                >
                    {editName}
                </h3>

                <div className="flex items-center gap-1 shrink-0">
                    <button type="button" onClick={() => setIsAdding(true)} disabled={isAdding} className={cn('p-1.5 rounded-lg transition-colors', isAdding ? 'bg-primary/10 text-primary' : 'hover:bg-muted text-muted-foreground hover:text-foreground')}>
                        <Plus className="size-4" />
                    </button>
                    {updateGroup.isPending ? (
                        <div className="p-1.5 text-primary">
                            <Loader2 className="size-4 animate-spin" />
                        </div>
                    ) : (
                        <button
                            type="button"
                            onClick={handleCopy}
                            className="p-1.5 rounded-lg transition-colors hover:bg-muted text-muted-foreground hover:text-foreground"
                        >
                            <AnimatePresence mode="wait">
                                <motion.div key={copied ? 'check' : 'copy'} {...scaleAnim}>
                                    {copied ? <Check className="size-4 text-primary" /> : <Copy className="size-4" />}
                                </motion.div>
                            </AnimatePresence>
                        </button>
                    )}
                    {!confirmDelete && (
                        <motion.button
                            layoutId={`delete-btn-group-${group.id}`}
                            type="button"
                            onClick={() => setConfirmDelete(true)}
                            className="p-1.5 rounded-lg hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                        >
                            <Trash2 className="size-4" />
                        </motion.button>
                    )}
                </div>

                <AnimatePresence>
                    {confirmDelete && (
                        <motion.div
                            layoutId={`delete-btn-group-${group.id}`}
                            className="absolute inset-0 flex items-center justify-center gap-2 bg-destructive p-2 rounded-xl"
                            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                        >
                            <button
                                type="button"
                                onClick={() => setConfirmDelete(false)}
                                className="flex h-7 w-7 items-center justify-center rounded-lg bg-destructive-foreground/20 text-destructive-foreground transition-all hover:bg-destructive-foreground/30 active:scale-95"
                            >
                                <X className="h-4 w-4" />
                            </button>
                            <button
                                type="button"
                                onClick={() => group.id && deleteGroup.mutate(group.id, { onSuccess: () => toast.success(t('toast.deleted')) })}
                                disabled={deleteGroup.isPending}
                                className="flex-1 h-7 flex items-center justify-center gap-2 rounded-lg bg-destructive-foreground text-destructive text-sm font-semibold transition-all hover:bg-destructive-foreground/90 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                <Trash2 className="h-3.5 w-3.5" />
                                {t('detail.actions.confirmDelete')}
                            </button>
                        </motion.div>
                    )}
                </AnimatePresence>
            </header>

            {/* Mode Selection */}
            <div className="flex gap-1 mb-3">
                {([1, 2, 3, 4] as const).map((m) => (
                    <button
                        key={m}
                        type="button"
                        onClick={() => m !== group.mode && updateGroup.mutate({ id: group.id!, mode: m }, { onSuccess: () => toast.success(t('toast.updated')) })}
                        className={cn(
                            'flex-1 py-1 text-xs rounded-lg transition-colors',
                            group.mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'
                        )}
                    >
                        {t(`mode.${m === 1 ? 'roundRobin' : m === 2 ? 'random' : m === 3 ? 'failover' : 'weighted'}`)}
                    </button>
                ))}
            </div>

            <section className={cn('flex-1 rounded-xl border bg-muted/30 overflow-hidden relative min-h-0 transition-colors duration-200', hasChanges ? 'border-primary/50' : 'border-border/50')}>
                <div className={cn('absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground transition-opacity duration-200', isEmpty ? 'opacity-100' : 'opacity-0 pointer-events-none')}>
                    <Layers className="size-8 opacity-40" />
                    <span className="text-xs">{t('card.empty')}</span>
                </div>

                <div className={cn('min-h-100  overflow-y-auto transition-opacity duration-200', isEmpty && 'opacity-0')}>
                    <div className="p-2 flex flex-col gap-1.5">
                        <Reorder.Group axis="y" values={editMembers} onReorder={setEditMembers} className="flex flex-col gap-1.5">
                            {editMembers.map((m, i) => (
                                <MemberItem key={m.id} member={m} onRemove={handleRemoveMember} onWeightChange={handleWeightChange} onDragStart={handleDragStart} onDragEnd={handleDragEnd} isRemoving={removingIds.has(m.id)} index={i} editable showWeight={group.mode === 4} />
                            ))}
                        </Reorder.Group>
                        {isAdding && <AddMemberRow index={editMembers.length} channels={channels} modelChannels={modelChannels} selectedMembers={editMembers} onConfirm={handleAddMember} onCancel={() => setIsAdding(false)} />}
                    </div>
                </div>
            </section>
        </article>
    );
}
