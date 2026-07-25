'use client';

import { useCallback, useState, type FormEvent } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { type Group } from '@/api/endpoints/group';
import { type RouteGroup, type RouteGroupAddRequest, type RouteGroupUpdateRequest } from '@/api/endpoints/route';
import { Button } from '@/components/ui/button';
import { useTranslations } from 'next-intl';

interface WorkGroupRow {
    uid: string;
    id?: number;
    group_id: number;
    category: string;
    description: string;
    keywords: string;
}

export interface RouteEditorValues {
    name: string;
    primary_group_id: number;
    dispatch_group_id: number;
    description: string;
    work_groups?: RouteGroupAddRequest[];
    work_groups_to_add?: RouteGroupAddRequest[];
    work_groups_to_update?: RouteGroupUpdateRequest[];
    work_groups_to_delete?: number[];
}

interface RouteEditorProps {
    initial?: Partial<RouteEditorValues>;
    initialWorkGroups?: RouteGroup[];
    groups: Group[];
    isSubmitting?: boolean;
    isEditing?: boolean;
    onCancel?: () => void;
    onSubmit: (values: RouteEditorValues) => void;
    submitText?: string;
}

let uidCounter = 0;
function nextUid() {
    return `wg-${++uidCounter}-${Date.now()}`;
}

function toRow(wg: RouteGroup): WorkGroupRow {
    return {
        uid: nextUid(),
        id: wg.id,
        group_id: wg.group_id,
        category: wg.category,
        description: wg.description ?? '',
        keywords: wg.keywords ?? '',
    };
}

