'use client';

import { memo, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import { Pencil, Trash2, ArrowDownToLine, ArrowUpFromLine, ChevronDown, CircleCheckBig, Gauge, KeyRound, RadioTower } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { useTranslations } from 'next-intl';
import { useUpdateModel, useDeleteModel, type ModelMarketItem } from '@/api/endpoints/model';
import { getModelIcon } from '@/lib/model-icons';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { ModelDeleteOverlay, ModelEditOverlay } from './ItemOverlays';
import { cn } from '@/lib/utils';
import { createPortal } from 'react-dom';
import { CopyIconButton } from '@/components/common/CopyButton';

interface ModelItemProps {
    model: ModelMarketItem;
    layout?: 'grid' | 'list';
}

export const ModelItem = memo(function ModelItem({ model, layout = 'grid' }: ModelItemProps) {
    const t = useTranslations('model');
    const isListLayout = layout === 'list';
    const [isEditOpen, setIsEditOpen] = useState(false);
    const [confirmDelete, setConfirmDelete] = useState(false);
    const [isExpanded, setIsExpanded] = useState(false);
    const [overlayRect, setOverlayRect] = useState<{ top: number; left: number; width: number } | null>(null);
    const instanceId = useId();
    const editLayoutId = `edit-btn-${model.name}-${instanceId}`;
    const deleteLayoutId = `delete-btn-${model.name}-${instanceId}`;
    const cardRef = useRef<HTMLElement | null>(null);
    const editButtonRef = useRef<HTMLButtonElement | null>(null);
    const editOverlayRef = useRef<HTMLDivElement | null>(null);
    const [editValues, setEditValues] = useState(() => ({
        input: model.input.toString(),
        output: model.output.toString(),
        cache_read: model.cache_read.toString(),
        cache_write: model.cache_write.toString(),
    }));

    const updateModel = useUpdateModel();
    const deleteModel = useDeleteModel();

    const { Avatar: ModelAvatar, color: brandColor, label: providerLabel } = useMemo(() => getModelIcon(model.name), [model.name]);
    const requestCount = model.request_success + model.request_failed;
    const visibleChannelTags = useMemo(() => model.channels.slice(0, isListLayout ? 4 : 3), [isListLayout, model.channels]);
    const hiddenChannelTagCount = Math.max(0, model.channels.length - visibleChannelTags.length);
    const successRateLabel = requestCount > 0 ? `${(model.success_rate * 100).toFixed(2)}%` : '—';
    const latencyLabel = requestCount > 0 && model.average_latency_ms > 0 ? `${model.average_latency_ms}ms` : '—';

    const updateOverlayRect = useCallback(() => {
        const card = cardRef.current;
        if (!card) return;
        const rect = card.getBoundingClientRect();
        setOverlayRect((prev) => {
            if (prev && prev.top === rect.top && prev.left === rect.left && prev.width === rect.width) {
                return prev;
            }
            return { top: rect.top, left: rect.left, width: rect.width };
        });
    }, []);

    const closeEdit = useCallback(() => {
        setIsEditOpen(false);
    }, []);

    const handleEditClick = () => {
        setConfirmDelete(false);
        setEditValues({
            input: model.input.toString(),
            output: model.output.toString(),
            cache_read: model.cache_read.toString(),
            cache_write: model.cache_write.toString(),
        });
        // Ensure first open already has anchor geometry so layout animation can run.
        updateOverlayRect();
        setIsEditOpen(true);
    };

    const handleCancelEdit = () => {
        closeEdit();
    };

    const handleSaveEdit = () => {
        updateModel.mutate({
            name: model.name,
            input: parseFloat(editValues.input) || 0,
            output: parseFloat(editValues.output) || 0,
            cache_read: parseFloat(editValues.cache_read) || 0,
            cache_write: parseFloat(editValues.cache_write) || 0,
        }, {
            onSuccess: () => {
                closeEdit();
                toast.success(t('toast.updated'));
            },
            onError: (error) => {
                toast.error(t('toast.updateFailed'), { description: error.message });
            }
        });
    };

    const handleDeleteClick = () => {
        closeEdit();
        setConfirmDelete(true);
    };
    const handleCancelDelete = () => setConfirmDelete(false);
    const handleConfirmDelete = () => {
        deleteModel.mutate(model.name, {
            onSuccess: () => {
                setConfirmDelete(false);
                toast.success(t('toast.deleted'));
            },
            onError: (error) => {
                setConfirmDelete(false);
                toast.error(t('toast.deleteFailed'), { description: error.message });
            }
        });
    };

    useEffect(() => {
        if (!isEditOpen) return;

        const handlePointerDown = (event: PointerEvent) => {
            const target = event.target as Node | null;
            if (!target) return;
            if (editOverlayRef.current?.contains(target)) return;
            if (editButtonRef.current?.contains(target)) return;
            closeEdit();
        };

        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') closeEdit();
        };

        updateOverlayRect();
        window.addEventListener('resize', updateOverlayRect);
        window.addEventListener('scroll', updateOverlayRect, true);
        document.addEventListener('pointerdown', handlePointerDown);
        document.addEventListener('keydown', handleKeyDown);

        return () => {
            window.removeEventListener('resize', updateOverlayRect);
            window.removeEventListener('scroll', updateOverlayRect, true);
            document.removeEventListener('pointerdown', handlePointerDown);
            document.removeEventListener('keydown', handleKeyDown);
        };
    }, [isEditOpen, updateOverlayRect, closeEdit]);

    const shouldRenderEditPortal = isEditOpen || overlayRect !== null;

    return (
        <article
            ref={cardRef}
            className={cn(
                'group relative rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow transition-all duration-300',
                (isEditOpen || confirmDelete) && 'z-50'
            )}
        >
            <div className="flex items-start gap-4">
                <div className="shrink-0">
                    <ModelAvatar size={52} />
                </div>

                <div className="min-w-0 flex-1 space-y-3">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="min-w-0 space-y-1">
                            <Tooltip side="top" sideOffset={10} align="start">
                                <TooltipTrigger className="max-w-full truncate text-left text-base font-semibold leading-tight text-card-foreground">
                                    {model.name}
                                </TooltipTrigger>
                                <TooltipContent key={model.name}>{model.name}</TooltipContent>
                            </Tooltip>
                            <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                                <span
                                    className="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium"
                                    style={{ borderColor: `${brandColor}55`, color: brandColor }}
                                >
                                    {providerLabel}
                                </span>
                                <span>{t('card.requests')}: {requestCount.toLocaleString()}</span>
                            </div>
                        </div>

                        <div
                            className={cn(
                                'flex shrink-0 items-center gap-2',
                                (isEditOpen || confirmDelete) && 'invisible pointer-events-none'
                            )}
                        >
                            <CopyIconButton
                                text={model.name}
                                className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-muted/20 text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground"
                                copyIconClassName="size-4"
                                checkIconClassName="size-4"
                            />
                            <button
                                type="button"
                                onClick={() => setIsExpanded((prev) => !prev)}
                                className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-muted/20 text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground"
                                title={isExpanded ? t('card.collapse') : t('card.expand')}
                            >
                                <ChevronDown className={cn('size-4 transition-transform', isExpanded && 'rotate-180')} />
                            </button>
                            <motion.button
                                ref={editButtonRef}
                                layoutId={editLayoutId}
                                type="button"
                                onClick={handleEditClick}
                                disabled={isEditOpen || confirmDelete}
                                className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-muted/20 text-muted-foreground transition-colors hover:bg-muted/30 hover:text-foreground disabled:opacity-50"
                                title={t('card.edit')}
                            >
                                <Pencil className="size-4" />
                            </motion.button>
                            <motion.button
                                layoutId={deleteLayoutId}
                                type="button"
                                onClick={handleDeleteClick}
                                disabled={isEditOpen || confirmDelete}
                                className="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-destructive/20 bg-destructive/10 text-destructive transition-colors hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
                                title={t('card.delete')}
                            >
                                <Trash2 className="size-4" />
                            </motion.button>
                        </div>
                    </div>

                    <div className={cn('grid gap-2 text-sm text-muted-foreground', isListLayout ? 'grid-cols-2 xl:grid-cols-4' : 'grid-cols-2')}>
                        <div className="inline-flex items-center gap-2 rounded-2xl border border-border/60 bg-muted/20 px-3 py-2">
                            <RadioTower className="size-4" style={{ color: brandColor }} />
                            <span>{t('card.channels')}</span>
                            <span className="tabular-nums text-foreground">{model.channel_count}</span>
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-2xl border border-border/60 bg-muted/20 px-3 py-2">
                            <KeyRound className="size-4" style={{ color: brandColor }} />
                            <span>{t('card.keys')}</span>
                            <span className="tabular-nums text-foreground">{model.enabled_key_count}</span>
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-2xl border border-border/60 bg-muted/20 px-3 py-2">
                            <Gauge className="size-4" style={{ color: brandColor }} />
                            <span>{t('card.latency')}</span>
                            <span className="tabular-nums text-foreground">{latencyLabel}</span>
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-2xl border border-border/60 bg-muted/20 px-3 py-2">
                            <CircleCheckBig className="size-4" style={{ color: brandColor }} />
                            <span>{t('card.successRate')}</span>
                            <span className="tabular-nums text-foreground">{successRateLabel}</span>
                        </div>
                    </div>

                    <div className="flex flex-wrap gap-2">
                        <span className="rounded-full border border-border/60 bg-muted/20 px-2.5 py-1 text-xs font-medium text-foreground">
                            {providerLabel}
                        </span>
                        {visibleChannelTags.map((channel) => (
                            <span
                                key={`${model.name}-${channel.channel_id}`}
                                className="rounded-full border border-border/60 bg-muted/20 px-2.5 py-1 text-xs text-muted-foreground"
                            >
                                {channel.channel_name}
                            </span>
                        ))}
                        {hiddenChannelTagCount > 0 ? (
                            <span className="rounded-full border border-border/60 bg-muted/20 px-2.5 py-1 text-xs text-muted-foreground">
                                +{hiddenChannelTagCount}
                            </span>
                        ) : null}
                    </div>
                </div>
            </div>

            <AnimatePresence initial={false}>
                {isExpanded ? (
                    <motion.div
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: 'auto', opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        transition={{ duration: 0.2 }}
                        className="overflow-hidden"
                    >
                        <div className="mt-4 space-y-4 border-t border-border/60 pt-4">
                            <div className={cn('grid gap-3', isListLayout ? 'xl:grid-cols-3' : 'md:grid-cols-2')}>
                                <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                                    <h4 className="text-sm font-medium text-foreground">{t('detail.pricing')}</h4>
                                    <div className="mt-3 space-y-2 text-sm text-muted-foreground">
                                        <div className="flex items-center justify-between gap-3">
                                            <span className="inline-flex items-center gap-1.5">
                                                <ArrowDownToLine className="size-3.5" style={{ color: brandColor }} />
                                                {t('card.inputCache')}
                                            </span>
                                            <span className="tabular-nums text-foreground">{model.input.toFixed(2)}/{model.cache_read.toFixed(2)}$</span>
                                        </div>
                                        <div className="flex items-center justify-between gap-3">
                                            <span className="inline-flex items-center gap-1.5">
                                                <ArrowUpFromLine className="size-3.5" style={{ color: brandColor }} />
                                                {t('card.outputCache')}
                                            </span>
                                            <span className="tabular-nums text-foreground">{model.output.toFixed(2)}/{model.cache_write.toFixed(2)}$</span>
                                        </div>
                                    </div>
                                </div>

                                <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                                    <h4 className="text-sm font-medium text-foreground">{t('detail.runtime')}</h4>
                                    <div className="mt-3 grid grid-cols-2 gap-2 text-sm text-muted-foreground">
                                        <div className="rounded-xl bg-background/70 px-3 py-2">
                                            <div>{t('detail.requestSuccess')}</div>
                                            <div className="mt-1 tabular-nums text-base font-medium text-foreground">{model.request_success.toLocaleString()}</div>
                                        </div>
                                        <div className="rounded-xl bg-background/70 px-3 py-2">
                                            <div>{t('detail.requestFailed')}</div>
                                            <div className="mt-1 tabular-nums text-base font-medium text-foreground">{model.request_failed.toLocaleString()}</div>
                                        </div>
                                        <div className="rounded-xl bg-background/70 px-3 py-2">
                                            <div>{t('card.latency')}</div>
                                            <div className="mt-1 tabular-nums text-base font-medium text-foreground">{latencyLabel}</div>
                                        </div>
                                        <div className="rounded-xl bg-background/70 px-3 py-2">
                                            <div>{t('card.successRate')}</div>
                                            <div className="mt-1 tabular-nums text-base font-medium text-foreground">{successRateLabel}</div>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                                <div className="flex flex-wrap items-center justify-between gap-3">
                                    <h4 className="text-sm font-medium text-foreground">{t('detail.channels')}</h4>
                                    <span className="text-xs text-muted-foreground">{model.channels.length} {t('card.channels')}</span>
                                </div>
                                <div className="mt-3 grid gap-2">
                                    {model.channels.length === 0 ? (
                                        <div className="rounded-xl bg-background/70 px-3 py-2 text-sm text-muted-foreground">
                                            {t('detail.noChannels')}
                                        </div>
                                    ) : (
                                        model.channels.map((channel) => (
                                            <div key={`${model.name}-detail-${channel.channel_id}`} className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-background/70 px-3 py-3">
                                                <div className="min-w-0">
                                                    <div className="truncate text-sm font-medium text-foreground">{channel.channel_name}</div>
                                                    <div className="mt-1 text-xs text-muted-foreground">ID {channel.channel_id}</div>
                                                </div>
                                                <div className="flex flex-wrap items-center gap-2 text-xs">
                                                    <span className={cn(
                                                        'rounded-full border px-2.5 py-1',
                                                        channel.enabled
                                                            ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700'
                                                            : 'border-border/60 bg-muted/40 text-muted-foreground'
                                                    )}>
                                                        {channel.enabled ? t('detail.enabled') : t('detail.disabled')}
                                                    </span>
                                                    <span className="rounded-full border border-border/60 bg-muted/20 px-2.5 py-1 text-muted-foreground">
                                                        {channel.enabled_key_count} {t('detail.keyCount')}
                                                    </span>
                                                </div>
                                            </div>
                                        ))
                                    )}
                                </div>
                            </div>
                        </div>
                    </motion.div>
                ) : null}
            </AnimatePresence>

            <AnimatePresence>
                {confirmDelete && (
                    <ModelDeleteOverlay
                        layoutId={deleteLayoutId}
                        isPending={deleteModel.isPending}
                        onCancel={handleCancelDelete}
                        onConfirm={handleConfirmDelete}
                    />
                )}
            </AnimatePresence>

            {shouldRenderEditPortal && typeof document !== 'undefined'
                ? createPortal(
                    <AnimatePresence onExitComplete={() => setOverlayRect(null)}>
                        {isEditOpen && overlayRect && (
                            <div
                                ref={editOverlayRef}
                                className="fixed z-[90]"
                                style={{
                                    top: `${overlayRect.top}px`,
                                    left: `${overlayRect.left}px`,
                                    width: `${overlayRect.width}px`,
                                }}
                            >
                                <div className="relative">
                                    <ModelEditOverlay
                                        layoutId={editLayoutId}
                                        modelName={model.name}
                                        brandColor={brandColor}
                                        editValues={editValues}
                                        isPending={updateModel.isPending}
                                        onChange={setEditValues}
                                        onCancel={handleCancelEdit}
                                        onSave={handleSaveEdit}
                                    />
                                </div>
                            </div>
                        )}
                    </AnimatePresence>,
                    document.body
                )
                : null}
        </article>
    );
});
