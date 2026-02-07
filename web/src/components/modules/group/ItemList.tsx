'use client';

import { useEffect, useId, useRef, useState, type CSSProperties } from 'react';
import { Layers, GripVertical, X, Trash2, RefreshCcw } from 'lucide-react';
import {
    DragDropContext,
    Draggable,
    Droppable,
    type DraggableProvided,
    type DropResult,
} from '@hello-pangea/dnd';
import { motion, AnimatePresence } from 'motion/react';
import { cn } from '@/lib/utils';
import { getModelIcon } from '@/lib/model-icons';
import type { LLMChannel } from '@/api/endpoints/model';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { useTranslations } from 'next-intl';

export interface SelectedMember extends LLMChannel {
    id: string;
    item_id?: number;
    weight?: number;
    cb_state?: 'CLOSED' | 'OPEN' | 'HALF_OPEN';
    cb_trip_count?: number;
    cb_open_until?: string;
    cb_open_remaining_second?: number;
}

function reorderList<T>(list: T[], startIndex: number, endIndex: number): T[] {
    const result = [...list];
    const [removed] = result.splice(startIndex, 1);
    result.splice(endIndex, 0, removed);
    return result;
}

type MemberItemDnd = {
    innerRef: DraggableProvided['innerRef'];
    draggableProps: DraggableProvided['draggableProps'];
    dragHandleProps: DraggableProvided['dragHandleProps'];
    isDragging: boolean;
};

const cbHoverStyle = {
    '--cb-hover-muted-foreground': 'lab(44.2956% .774592 7.66346)',
    '--cb-hover-bg': 'lab(44.2956% .774592 7.66346 / 0.10)',
} as CSSProperties;

