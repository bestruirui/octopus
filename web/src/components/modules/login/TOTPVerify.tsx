'use client';

import { useState } from "react"
import { motion } from "motion/react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import Logo from "@/components/modules/logo"
import { Shield } from "lucide-react"

interface TOTPVerifyProps {
    tempToken: string;
    onVerifySuccess: (token: string, expireAt: string) => void;
    onBack: () => void;
    isVerifying?: boolean;
}

export function TOTPVerify({ tempToken, onVerifySuccess, onBack, isVerifying }: TOTPVerifyProps) {
    const t = useTranslations('totp')
    const [code, setCode] = useState("")
    const [error, setError] = useState<string | null>(null)

    // 对验证码输入做格式化
    const handleCodeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = e.target.value.replace(/[^0-9]/g, '').slice(0, 6)
        setCode(value)
        setError(null)
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setError(null)

        if (code.length !== 6) {
            setError(t('codeLengthError'))
            return
        }

        try {
            const response = await fetch('/api/v1/user/login/verify-2fa', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${tempToken}`,
                },
                body: JSON.stringify({
                    temp_token: tempToken,
                    totp_code: code,
                    expire: 86400,
                }),
            })

            const data = await response.json()

            if (!response.ok) {
                throw new Error(data.message || t('verifyError'))
            }

            onVerifySuccess(data.data.token, data.data.expire_at)
        } catch (err: unknown) {
            const errorMessage = err instanceof Error ? err.message : t('genericError')
            setError(errorMessage)
        }
    }

    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
            className="min-h-screen flex items-center justify-center px-6 text-foreground"
        >
            <div className="w-full max-w-sm space-y-8">
                <header className="flex flex-col items-center gap-3">
                    <Logo size={48} />
                    <h1 className="text-2xl font-bold">Octopus</h1>
                </header>

                <form onSubmit={handleSubmit} className="space-y-6 pt-2">
                    <div className="text-center space-y-2">
                        <div className="flex justify-center">
                            <Shield className="w-12 h-12 text-primary" />
                        </div>
                        <p className="text-lg font-medium">{t('title')}</p>
                        <p className="text-sm text-muted-foreground">{t('description')}</p>
                    </div>

                    <Field>
                        <FieldLabel htmlFor="totp-code">{t('codeLabel')}</FieldLabel>
                        <Input
                            id="totp-code"
                            type="text"
                            inputMode="numeric"
                            autoComplete="one-time-code"
                            placeholder="000000"
                            value={code}
                            onChange={handleCodeChange}
                            required
                            disabled={isVerifying}
                            maxLength={6}
                            className="text-center text-2xl tracking-[0.5em] font-mono"
                        />
                    </Field>

                    {error && <FieldDescription className="text-destructive text-center">{error}</FieldDescription>}

                    <Button type="submit" disabled={isVerifying || code.length !== 6} className="w-full">
                        {isVerifying ? t('verifying') : t('verify')}
                    </Button>

                    <div className="text-center">
                        <button
                            type="button"
                            onClick={onBack}
                            className="text-sm text-muted-foreground hover:text-foreground transition-colors"
                            disabled={isVerifying}
                        >
                            {t('back')}
                        </button>
                    </div>
                </form>
            </div>
        </motion.div>
    )
}
