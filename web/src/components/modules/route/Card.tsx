'use client';

import { useState, useCallback, useMemo } from 'react';
import { Trash2, X, Pencil, Route as RouteIcon, ChevronDown, ChevronUp, Plus } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { type Route, useDeleteRoute, useUpdateRoute } from '@/api/endpoints/route';
import { useGroupList } from '@/api/endpoints/group';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { RouteEditor, type RouteEditorValues } from './Editor';

interface RouteCardProps {
    route: Route;
}

export function RouteCard({ route }: RouteCardProps) {
    const t = useTranslations('route');
    const deleteRoute = useDeleteRoute();
    const updateRoute = useUpdateRoute();
    const { data: groups = [] } = useGroupList();

    const [confirmDelete, setConfirmDelete] = useState(false);
    const [isEditing, setIsEditing] = useState(false);
    const [expanded, setExpanded] = useState(false);

    const primaryGroup = useMemo(() =>
        groups.find((g) => g.id === route.primary_group_id),
        [groups, route.primary_group_id]
    );

    const dispatchGroup = useMemo(() =>
        route.dispatch_group_id ? groups.find((g) => g.id === route.dispatch_group_id) : null,
        [groups, route.dispatch_group_id]
    );

    const workGroupDetails = useMemo(() =>
        (route.work_groups || []).map((wg) => ({
            ...wg,
            groupName: groups.find((g) => g.id === wg.group_id)?.name || `Group ${wg.group_id}`,
        })),
        [route.work_groups, groups]
    );

    const handleDelete = useCallback(() => {
        if (!route.id) return;
        deleteRoute.mutate(route.id, {
            onSuccess: () => toast.success(t('toast.deleted')),
            onError: (error) => toast.error(t('toast.deleteFailed'), { description: error.message }),
        });
    }, [route.id, deleteRoute, t]);

    const handleEdit = useCallback((values: RouteEditorValues) => {
        if (!route.id) return;
        updateRoute.mutate({
            id: route.id,
            name: values.name !== route.name ? values.name : undefined,
            primary_group_id: values.primary_group_id !== route.primary_group_id ? values.primary_group_id : undefined,
            dispatch_group_id: values.dispatch_group_id !== (route.dispatch_group_id ?? 0) ? values.dispatch_group_id : undefined,
            description: values.description !== (route.description ?? '') ? values.description : undefined,
            work_groups_to_add: values.work_groups_to_add,
            work_groups_to_update: values.work_groups_to_update,
            work_groups_to_delete: values.work_groups_to_delete,
        }, {
            onSuccess: () => {
                toast.success(t('toast.updated'));
                setIsEditing(false);
            },
            onError: (error) => toast.error(t('toast.updateFailed'), { description: error.message }),
        });
    }, [route, updateRoute, t]);

    if (isEditing) {
        return (
            <article className="flex flex-col rounded-3xl border border-primary bg-card text-card-foreground p-4 custom-shadow">
                <header className="flex items-center justify-between mb-3">
                    <h3 className="text-lg font-bold">{t('detail.actions.edit')}</h3>
                    <button
                        type="button"
                        onClick={() => setIsEditing(false)}
                        className="p-1.5 rounded-lg hover:bg-muted text-muted-foreground"
                    >
                        <X className="size-4" />
                    </button>
                </header>
                <RouteEditor
                    initial={{
                        name: route.name,
                        primary_group_id: route.primary_group_id,
                        dispatch_group_id: route.dispatch_group_id ?? 0,
                        description: route.description ?? '',
                    }}
                    initialWorkGroups={route.work_groups}
                    groups={groups}
                    isSubmitting={updateRoute.isPending}
                    isEditing
                    onCancel={() => setIsEditing(false)}
                    onSubmit={handleEdit}
                />
            </article>
        );
    }

    return (
        <article className="flex flex-col rounded-3xl border border-border bg-card text-card-foreground p-4 custom-shadow">
            {/* Header */}
            <header className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-2 min-w-0">
                    <RouteIcon className="size-5 text-primary shrink-0" />
                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger asChild>
                            <h3 className="text-lg font-bold truncate">{route.name}</h3>
                        </TooltipTrigger>
                        <TooltipContent>{route.name}</TooltipContent>
                    </Tooltip>
                </div>

                <div className="flex items-center gap-1 shrink-0">
                    <Tooltip side="top" sideOffset={10} align="center">
                        <TooltipTrigger>
                            <button
                                type="button"
                                onClick={() => setIsEditing(true)}
                                className="p-1.5 rounded-lg transition-colors hover:bg-muted text-muted-foreground hover:text-foreground"
                            >
                                <Pencil className="size-4" />
                            </button>
                        </TooltipTrigger>
                        <TooltipContent>{t('detail.actions.edit')}</TooltipContent>
                    </Tooltip>

                    {!confirmDelete && (
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger>
                                <motion.button
                                    layoutId={`delete-btn-route-${route.id}`}
                                    type="button"
                                    onClick={() => setConfirmDelete(true)}
                                    className="p-1.5 rounded-lg hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                                >
                                    <Trash2 className="size-4" />
                                </motion.button>
                            </TooltipTrigger>
                            <TooltipContent>{t('detail.actions.delete')}</TooltipContent>
                        </Tooltip>
                    )}
                </div>

                <AnimatePresence>
                    {confirmDelete && (
                        <motion.div
                            layoutId={`delete-btn-route-${route.id}`}
                            className="absolute inset-0 flex items-center justify-center gap-2 bg-destructive p-2 rounded-xl"
                            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                        >
                            <button
                                type="button"
                                onClick={() => setConfirmDelete(false)}
                                className="flex h-7 w-7 items-center justify-center rounded-lg bg-destructive-foreground/20 text-destructive-foreground transition-all hover:bg-destructive-foreground/30 active:scale-95"
                            >
                                <X className="size-4" />
                            </button>
                            <button
                                type="button"
                                onClick={handleDelete}
                                disabled={deleteRoute.isPending}
                                className="flex-1 h-7 flex items-center justify-center gap-2 rounded-lg bg-destructive-foreground text-destructive text-sm font-semibold transition-all hover:bg-destructive-foreground/90 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                <Trash2 className="size-3.5" />
                                {t('detail.actions.confirmDelete')}
                            </button>
                        </motion.div>
                    )}
                </AnimatePresence>
            </header>

            {/* Route Info */}
            <div className="space-y-2 mb-3">
                <div className="flex items-center gap-2 text-sm">
                    <span className="text-muted-foreground">{t('detail.primaryGroup')}:</span>
                    <span className="font-medium">{primaryGroup?.name || `Group ${route.primary_group_id}`}</span>
                </div>
                {dispatchGroup && (
                    <div className="flex items-center gap-2 text-sm">
                        <span className="text-muted-foreground">{t('detail.dispatchGroup')}:</span>
                        <span className="font-medium">{dispatchGroup.name}</span>
                    </div>
                )}
                {route.description && (
                    <p className="text-sm text-muted-foreground line-clamp-2">{route.description}</p>
                )}
            </div>

            {/* Work Groups */}
            {workGroupDetails.length > 0 && (
                <div className="rounded-xl border border-border/50 bg-muted/30 overflow-hidden">
                    <button
                        type="button"
                        onClick={() => setExpanded(!expanded)}
                        className="w-full flex items-center justify-between px-3 py-2 text-sm font-medium hover:bg-muted/50 transition-colors"
                    >
                        <span>{t('detail.workGroups')} ({workGroupDetails.length})</span>
                        {expanded ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
                    </button>
                    <AnimatePresence>
                        {expanded && (
                            <motion.div
                                initial={{ height: 0, opacity: 0 }}
                                animate={{ height: 'auto', opacity: 1 }}
                                exit={{ height: 0, opacity: 0 }}
                                transition={{ duration: 0.2 }}
                                className="overflow-hidden"
                            >
                                <div className="px-3 pb-2 space-y-1">
                                    {workGroupDetails.map((wg) => (
                                        <div key={wg.id} className="flex items-center gap-2 text-sm py-1 border-b border-border/30 last:border-0">
                                            <span className="font-medium">{wg.category}</span>
                                            <span className="text-muted-foreground">→</span>
                                            <span>{wg.groupName}</span>
                                            {wg.keywords && (
                                                <span className="ml-auto text-xs text-muted-foreground truncate max-w-[120px]">
                                                    {wg.keywords}
                                                </span>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            </motion.div>
                        )}
                    </AnimatePresence>
                </div>
            )}
        </article>
    );
}