function MemberItem({
    member,
    onRemove,
    onWeightChange,
    isRemoving,
    index,
    showWeight = false,
    showConfirmDelete = true,
    layoutScope,
    dnd,
    showCircuitBreaker = false,
    onCBReset,
}: {
    member: SelectedMember;
    onRemove: (id: string) => void;
    onWeightChange?: (id: string, weight: number) => void;
    isRemoving?: boolean;
    index: number;
    showWeight?: boolean;
    showConfirmDelete?: boolean;
    layoutScope?: string;
    dnd: MemberItemDnd;
    showCircuitBreaker?: boolean;
    onCBReset?: (id: string) => void;
}) {
    const { Avatar: ModelAvatar } = getModelIcon(member.name);
    const t = useTranslations('group');
    const [confirmDelete, setConfirmDelete] = useState(false);
    const isDisabled = member.enabled === false;
    const remainingSeconds = member.cb_open_remaining_second ?? 0;

    return (
        <div
            // DnD libraries provide imperative refs/props; the hook lint rule (`react-hooks/refs`)
            // flags this pattern, but it's safe and required for correct drag behavior.
            // eslint-disable-next-line react-hooks/refs
            ref={dnd.innerRef}
            // eslint-disable-next-line react-hooks/refs
            {...dnd.draggableProps}
            className={cn('rounded-lg grid transition-[grid-template-rows] duration-200', isRemoving ? 'grid-rows-[0fr]' : 'grid-rows-[1fr]')}
            // eslint-disable-next-line react-hooks/refs
            style={{
                /* eslint-disable-next-line react-hooks/refs */
                ...(dnd.draggableProps?.style ?? {}),
                /* eslint-disable-next-line react-hooks/refs */
                ...(dnd.isDragging ? { zIndex: 50, boxShadow: '0 8px 32px rgba(0,0,0,0.15)' } : null),
            }}
        >
            <div className={cn(
                'flex items-center gap-2 rounded-lg bg-background border border-border/50 px-2.5 py-2 select-none transition-opacity duration-200 relative overflow-hidden',
                isRemoving && 'opacity-0',
                isDisabled && 'opacity-60 grayscale'
            )}>
                <span className={cn(
                    'size-5 rounded-md text-xs font-bold grid place-items-center shrink-0',
                    isDisabled ? 'bg-muted text-muted-foreground' : 'bg-primary/10 text-primary'
                )}>
                    {index + 1}
                </span>

                <div
                    className={cn(
                        'p-0.5 rounded touch-none transition-colors',
                        isDisabled
                            ? 'cursor-grab active:cursor-grabbing hover:bg-muted/60'
                            : 'cursor-grab active:cursor-grabbing hover:bg-muted'
                    )}
                    // eslint-disable-next-line react-hooks/refs
                    {...dnd.dragHandleProps}
                >
                    <GripVertical className="size-3.5 text-muted-foreground" />
                </div>

                <span className={cn(isDisabled && 'opacity-70')}>
                    <ModelAvatar size={18} />
                </span>

                <div className="flex flex-col min-w-0 flex-1">
                    <Tooltip side="top" sideOffset={10} align="start">
                        <TooltipTrigger className={cn(
                            'text-sm font-medium truncate leading-tight',
                            isDisabled && 'text-muted-foreground'
                        )}>
                            {member.name}
                        </TooltipTrigger>
                        <TooltipContent key={member.name}>{member.name}</TooltipContent>
                    </Tooltip>
                    <span className="text-[10px] text-muted-foreground truncate leading-tight">{member.channel_name}</span>
                </div>

                {showWeight && (
                    <input
                        type="number"
                        min={1}
                        value={member.weight ?? 1}
                        onChange={(e) => onWeightChange?.(member.id, Math.max(1, parseInt(e.target.value) || 1))}
                        className={cn(
                            'w-12 h-6 text-xs text-center rounded border border-border bg-muted/50 focus:outline-none focus:ring-1 focus:ring-primary',
                            isDisabled && 'text-muted-foreground'
                        )}
                    />
                )}

                {showCircuitBreaker && (
                    <div className="flex items-center gap-[30px] shrink-0 text-[11px]" style={cbHoverStyle}>
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <span className={cn(
                                    'inline-flex w-fit justify-self-start whitespace-nowrap px-1.5 py-0.5 rounded border text-left',
                                    member.cb_state === 'OPEN'
                                        ? 'text-destructive border-destructive/30 bg-destructive/10'
                                        : member.cb_state === 'HALF_OPEN'
                                            ? 'text-amber-600 border-amber-500/30 bg-amber-500/10'
                                            : 'text-emerald-600 border-emerald-500/30 bg-emerald-500/10'
                                )}>
                                    {member.cb_state ?? 'CLOSED'}
                                </span>
                            </TooltipTrigger>
                            <TooltipContent>{t('circuitBreaker.col.state')}</TooltipContent>
                        </Tooltip>
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <span className="inline-flex h-6 w-6 items-center justify-center rounded tabular-nums text-muted-foreground text-left transition-colors hover:bg-[var(--cb-hover-bg)] hover:text-[var(--cb-hover-muted-foreground)]">
                                    {member.cb_trip_count ?? 0}
                                </span>
                            </TooltipTrigger>
                            <TooltipContent>{t('circuitBreaker.col.trip')}</TooltipContent>
                        </Tooltip>
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <span className="inline-flex h-6 items-center rounded px-1.5 tabular-nums text-muted-foreground text-left transition-colors hover:bg-[var(--cb-hover-bg)] hover:text-[var(--cb-hover-muted-foreground)]">
                                    {remainingSeconds}s
                                </span>
                            </TooltipTrigger>
                            <TooltipContent>{t('circuitBreaker.col.openUntil')}</TooltipContent>
                        </Tooltip>
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <button
                                    type="button"
                                    className="group h-6 w-6 justify-self-start grid place-items-center rounded hover:bg-[var(--cb-hover-bg)] hover:text-[var(--cb-hover-muted-foreground)] transition-colors"
                                    onClick={() => onCBReset?.(member.id)}
                                >
                                    <RefreshCcw className="size-3.5 text-muted-foreground transition-colors group-hover:text-[var(--cb-hover-muted-foreground)]" />
                                </button>
                            </TooltipTrigger>
                            <TooltipContent>{t('circuitBreaker.col.action')}</TooltipContent>
                        </Tooltip>
                    </div>
                )}

                {(!showConfirmDelete || !confirmDelete) && (
                    <motion.button
                        layoutId={`delete-btn-member-${layoutScope ?? 'default'}-${member.id}`}
                        type="button"
                        onClick={() => showConfirmDelete ? setConfirmDelete(true) : onRemove(member.id)}
                        className="h-6 w-6 shrink-0 grid place-items-center rounded hover:bg-destructive/10 hover:text-destructive transition-colors"
                        initial={false}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ duration: 0.15 }}
                        style={{ pointerEvents: 'auto' }}
                    >
                        <X className="size-3" />
                    </motion.button>
                )}

                <AnimatePresence>
                    {showConfirmDelete && confirmDelete && (
                        <motion.div
                            layoutId={`delete-btn-member-${layoutScope ?? 'default'}-${member.id}`}
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
    );
}

export interface MemberListProps {
    members: SelectedMember[];
    onReorder: (members: SelectedMember[]) => void;
    onRemove: (id: string) => void;
    onWeightChange?: (id: string, weight: number) => void;
    /**
     * When true, auto-scroll the list to bottom when a *new visible* member appears
     * (i.e. a new member id is added). Useful in "editor" flows. Defaults to true.
     */
    autoScrollOnAdd?: boolean;
    onDragStart?: () => void;
    /**
     * Called only when a drop results in a different order (i.e. commit reorder).
     * Useful for persisting the new order.
     */
    onDrop?: (members: SelectedMember[]) => void;
    /**
     * Called whenever a drag ends (including cancel / same-index drop).
     * Useful for lifecycle cleanup (e.g. clearing "isDragging" flags).
     */
    onDragFinish?: () => void;
    removingIds?: Set<string>;
    showWeight?: boolean;
    /**
     * When true, show a confirmation overlay before removing an item.
     * When false, clicking the delete button removes the item immediately.
     * Defaults to true.
     */
    showConfirmDelete?: boolean;
    layoutScope?: string;
    showCircuitBreaker?: boolean;
    onCBReset?: (id: string) => void;
}

