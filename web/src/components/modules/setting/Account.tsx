'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { User, KeyRound, Lock, Eye, EyeOff } from 'lucide-react';
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

    return (
        <div className="rounded-3xl border border-border bg-card p-6 custom-shadow space-y-6">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <User className="h-5 w-5" />
                {t('account.title')}
            </h2>

            {/* 修改用户名 */}
            <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <KeyRound className="size-4" />
                    {t('account.username.label')}
                </div>
                <div className="flex gap-2">
                    <Input
                        value={newUsername}
                        onChange={(e) => setNewUsername(e.target.value)}
                        placeholder={t('account.username.placeholder')}
                        className="flex-1 rounded-xl"
                    />
                    <Button
                        onClick={handleChangeUsername}
                        disabled={changeUsername.isPending || !newUsername.trim()}
                        className="rounded-xl"
                    >
                        {changeUsername.isPending ? t('account.saving') : t('account.save')}
                    </Button>
                </div>
            </div>

            <div className="border-t border-border" />

            {/* 修改密码 */}
            <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <Lock className="size-4" />
                    {t('account.password.label')}
                </div>
                <div className="space-y-2">
                    <div className="relative">
                        <Input
                            type={showOldPassword ? 'text' : 'password'}
                            value={oldPassword}
                            onChange={(e) => setOldPassword(e.target.value)}
                            placeholder={t('account.password.oldPlaceholder')}
                            className="rounded-xl pr-10"
                        />
                        <button
                            type="button"
                            onClick={() => setShowOldPassword(!showOldPassword)}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
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
                            className="rounded-xl pr-10"
                        />
                        <button
                            type="button"
                            onClick={() => setShowNewPassword(!showNewPassword)}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
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
                            className="rounded-xl pr-10"
                        />
                        <button
                            type="button"
                            onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        >
                            {showConfirmPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                        </button>
                    </div>
                    <Button
                        onClick={handleChangePassword}
                        disabled={changePassword.isPending || !oldPassword || !newPassword || !confirmPassword}
                        className="w-full rounded-xl"
                    >
                        {changePassword.isPending ? t('account.saving') : t('account.password.change')}
                    </Button>
                </div>
            </div>
        </div>
    );
}

