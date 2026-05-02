'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { User, KeyRound, Lock, Eye, EyeOff, LogOut } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useChangeUsername, useChangePassword, useAuth } from '@/api/endpoints/user';
import { toast } from '@/components/common/Toast';

export function SettingAccount() {
    const t = useTranslations('setting');
    const { logout } = useAuth();
    const changeUsername = useChangeUsername();
    const changePassword = useChangePassword();

    const [newUsername, setNewUsername] = useState('');
    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');

    const [showOldPassword, setShowOldPassword] = useState(false);
    const [showNewPassword, setShowNewPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);

    const handleChangeUsername = () => {
        if (!newUsername.trim()) {
            toast.error(t('account.username.empty'));
            return;
        }

        changeUsername.mutate(
            { newUsername: newUsername.trim() },
            {
                onSuccess: () => {
                    toast.success(t('account.username.success'));
                    setTimeout(() => logout(), 1000);
                },
                onError: () => {
                    toast.error(t('account.username.failed'));
                },
            }
        );
    };

    const handleChangePassword = () => {
        if (!oldPassword) {
            toast.error(t('account.password.oldEmpty'));
            return;
        }
        if (!newPassword) {
            toast.error(t('account.password.newEmpty'));
            return;
        }
        if (newPassword !== confirmPassword) {
            toast.error(t('account.password.mismatch'));
            return;
        }
        if (newPassword.length < 6) {
            toast.error(t('account.password.tooShort'));
            return;
        }

        changePassword.mutate(
            { oldPassword, newPassword },
            {
                onSuccess: () => {
                    toast.success(t('account.password.success'));
                    setTimeout(() => logout(), 1000);
                },
                onError: () => {
                    toast.error(t('account.password.failed'));
                },
            }
        );
    };

    const sectionClassName = 'waterhouse-pod relative overflow-hidden rounded-[2rem] border border-border/30 bg-background/36 p-5 shadow-waterhouse-soft';

    return (
        <div className="waterhouse-island relative overflow-hidden rounded-[2.25rem] border-border/35 bg-card/62 p-6 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
            <div className="space-y-5">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <div className="space-y-1.5">
                        <h2 className="flex items-center gap-2 text-lg font-bold text-card-foreground">
                            <User className="h-5 w-5" />
                            {t('account.title')}
                        </h2>
                        <p className="text-sm text-muted-foreground">{t('account.logout.label')}</p>
                    </div>
                </div>

                <div className={sectionClassName}>
                    <div className="mb-4 flex items-start justify-between gap-4">
                        <div className="flex items-start gap-3">
                            <span className="grid size-9 shrink-0 place-items-center rounded-[1.1rem] bg-primary/12 text-xs font-semibold text-primary shadow-waterhouse-soft">
                                01
                            </span>
                            <div className="space-y-1">
                                <div className="flex items-center gap-2 text-sm font-medium text-card-foreground">
                                    <KeyRound className="size-4 text-muted-foreground" />
                                    {t('account.username.label')}
                                </div>
                                <p className="text-xs text-muted-foreground">{t('account.username.placeholder')}</p>
                            </div>
                        </div>
                        <Button
                            onClick={handleChangeUsername}
                            disabled={changeUsername.isPending || !newUsername.trim()}
                            className="hidden rounded-[1.2rem] lg:inline-flex"
                        >
                            {changeUsername.isPending ? t('account.saving') : t('account.save')}
                        </Button>
                    </div>

                    <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]">
                        <Input
                            value={newUsername}
                            onChange={(e) => setNewUsername(e.target.value)}
                            placeholder={t('account.username.placeholder')}
                            className="rounded-[1.3rem]"
                        />
                        <Button
                            onClick={handleChangeUsername}
                            disabled={changeUsername.isPending || !newUsername.trim()}
                            className="rounded-[1.3rem] lg:hidden"
                        >
                            {changeUsername.isPending ? t('account.saving') : t('account.save')}
                        </Button>
                    </div>
                </div>

                <div className={sectionClassName}>
                    <div className="mb-4 flex items-start gap-3">
                        <span className="grid size-9 shrink-0 place-items-center rounded-[1.1rem] bg-primary/12 text-xs font-semibold text-primary shadow-waterhouse-soft">
                            02
                        </span>
                        <div className="space-y-1">
                            <div className="flex items-center gap-2 text-sm font-medium text-card-foreground">
                                <Lock className="size-4 text-muted-foreground" />
                                {t('account.password.label')}
                            </div>
                            <p className="text-xs text-muted-foreground">{t('account.password.change')}</p>
                        </div>
                    </div>

                    <div className="grid gap-3 xl:grid-cols-2">
                        <div className="relative xl:col-span-2">
                            <Input
                                type={showOldPassword ? 'text' : 'password'}
                                value={oldPassword}
                                onChange={(e) => setOldPassword(e.target.value)}
                                placeholder={t('account.password.oldPlaceholder')}
                                className="rounded-[1.3rem] pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => setShowOldPassword(!showOldPassword)}
                                aria-label={showOldPassword ? t('account.password.hideOld') : t('account.password.showOld')}
                                aria-pressed={showOldPassword}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                            >
                                {showOldPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                            </button>
                        </div>
                        <div className="relative">
                            <Input
                                type={showNewPassword ? 'text' : 'password'}
                                value={newPassword}
                                onChange={(e) => setNewPassword(e.target.value)}
                                placeholder={t('account.password.newPlaceholder')}
                                className="rounded-[1.3rem] pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => setShowNewPassword(!showNewPassword)}
                                aria-label={showNewPassword ? t('account.password.hideNew') : t('account.password.showNew')}
                                aria-pressed={showNewPassword}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                            >
                                {showNewPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                            </button>
                        </div>
                        <div className="relative">
                            <Input
                                type={showConfirmPassword ? 'text' : 'password'}
                                value={confirmPassword}
                                onChange={(e) => setConfirmPassword(e.target.value)}
                                placeholder={t('account.password.confirmPlaceholder')}
                                className="rounded-[1.3rem] pr-10"
                            />
                            <button
                                type="button"
                                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                aria-label={showConfirmPassword ? t('account.password.hideConfirm') : t('account.password.showConfirm')}
                                aria-pressed={showConfirmPassword}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                            >
                                {showConfirmPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                            </button>
                        </div>
                    </div>

                    <div className="mt-4 flex justify-end">
                        <Button
                            onClick={handleChangePassword}
                            disabled={changePassword.isPending || !oldPassword || !newPassword || !confirmPassword}
                            className="w-full rounded-[1.3rem] sm:w-auto sm:min-w-36"
                        >
                            {changePassword.isPending ? t('account.saving') : t('account.password.change')}
                        </Button>
                    </div>
                </div>

                <div className="waterhouse-pod relative overflow-hidden rounded-[2rem] border border-destructive/20 bg-destructive/6 p-5 shadow-waterhouse-soft">
                    <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                        <div className="flex items-start gap-3">
                            <span className="grid size-9 shrink-0 place-items-center rounded-[1.1rem] bg-destructive/12 text-xs font-semibold text-destructive shadow-waterhouse-soft">
                                03
                            </span>
                            <div className="space-y-1">
                                <div className="flex items-center gap-2 text-sm font-medium text-card-foreground">
                                    <LogOut className="size-4 text-destructive" />
                                    {t('account.logout.label')}
                                </div>
                                <p className="text-xs text-muted-foreground">{t('account.logout.button')}</p>
                            </div>
                        </div>
                        <Button
                            variant="destructive"
                            size="sm"
                            onClick={logout}
                            className="rounded-[1.3rem] sm:min-w-32"
                        >
                            {t('account.logout.button')}
                        </Button>
                    </div>
                </div>
            </div>
        </div>
    );
}

