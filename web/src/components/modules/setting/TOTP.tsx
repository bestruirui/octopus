'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import { Shield, ShieldOff, Smartphone, Copy, Check, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field';
import { useTOTPStatus, useTOTPInit, useTOTPVerifySetup, useTOTPDisable } from '@/api/endpoints/totp';
import { toast } from '@/components/common/Toast';
import { Switch } from '@/components/ui/switch';

export function SettingTOTP() {
    const t = useTranslations('setting.totp');
    const { data: status, isLoading: statusLoading } = useTOTPStatus();
    const totpInit = useTOTPInit();
    const totpVerifySetup = useTOTPVerifySetup();
    const totpDisable = useTOTPDisable();

    const [setupStep, setSetupStep] = useState<'idle' | 'init' | 'verify'>('idle');
    const [secret, setSecret] = useState('');
    const [uri, setUri] = useState('');
    const [verifyCode, setVerifyCode] = useState('');
    const [disableCode, setDisableCode] = useState('');
    const [copied, setCopied] = useState(false);

    const isEnabled = status?.enabled ?? false;

    const handleEnable = async () => {
        try {
            const result = await totpInit.mutateAsync();
            setSecret(result.secret);
            setUri(result.uri);
            setSetupStep('init');
        } catch {
            toast.error(t('initError'));
        }
    };

    const handleVerifySetup = async () => {
        if (verifyCode.length !== 6) {
            toast.error(t('codeLengthError'));
            return;
        }
        try {
            await totpVerifySetup.mutateAsync({ code: verifyCode });
            toast.success(t('enableSuccess'));
            setSetupStep('idle');
            setVerifyCode('');
            setSecret('');
            setUri('');
        } catch {
            toast.error(t('verifyError'));
        }
    };

    const handleDisable = async () => {
        if (disableCode.length !== 6) {
            toast.error(t('codeLengthError'));
            return;
        }
        try {
            await totpDisable.mutateAsync({ code: disableCode });
            toast.success(t('disableSuccess'));
            setDisableCode('');
        } catch {
            toast.error(t('disableError'));
        }
    };

    const handleCopySecret = async () => {
        try {
            await navigator.clipboard.writeText(secret);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
            toast.success(t('copySuccess'));
        } catch {
            toast.error(t('copyFailed'));
        }
    };

    const handleCancelSetup = () => {
        setSetupStep('idle');
        setSecret('');
        setUri('');
        setVerifyCode('');
    };

    if (statusLoading) {
        return null;
    }

    // 显示正在设置流程
    if (setupStep === 'init' || setupStep === 'verify') {
        return (
            <div className="rounded-3xl border border-border bg-card p-6 space-y-6">
                <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                    <Shield className="h-5 w-5" />
                    {t('title')}
                </h2>

                {setupStep === 'init' && (
                    <div className="space-y-4">
                        <p className="text-sm text-muted-foreground">{t('setupStep1')}</p>

                        {/* 密钥 */}
                        <div className="space-y-2">
                            <FieldLabel>{t('secretLabel')}</FieldLabel>
                            <div className="flex gap-2">
                                <code className="flex-1 p-3 rounded-xl bg-muted text-xs font-mono break-all select-all">
                                    {secret}
                                </code>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    onClick={handleCopySecret}
                                    className="shrink-0 rounded-xl"
                                >
                                    {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                                </Button>
                            </div>
                        </div>

                        <p className="text-sm text-muted-foreground">{t('setupStep2')}</p>

                        <div className="flex gap-2">
                            <Button
                                onClick={() => setSetupStep('verify')}
                                disabled={totpInit.isPending}
                                className="flex-1 rounded-xl"
                            >
                                {t('continueButton')}
                            </Button>
                            <Button
                                variant="outline"
                                onClick={handleCancelSetup}
                                disabled={totpInit.isPending}
                                className="rounded-xl"
                            >
                                {t('cancelButton')}
                            </Button>
                        </div>
                    </div>
                )}

                {setupStep === 'verify' && (
                    <div className="space-y-4">
                        <p className="text-sm text-muted-foreground">{t('setupStep3')}</p>

                        <Field>
                            <FieldLabel htmlFor="totp-verify-code">{t('codeLabel')}</FieldLabel>
                            <Input
                                id="totp-verify-code"
                                type="text"
                                inputMode="numeric"
                                placeholder="000000"
                                value={verifyCode}
                                onChange={(e) => {
                                    const val = e.target.value.replace(/[^0-9]/g, '').slice(0, 6);
                                    setVerifyCode(val);
                                }}
                                maxLength={6}
                                className="text-center text-2xl tracking-[0.5em] font-mono"
                            />
                        </Field>

                        <div className="flex gap-2">
                            <Button
                                onClick={handleVerifySetup}
                                disabled={totpVerifySetup.isPending || verifyCode.length !== 6}
                                className="flex-1 rounded-xl"
                            >
                                {totpVerifySetup.isPending ? t('verifying') : t('verifyButton')}
                            </Button>
                            <Button
                                variant="outline"
                                onClick={() => setSetupStep('init')}
                                disabled={totpVerifySetup.isPending}
                                className="rounded-xl"
                            >
                                {t('backButton')}
                            </Button>
                        </div>
                    </div>
                )}
            </div>
        );
    }

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-6">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Shield className="h-5 w-5" />
                {t('title')}
            </h2>

            {/* 状态显示 */}
            <div className="flex items-center gap-3 p-3 rounded-xl bg-muted/50">
                {isEnabled ? (
                    <>
                        <Shield className="size-5 text-green-500" />
                        <span className="text-sm font-medium text-green-600 dark:text-green-400">{t('enabled')}</span>
                    </>
                ) : (
                    <>
                        <ShieldOff className="size-5 text-muted-foreground" />
                        <span className="text-sm font-medium text-muted-foreground">{t('disabled')}</span>
                    </>
                )}
            </div>

            {isEnabled ? (
                <div className="space-y-3">
                    <p className="text-sm text-muted-foreground">{t('disableHint')}</p>
                    <Field>
                        <FieldLabel htmlFor="totp-disable-code">{t('disableCodeLabel')}</FieldLabel>
                        <Input
                            id="totp-disable-code"
                            type="text"
                            inputMode="numeric"
                            placeholder="000000"
                            value={disableCode}
                            onChange={(e) => {
                                const val = e.target.value.replace(/[^0-9]/g, '').slice(0, 6);
                                setDisableCode(val);
                            }}
                            maxLength={6}
                            className="text-center text-2xl tracking-[0.5em] font-mono"
                        />
                    </Field>
                    <Button
                        variant="destructive"
                        onClick={handleDisable}
                        disabled={totpDisable.isPending || disableCode.length !== 6}
                        className="w-full rounded-xl"
                    >
                        {totpDisable.isPending ? t('disabling') : t('disableButton')}
                    </Button>
                </div>
            ) : (
                <div className="space-y-3">
                    <p className="text-sm text-muted-foreground">{t('enableHint')}</p>
                    <Button
                        onClick={handleEnable}
                        disabled={totpInit.isPending}
                        className="w-full rounded-xl"
                    >
                        <Smartphone className="size-4 mr-2" />
                        {totpInit.isPending ? t('initializing') : t('enableButton')}
                    </Button>
                </div>
            )}
        </div>
    );
}
