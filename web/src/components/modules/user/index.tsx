'use client';

import { useState } from 'react';
import { Users, UserCog, Trash2, Shield, ShieldCheck, ShieldAlert, Plus } from 'lucide-react';
import { useUserList, useUpdateUserRole, useDeleteUser, type UserInfo } from '@/api/endpoints/user';
import { PageWrapper } from '@/components/common/PageWrapper';
import { Loader } from 'lucide-react';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';
import {
    MorphingDialog,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogTrigger,
} from '@/components/ui/morphing-dialog';
import { buttonVariants } from '@/components/ui/button';
import { CreateDialogContent } from './Create';

function RoleBadge({ role, label }: { role: string; label: string }) {
    const config = {
        admin: { icon: ShieldCheck, color: 'text-amber-500 bg-amber-500/10 border-amber-500/20' },
        editor: { icon: Shield, color: 'text-blue-500 bg-blue-500/10 border-blue-500/20' },
        viewer: { icon: ShieldAlert, color: 'text-slate-500 bg-slate-500/10 border-slate-500/20' },
    }[role] || { icon: ShieldAlert, color: 'text-slate-500 bg-slate-500/10' };
    const Icon = config.icon;
    return (
        <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium border ${config.color}`}>
            <Icon className="h-3 w-3" />
            {label}
        </span>
    );
}

export function User() {
    const t = useTranslations('user');
    const { data: users, isLoading } = useUserList();
    const updateRole = useUpdateUserRole();
    const deleteUser = useDeleteUser();
    const [editingId, setEditingId] = useState<number | null>(null);
    const roleOptions = ['admin', 'editor', 'viewer'] as const;

    const handleRoleChange = (id: number, role: string) => {
        updateRole.mutate({ id, role }, {
            onSuccess: () => {
                toast.success(t('toast.roleUpdated'));
                setEditingId(null);
            },
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    const handleDelete = (id: number) => {
        if (!confirm(t('confirmDelete'))) return;
        deleteUser.mutate(id, {
            onSuccess: () => toast.success(t('toast.deleted')),
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    if (isLoading) return <Loader className="size-6 animate-spin mx-auto mt-12" />;

    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl space-y-4 pb-24 md:pb-6">
            <div className="rounded-3xl border border-border bg-card p-6">
                <div className="mb-4 flex items-center justify-between gap-3">
                    <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                        <Users className="h-5 w-5" />
                        {t('title')}
                    </h2>

                    <MorphingDialog>
                        <MorphingDialogTrigger className={buttonVariants({ variant: 'outline', className: 'rounded-xl min-w-28' })}>
                            <Plus className="mr-2 size-4" />
                            {t('create.open')}
                        </MorphingDialogTrigger>
                        <MorphingDialogContainer>
                            <MorphingDialogContent className="w-fit max-w-full bg-card text-card-foreground px-6 py-4 rounded-3xl custom-shadow max-h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                                <CreateDialogContent />
                            </MorphingDialogContent>
                        </MorphingDialogContainer>
                    </MorphingDialog>
                </div>

                <div className="space-y-2">
                    {(users || []).map((user: UserInfo) => (
                        <div key={user.id}
                            className="flex items-center justify-between gap-4 p-3 rounded-xl bg-muted/50 hover:bg-muted transition-colors">
                            <div className="flex items-center gap-3 min-w-0">
                                <UserCog className="h-5 w-5 text-muted-foreground shrink-0" />
                                <span className="font-medium text-sm truncate">{user.username}</span>
                                <RoleBadge role={user.role} label={t(`roles.${user.role}`)} />
                            </div>

                            <div className="flex items-center gap-2 shrink-0">
                                {editingId === user.id ? (
                                    <div className="flex gap-1">
                                        {roleOptions.map((role) => (
                                            <button key={role}
                                                onClick={() => handleRoleChange(user.id, role)}
                                                className={`px-2.5 py-1 rounded-lg text-xs font-medium transition-all active:scale-95
                                                    ${user.role === role
                                                        ? 'bg-primary text-primary-foreground'
                                                        : 'bg-muted text-muted-foreground hover:bg-muted/80'}`}
                                            >
                                                {t(`roles.${role}`)}
                                            </button>
                                        ))}
                                        <button onClick={() => setEditingId(null)}
                                            className="px-2 py-1 rounded-lg text-xs text-muted-foreground hover:bg-muted">
                                            {t('cancel')}
                                        </button>
                                    </div>
                                ) : (
                                    <button onClick={() => setEditingId(user.id)}
                                        className="px-3 py-1.5 rounded-xl text-xs font-medium bg-muted text-muted-foreground hover:bg-muted/80 transition-all active:scale-95">
                                        {t('changeRole')}
                                    </button>
                                )}
                                <button onClick={() => handleDelete(user.id)}
                                    disabled={user.id === 1}
                                    className="p-1.5 rounded-xl text-muted-foreground hover:text-red-500 hover:bg-red-500/10 transition-all active:scale-95 disabled:opacity-30 disabled:cursor-not-allowed"
                                    title={user.id === 1 ? t('cannotDeletePrimaryAdmin') : t('deleteUser')}>
                                    <Trash2 className="h-4 w-4" />
                                </button>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        </PageWrapper>
    );
}