export function MemberList({
    members,
    onReorder,
    onRemove,
    onWeightChange,
    autoScrollOnAdd = true,
    onDragStart,
    onDrop,
    onDragFinish,
    removingIds = new Set(),
    showWeight = false,
    showConfirmDelete = true,
    layoutScope: externalLayoutScope,
    showCircuitBreaker = false,
    onCBReset,
}: MemberListProps) {
    const internalLayoutScope = useId();
    const layoutScope = externalLayoutScope ?? internalLayoutScope;

    const scrollContainerRef = useRef<HTMLDivElement | null>(null);
    const prevMemberCountRef = useRef<number>(0);
    const hasMountedRef = useRef(false);

    const visibleCount = members.filter((m) => !removingIds.has(m.id)).length;
    const isEmpty = visibleCount === 0;
    const t = useTranslations('group');

    useEffect(() => {
        // Skip the initial mount so we don't auto-scroll on first render / initial data load.
        if (!hasMountedRef.current) {
            hasMountedRef.current = true;
            prevMemberCountRef.current = members.length;
            return;
        }

        if (!autoScrollOnAdd) {
            prevMemberCountRef.current = members.length;
            return;
        }

        const hasNewMember = members.length > prevMemberCountRef.current;

        // Auto-scroll only when member count increases (i.e. added; not reorder / not "unhide").
        if (hasNewMember) {
            // Wait a tick for DOM/placeholder/layout to settle.
            requestAnimationFrame(() => {
                const el = scrollContainerRef.current;
                if (!el) return;
                el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
            });
        }

        prevMemberCountRef.current = members.length;
    }, [members.length, autoScrollOnAdd]);

    const handleDragEnd = (result: DropResult) => {
        try {
            const { destination, source } = result;
            if (!destination) return;
            if (destination.index === source.index) return;

            const next = reorderList(members, source.index, destination.index);
            onReorder(next);
            onDrop?.(next);
        } finally {
            // Ensure drag lifecycle always finishes, even when drop is canceled.
            onDragFinish?.();
        }
    };

    return (
        <div className="relative h-full min-h-0">
            <div
                className={cn(
                    'absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground',
                    'transition-opacity duration-200 ease-out',
                    isEmpty ? 'opacity-100' : 'opacity-0 pointer-events-none'
                )}
            >
                <Layers className="size-10 opacity-40" />
                <span className="text-sm">{t('card.empty')}</span>
            </div>

            <div
                className={cn(
                    'h-full overflow-y-auto transition-opacity duration-200',
                    isEmpty ? 'opacity-0' : 'opacity-100'
                )}
                ref={scrollContainerRef}
            >
                <DragDropContext
                    onDragStart={() => onDragStart?.()}
                    onDragEnd={handleDragEnd}
                >
                    <Droppable droppableId={`members-${layoutScope}`}>
                        {(droppableProvided) => (
                            <div
                                ref={droppableProvided.innerRef}
                                {...droppableProvided.droppableProps}
                                className="p-2 flex flex-col space-y-1.5"
                            >
                                {members.map((member, index) => (
                                    <Draggable
                                        key={member.id}
                                        draggableId={member.id}
                                        index={index}
                                        isDragDisabled={removingIds.has(member.id)}
                                    >
                                        {(draggableProvided, snapshot) => (
                                            <MemberItem
                                                member={member}
                                                onRemove={onRemove}
                                                onWeightChange={onWeightChange}
                                                isRemoving={removingIds.has(member.id)}
                                                index={index}
                                                showWeight={showWeight}
                                                showConfirmDelete={showConfirmDelete}
                                                layoutScope={layoutScope}
                                                showCircuitBreaker={showCircuitBreaker}
                                                onCBReset={onCBReset}
                                                dnd={{
                                                    innerRef: draggableProvided.innerRef,
                                                    draggableProps: draggableProvided.draggableProps,
                                                    dragHandleProps: draggableProvided.dragHandleProps,
                                                    isDragging: snapshot.isDragging,
                                                }}
                                            />
                                        )}
                                    </Draggable>
                                ))}
                                {droppableProvided.placeholder}
                            </div>
                        )}
                    </Droppable>
                </DragDropContext>
            </div>
        </div>
    );
}
