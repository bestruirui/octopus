'use client';

import {
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { useCreateRoute } from '@/api/endpoints/route';
import { useGroupList } from '@/api/endpoints/group';
import { useTranslations } from 'next-intl';
import { RouteEditor } from './Editor';
import { toast } from '@/components/common/Toast';

export function CreateDialogContent() {
    const { setIsOpen } = useMorphingDialog();
    const createRoute = useCreateRoute();
    const { data: groups = [] } = useGroupList();
    const t = useTranslations('route');

    return (
        <div className="w-screen max-w-full md:max-w-4xl h-[calc(100vh-2rem)] min-h-0 flex flex-col">
            <MorphingDialogTitle className="shrink-0">
                <header className="mb-5 flex items-center justify-between">
                    <h2 className="text-2xl font-bold text-card-foreground">
                        {t('create.title')}
                    </h2>
                    <MorphingDialogClose
                        className="relative right-0 top-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 },
                        }}
                    />
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription className="flex-1 min-h-0 overflow-hidden">
                <RouteEditor
                    groups={groups}
                    isSubmitting={createRoute.isPending}
                    onCancel={() => setIsOpen(false)}
                    onSubmit={(values) => {
                        createRoute.mutate(
                            {
                                name: values.name,
                                primary_group_id: values.primary_group_id,
                                dispatch_group_id: values.dispatch_group_id || undefined,
                                description: values.description || undefined,
                                work_groups: values.work_groups,
                            },
                            {
                                onSuccess: () => {
                                    toast.success(t('toast.created'));
                                    setIsOpen(false);
                                },
                                onError: (error) => toast.error(t('toast.createFailed'), { description: error.message }),
                            }
                        );
                    }}
                />
            </MorphingDialogDescription>
        </div>
    );
}
