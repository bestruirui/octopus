'use client';

import { Check, Loader, Trash2, X } from 'lucide-react';
import { motion } from 'motion/react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/input';

type EditValues = {
    input: string;
    output: string;
    cache_read: string;
    cache_write: string;
};

type ModelDeleteOverlayProps = {
    layoutId: string;
    isPending: boolean;
    onCancel: () => void;
    onConfirm: () => void;
};

export function ModelDeleteOverlay({
    layoutId,
    isPending,
    onCancel,
    onConfirm,
}: ModelDeleteOverlayProps) {
    const t = useTranslations('model.overlay');
    return (
        <motion.div
            layoutId={layoutId}
            className="absolute inset-0 flex items-center justify-center gap-3 rounded-[1.7rem] border border-destructive/20 bg-[radial-gradient(circle_at_50%_0%,rgba(255,255,255,0.18),transparent_40%),linear-gradient(145deg,color-mix(in_oklch,var(--destructive)_92%,black),color-mix(in_oklch,var(--destructive)_72%,black))] p-4 shadow-waterhouse-deep backdrop-blur-md"
            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
        >
            <button
                type="button"
                onClick={onCancel}
                className="waterhouse-pod flex h-10 items-center justify-center gap-1.5 rounded-[1.15rem] border border-white/15 bg-destructive-foreground/16 px-4 text-sm font-medium text-destructive-foreground transition-all hover:bg-destructive-foreground/24 active:scale-[0.98]"
            >
                <X className="size-4" />
                {t('cancel')}
            </button>
            <button
                type="button"
                onClick={onConfirm}
                disabled={isPending}
                className="flex h-10 items-center justify-center gap-1.5 rounded-[1.15rem] bg-destructive-foreground px-4 text-sm font-medium text-destructive transition-all hover:bg-destructive-foreground/92 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
            >
                {isPending ? (
                    <Loader className="size-4 animate-spin" />
                ) : (
                    <Trash2 className="size-4" />
                )}
                {isPending ? t('deleting') : t('confirmDelete')}
            </button>
        </motion.div>
    );
}

type ModelEditOverlayProps = {
    layoutId: string;
    modelName: string;
    brandColor: string;
    editValues: EditValues;
    isPending: boolean;
    onChange: (next: EditValues) => void;
    onCancel: () => void;
    onSave: () => void;
};

export function ModelEditOverlay({
    layoutId,
    modelName,
    brandColor,
    editValues,
    isPending,
    onChange,
    onCancel,
    onSave,
}: ModelEditOverlayProps) {
    const t = useTranslations('model.overlay');
    return (
        <motion.div
            layoutId={layoutId}
            className="absolute inset-x-0 top-0 z-20 flex flex-col overflow-hidden rounded-[2rem] border border-border/35 bg-background/88 p-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]"
            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
        >
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_14%,color-mix(in_oklch,var(--waterhouse-highlight)_16%,transparent)_0%,transparent_28%),linear-gradient(145deg,color-mix(in_oklch,white_10%,transparent),transparent_52%,color-mix(in_oklch,var(--primary)_6%,transparent))]" />
            <div className="relative">
                <div className="mb-3 inline-flex items-center rounded-full border border-primary/12 bg-background/42 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft">
                    {t('save')}
                </div>
                <h3 className="mb-4 line-clamp-1 text-base font-semibold text-card-foreground">
                    {modelName}
                </h3>

                <div className="grid grid-cols-2 gap-2">
                    <label className="grid gap-1.5 text-xs text-muted-foreground">
                        {t('input')}
                        <Input
                            type="number"
                            step="any"
                            value={editValues.input}
                            onChange={(e) => onChange({ ...editValues, input: e.target.value })}
                            className="h-10 rounded-[1.1rem] border-border/25 bg-background/46 text-sm shadow-waterhouse-soft"
                        />
                    </label>
                    <label className="grid gap-1.5 text-xs text-muted-foreground">
                        {t('output')}
                        <Input
                            type="number"
                            step="any"
                            value={editValues.output}
                            onChange={(e) => onChange({ ...editValues, output: e.target.value })}
                            className="h-10 rounded-[1.1rem] border-border/25 bg-background/46 text-sm shadow-waterhouse-soft"
                        />
                    </label>
                    <label className="grid gap-1.5 text-xs text-muted-foreground">
                        {t('cacheRead')}
                        <Input
                            type="number"
                            step="any"
                            value={editValues.cache_read}
                            onChange={(e) => onChange({ ...editValues, cache_read: e.target.value })}
                            className="h-10 rounded-[1.1rem] border-border/25 bg-background/46 text-sm shadow-waterhouse-soft"
                        />
                    </label>
                    <label className="grid gap-1.5 text-xs text-muted-foreground">
                        {t('cacheWrite')}
                        <Input
                            type="number"
                            step="any"
                            value={editValues.cache_write}
                            onChange={(e) => onChange({ ...editValues, cache_write: e.target.value })}
                            className="h-10 rounded-[1.1rem] border-border/25 bg-background/46 text-sm shadow-waterhouse-soft"
                        />
                    </label>
                </div>

                <div className="mt-4 flex gap-2">
                    <button
                        type="button"
                        onClick={onCancel}
                        disabled={isPending}
                        className="waterhouse-pod flex h-10 flex-1 items-center justify-center gap-1.5 rounded-[1.15rem] border border-border/25 bg-background/46 text-sm font-medium text-muted-foreground shadow-waterhouse-soft transition-all hover:bg-background/62 active:scale-[0.98] disabled:opacity-50"
                    >
                        <X className="size-4" />
                        {t('cancel')}
                    </button>
                    <button
                        type="button"
                        onClick={onSave}
                        disabled={isPending}
                        className="flex h-10 flex-1 items-center justify-center gap-1.5 rounded-[1.15rem] text-sm font-medium shadow-waterhouse-soft transition-all active:scale-[0.98] disabled:opacity-50"
                        style={{ backgroundColor: brandColor, color: '#fff' }}
                    >
                        {isPending ? <Loader className="size-4 animate-spin" /> : <Check className="size-4" />}
                        {t('save')}
                    </button>
                </div>
            </div>
        </motion.div>
    );
}
