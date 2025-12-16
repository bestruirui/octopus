'use client';

import { useState, useMemo } from 'react';
import { GripVertical, X, Trash2 } from 'lucide-react';
import { Reorder, useDragControls, motion, AnimatePresence } from 'motion/react';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { type LLMChannel } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';

export interface SelectedMember extends LLMChannel {
    id: string;
    item_id?: number;
    weight?: number;
}

// 可拖拽成员项
export function MemberItem({
    member,
    onRemove,
    onWeightChange,
    onDragStart,
    onDragEnd,
    isRemoving,
    index,
    editable = true,
    showWeight = false,
}: {
    member: SelectedMember;
    onRemove: (id: string) => void;
    onWeightChange?: (id: string, weight: number) => void;
    onDragStart?: () => void;
    onDragEnd?: () => void;
    isRemoving?: boolean;
    index: number;
    editable?: boolean;
    showWeight?: boolean;
}) {
    const controls = useDragControls();
    const { Avatar: ModelAvatar } = getModelIcon(member.name);
    const [confirmDelete, setConfirmDelete] = useState(false);

    return (
        <Reorder.Item
            value={member}
            dragListener={false}
            dragControls={controls}
            onDragStart={onDragStart}
            onDragEnd={onDragEnd}
            className={cn('grid transition-[grid-template-rows] duration-200', isRemoving ? 'grid-rows-[0fr]' : 'grid-rows-[1fr]')}
            whileDrag={editable ? { scale: 1.02, boxShadow: '0 8px 32px rgba(0,0,0,0.15)', zIndex: 50 } : undefined}
        >
            <div className="overflow-hidden">
                <div className={cn(
                    'flex items-center gap-2 rounded-lg bg-background border border-border/50 px-2.5 py-2 select-none transition-opacity duration-200 relative overflow-hidden',
                    isRemoving && 'opacity-0'
                )}>
                    <span className="size-5 rounded-md bg-primary/10 text-primary text-xs font-bold grid place-items-center shrink-0">
                        {index + 1}
                    </span>

                    <div
                        className={cn('p-0.5 rounded transition-colors', editable && 'cursor-grab active:cursor-grabbing hover:bg-muted touch-none')}
                        onPointerDown={editable ? (e) => controls.start(e) : undefined}
                    >
                        <GripVertical className={cn('size-3.5 transition-colors', editable ? 'text-muted-foreground' : 'text-muted-foreground/30')} />
                    </div>

                    <ModelAvatar size={18} />

                    <div className="flex flex-col min-w-0 flex-1">
                        <span className="text-sm font-medium truncate leading-tight">{member.name}</span>
                        <span className="text-[10px] text-muted-foreground truncate leading-tight">{member.channel_name}</span>
                    </div>

                    {showWeight && editable && (
                        <input
                            type="number"
                            min={1}
                            value={member.weight ?? 1}
                            onChange={(e) => onWeightChange?.(member.id, Math.max(1, parseInt(e.target.value) || 1))}
                            className="w-12 h-6 text-xs text-center rounded border border-border bg-muted/50 focus:outline-none focus:ring-1 focus:ring-primary"
                        />
                    )}

                    {!confirmDelete && (
                        <motion.button
                            layoutId={`delete-btn-member-${member.id}`}
                            type="button"
                            onClick={() => setConfirmDelete(true)}
                            className="p-1 rounded hover:bg-destructive/10 hover:text-destructive transition-colors"
                            initial={false}
                            animate={{ opacity: editable ? 1 : 0, x: editable ? 0 : 8 }}
                            transition={{ duration: 0.15 }}
                            style={{ pointerEvents: editable ? 'auto' : 'none' }}
                        >
                            <X className="size-3" />
                        </motion.button>
                    )}

                    <AnimatePresence>
                        {confirmDelete && (
                            <motion.div
                                layoutId={`delete-btn-member-${member.id}`}
                                className="absolute inset-0 flex items-center justify-center gap-2 bg-destructive p-1.5 rounded-lg"
                                transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                            >
                                <button
                                    type="button"
                                    onClick={() => setConfirmDelete(false)}
                                    className="flex h-6 w-6 items-center justify-center rounded-md bg-destructive-foreground/20 text-destructive-foreground transition-all hover:bg-destructive-foreground/30 active:scale-95"
                                >
                                    <X className="h-3 w-3" />
                                </button>
                                <button
                                    type="button"
                                    onClick={() => onRemove(member.id)}
                                    className="flex-1 h-6 flex items-center justify-center gap-1.5 rounded-md bg-destructive-foreground text-destructive text-xs font-semibold transition-all hover:bg-destructive-foreground/90 active:scale-[0.98]"
                                >
                                    <Trash2 className="h-3 w-3" />
                                </button>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>
            </div>
        </Reorder.Item>
    );
}

// 添加成员行
export function AddMemberRow({
    index,
    channels,
    modelChannels,
    selectedMembers,
    onConfirm,
    onCancel,
}: {
    index: number;
    channels: { id: number; name: string }[];
    modelChannels: LLMChannel[];
    selectedMembers: SelectedMember[];
    onConfirm: (channel: LLMChannel) => void;
    onCancel: () => void;
}) {
    const t = useTranslations('group');
    const [channelId, setChannelId] = useState('');

    // 过滤掉所有模型都已被选择的渠道
    const availableChannels = useMemo(() => {
        return channels.filter((c) => {
            const channelModels = modelChannels.filter((mc) => mc.channel_id === c.id);
            // 检查该渠道是否还有未被选择的模型
            return channelModels.some((m) => !selectedMembers.some((s) => s.channel_id === m.channel_id && s.name === m.name));
        });
    }, [channels, modelChannels, selectedMembers]);

    const models = useMemo(() => {
        if (!channelId) return [];
        return modelChannels.filter((mc) => mc.channel_id === +channelId);
    }, [modelChannels, channelId]);

    // 选择模型时直接确认
    const handleModelSelect = (modelName: string) => {
        const channel = modelChannels.find((mc) => mc.channel_id === +channelId && mc.name === modelName);
        if (channel) {
            onConfirm(channel);
        }
    };

    return (
        <div className="flex items-center gap-2 rounded-lg bg-background border-2 border-dashed border-primary/30 px-2.5 py-2">
            <span className="size-5 rounded-md bg-primary/10 text-primary text-xs font-bold grid place-items-center shrink-0">
                {index + 1}
            </span>
            <div className="p-0.5">
                <GripVertical className="size-3.5 text-muted-foreground/30" />
            </div>

            <Select value={channelId} onValueChange={setChannelId}>
                <SelectTrigger className="flex-1 h-7 rounded-md text-xs min-w-0" size="sm">
                    <SelectValue placeholder={t('form.selectChannel')} />
                </SelectTrigger>
                <SelectContent>
                    {availableChannels.map((c) => (
                        <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>
                    ))}
                </SelectContent>
            </Select>

            <Select value="" onValueChange={handleModelSelect} disabled={!channelId}>
                <SelectTrigger className="flex-1 h-7 rounded-md text-xs min-w-0 [&_svg]:text-inherit!" size="sm">
                    <SelectValue placeholder={t('form.selectModel')} />
                </SelectTrigger>
                <SelectContent>
                    {models
                        .filter((m) => !selectedMembers.some((s) => s.channel_id === m.channel_id && s.name === m.name))
                        .map((m) => {
                            const { Avatar } = getModelIcon(m.name);
                            return (
                                <SelectItem key={m.name} value={m.name} className="[&_svg]:text-inherit!">
                                    <span className="flex items-center gap-2"><Avatar size={14} />{m.name}</span>
                                </SelectItem>
                            );
                        })}
                </SelectContent>
            </Select>

            <button
                type="button"
                onClick={onCancel}
                className="size-6 rounded-md grid place-items-center shrink-0 hover:bg-destructive/10 hover:text-destructive text-muted-foreground transition-colors"
            >
                <X className="size-3.5" />
            </button>
        </div>
    );
}
