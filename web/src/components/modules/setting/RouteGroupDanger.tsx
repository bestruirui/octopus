'use client';

import { useMemo, useState } from 'react';
import { FolderX, Trash2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useDeleteAllGroups, useGroupList } from '@/api/endpoints/group';
import { toast } from '@/components/common/Toast';
import { Button } from '@/components/ui/button';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog';

export function SettingRouteGroupDanger() {
    const t = useTranslations('setting');
    const { data: groups = [] } = useGroupList();
    const deleteAllGroups = useDeleteAllGroups();
    const [open, setOpen] = useState(false);

    const groupCount = groups.length;
    const disabled = groupCount === 0 || deleteAllGroups.isPending;
    const countLabel = useMemo(
        () => t('routeGroups.count', { count: groupCount }),
        [groupCount, t],
    );

    const handleDeleteAll = () => {
        deleteAllGroups.mutate(undefined, {
            onSuccess: (result) => {
                setOpen(false);
                toast.success(t('routeGroups.success', { count: result.deleted_count }));
            },
            onError: (error: Error) => {
                toast.error(t('routeGroups.failed'), { description: error.message });
            },
        });
    };

    return (
        <>
            <div className="waterhouse-island rounded-[2.1rem] border border-destructive/20 bg-card/58 p-6 space-y-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
                <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                    <FolderX className="h-5 w-5 text-destructive" />
                    {t('routeGroups.title')}
                </h2>

                <div className="waterhouse-pod space-y-2 rounded-[1.55rem] border border-destructive/15 bg-destructive/6 p-4 shadow-waterhouse-soft">
                    <p className="text-sm text-muted-foreground">{t('routeGroups.description')}</p>
                    <p className="text-xs text-muted-foreground">{countLabel}</p>
                </div>

                <Button
                    type="button"
                    variant="destructive"
                    className="w-full rounded-xl"
                    disabled={disabled}
                    onClick={() => setOpen(true)}
                >
                    <Trash2 className="size-4" />
                    {deleteAllGroups.isPending ? t('routeGroups.deleting') : t('routeGroups.button')}
                </Button>
            </div>

            <AlertDialog open={open} onOpenChange={setOpen}>
                <AlertDialogContent className="rounded-[2rem]">
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('routeGroups.confirmTitle')}</AlertDialogTitle>
                        <AlertDialogDescription className="whitespace-pre-line">
                            {t('routeGroups.confirmDescription', { count: groupCount })}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={deleteAllGroups.isPending}>
                            {t('routeGroups.cancel')}
                        </AlertDialogCancel>
                        <AlertDialogAction
                            disabled={deleteAllGroups.isPending}
                            onClick={(event) => {
                                event.preventDefault();
                                handleDeleteAll();
                            }}
                        >
                            {deleteAllGroups.isPending ? t('routeGroups.deleting') : t('routeGroups.confirm')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