export function RouteEditor({
    initial,
    initialWorkGroups,
    groups,
    isSubmitting,
    isEditing = false,
    onCancel,
    onSubmit,
    submitText,
}: RouteEditorProps) {
    const t = useTranslations('route');

    const [name, setName] = useState(initial?.name ?? '');
    const [primaryGroupId, setPrimaryGroupId] = useState(initial?.primary_group_id ?? 0);
    const [dispatchGroupId, setDispatchGroupId] = useState(initial?.dispatch_group_id ?? 0);
    const [description, setDescription] = useState(initial?.description ?? '');
    const [workGroups, setWorkGroups] = useState<WorkGroupRow[]>(
        initialWorkGroups?.map(toRow) ?? []
    );

    const addWorkGroup = useCallback(() => {
        setWorkGroups((prev) => [
            ...prev,
            { uid: nextUid(), group_id: 0, category: '', description: '', keywords: '' },
        ]);
    }, []);

    const removeWorkGroup = useCallback((uid: string) => {
        setWorkGroups((prev) => prev.filter((w) => w.uid !== uid));
    }, []);

    const updateWorkGroup = useCallback((uid: string, field: keyof WorkGroupRow, value: string | number) => {
        setWorkGroups((prev) =>
            prev.map((w) => (w.uid === uid ? { ...w, [field]: value } : w))
        );
    }, []);

    const handleSubmit = useCallback((e: FormEvent) => {
        e.preventDefault();
        if (!name.trim() || primaryGroupId === 0) return;

        const base = {
            name: name.trim(),
            primary_group_id: primaryGroupId,
            dispatch_group_id: dispatchGroupId,
            description: description.trim(),
        };

        const validWorkGroups = workGroups.filter((w) => w.group_id > 0 && w.category.trim());

        if (isEditing) {
            const originalIds = new Set(initialWorkGroups?.map((w) => w.id).filter(Boolean) as number[]);
            const currentIds = new Set(validWorkGroups.filter((w) => w.id).map((w) => w.id!));

            const toDelete = [...originalIds].filter((id) => !currentIds.has(id));

            const toAdd: RouteGroupAddRequest[] = validWorkGroups
                .filter((w) => !w.id)
                .map((w) => ({
                    group_id: w.group_id,
                    category: w.category.trim(),
                    description: w.description.trim() || undefined,
                    keywords: w.keywords.trim() || undefined,
                }));

            const toUpdate: RouteGroupUpdateRequest[] = validWorkGroups
                .filter((w) => {
                    if (!w.id) return false;
                    const orig = initialWorkGroups?.find((o) => o.id === w.id);
                    if (!orig) return true;
                    return (
                        w.group_id !== orig.group_id ||
                        w.category.trim() !== orig.category ||
                        (w.description.trim() || undefined) !== (orig.description || undefined) ||
                        (w.keywords.trim() || undefined) !== (orig.keywords || undefined)
                    );
                })
                .map((w) => ({
                    id: w.id!,
                    group_id: w.group_id,
                    category: w.category.trim(),
                    description: w.description.trim() || undefined,
                    keywords: w.keywords.trim() || undefined,
                }));

            onSubmit({
                ...base,
                work_groups_to_add: toAdd.length > 0 ? toAdd : undefined,
                work_groups_to_update: toUpdate.length > 0 ? toUpdate : undefined,
                work_groups_to_delete: toDelete.length > 0 ? toDelete : undefined,
            });
        } else {
            onSubmit({
                ...base,
                work_groups: validWorkGroups.map((w) => ({
                    group_id: w.group_id,
                    category: w.category.trim(),
                    description: w.description.trim() || undefined,
                    keywords: w.keywords.trim() || undefined,
                })),
            });
        }
    }, [name, primaryGroupId, dispatchGroupId, description, workGroups, isEditing, initialWorkGroups, onSubmit]);

    return (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {/* Route Name */}
            <div className="space-y-2">
                <label className="text-sm font-medium">{t('editor.name')}</label>
                <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder={t('editor.namePlaceholder')}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground"
                    required
                />
            </div>

            {/* Primary Group */}
            <div className="space-y-2">
                <label className="text-sm font-medium">{t('editor.primaryGroup')} *</label>
                <select
                    value={primaryGroupId}
                    onChange={(e) => setPrimaryGroupId(Number(e.target.value))}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground"
                    required
                >
                    <option value={0}>{t('editor.selectGroup')}</option>
                    {groups.map((g) => (
                        <option key={g.id} value={g.id}>{g.name}</option>
                    ))}
                </select>
            </div>

            {/* Dispatch Group */}
            <div className="space-y-2">
                <label className="text-sm font-medium">{t('editor.dispatchGroup')}</label>
                <select
                    value={dispatchGroupId}
                    onChange={(e) => setDispatchGroupId(Number(e.target.value))}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground"
                >
                    <option value={0}>{t('editor.noDispatch')}</option>
                    {groups.map((g) => (
                        <option key={g.id} value={g.id}>{g.name}</option>
                    ))}
                </select>
                <p className="text-xs text-muted-foreground">{t('editor.dispatchHint')}</p>
            </div>

            {/* Description */}
            <div className="space-y-2">
                <label className="text-sm font-medium">{t('editor.description')}</label>
                <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder={t('editor.descriptionPlaceholder')}
                    className="w-full px-3 py-2 rounded-lg border border-border bg-background text-foreground resize-none"
                    rows={2}
                />
            </div>

            {/* Work Groups */}
            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium">{t('editor.workGroups')}</label>
                    <Button type="button" variant="outline" size="sm" onClick={addWorkGroup}>
                        <Plus className="size-3.5 mr-1" />
                        {t('editor.addWorkGroup')}
                    </Button>
                </div>

                {workGroups.length === 0 ? (
                    <p className="text-xs text-muted-foreground py-2">{t('editor.noWorkGroups')}</p>
                ) : (
                    <div className="space-y-2">
                        {workGroups.map((wg) => (
                            <div key={wg.uid} className="rounded-lg border border-border/50 bg-muted/20 p-2 space-y-2">
                                <div className="flex items-center gap-2">
                                    <select
                                        value={wg.group_id}
                                        onChange={(e) => updateWorkGroup(wg.uid, 'group_id', Number(e.target.value))}
                                        className="flex-1 px-2 py-1.5 rounded-md border border-border bg-background text-foreground text-sm"
                                    >
                                        <option value={0}>{t('editor.selectGroup')}</option>
                                        {groups.map((g) => (
                                            <option key={g.id} value={g.id}>{g.name}</option>
                                        ))}
                                    </select>
                                    <button
                                        type="button"
                                        onClick={() => removeWorkGroup(wg.uid)}
                                        className="p-1.5 rounded-md hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                                    >
                                        <Trash2 className="size-3.5" />
                                    </button>
                                </div>
                                <div className="grid grid-cols-2 gap-2">
                                    <input
                                        type="text"
                                        value={wg.category}
                                        onChange={(e) => updateWorkGroup(wg.uid, 'category', e.target.value)}
                                        placeholder={t('editor.categoryPlaceholder')}
                                        className="px-2 py-1.5 rounded-md border border-border bg-background text-foreground text-sm"
                                    />
                                    <input
                                        type="text"
                                        value={wg.keywords}
                                        onChange={(e) => updateWorkGroup(wg.uid, 'keywords', e.target.value)}
                                        placeholder={t('editor.keywordsPlaceholder')}
                                        className="px-2 py-1.5 rounded-md border border-border bg-background text-foreground text-sm"
                                    />
                                </div>
                                <input
                                    type="text"
                                    value={wg.description}
                                    onChange={(e) => updateWorkGroup(wg.uid, 'description', e.target.value)}
                                    placeholder={t('editor.descriptionPlaceholder')}
                                    className="w-full px-2 py-1.5 rounded-md border border-border bg-background text-foreground text-sm"
                                />
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Actions */}
            <div className="flex gap-2 justify-end">
                {onCancel && (
                    <Button type="button" variant="outline" onClick={onCancel}>
                        {t('editor.cancel')}
                    </Button>
                )}
                <Button type="submit" disabled={isSubmitting || !name.trim() || primaryGroupId === 0}>
                    {isSubmitting ? t('editor.submitting') : (submitText || t('editor.submit'))}
                </Button>
            </div>
        </form>
    );
}
