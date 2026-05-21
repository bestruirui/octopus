'use client';

import { useState } from "react"
import { AnimatePresence, motion } from "motion/react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useLogin, useAuthStore } from "@/api/endpoints/user"
import { useAPIKeyLogin } from "@/api/endpoints/apikey"
import { TOTPVerify } from "./TOTPVerify"
import Logo from "@/components/modules/logo"
import { KeyRound, User } from "lucide-react"
import {
  Tabs,
  TabsList,
  TabsHighlight,
  TabsHighlightItem,
  TabsTrigger,
  TabsContents,
  TabsContent,
} from "@/components/animate-ui/primitives/animate/tabs"

type LoginMode = 'user' | 'apikey';
type LoginStep = 'credentials' | 'totp';

export function LoginForm({ onLoginSuccess }: { onLoginSuccess?: () => void }) {
  const t = useTranslations('login')
  const totpTranslations = useTranslations('totp')
  const [mode, setMode] = useState<LoginMode>('user')
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [apiKey, setApiKey] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [loginStep, setLoginStep] = useState<LoginStep>('credentials')
  const [tempToken, setTempToken] = useState<string>("")
  const [isVerifyingTOTP, setIsVerifyingTOTP] = useState(false)

  const loginMutation = useLogin()
  const apiKeyLoginMutation = useAPIKeyLogin()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      if (mode === 'user') {
        // 尝试使用密码登录
        const result = await loginMutation.mutateAsync({
          username,
          password,
          expire: 86400,
        })

        // 检查是否返回了 2FA 需要的响应
        if (result && typeof result === 'object' && '2fa_required' in result && result['2fa_required']) {
          const totpResult = result as unknown as { temp_token: string; expire_at: string };
          setTempToken(totpResult.temp_token)
          setLoginStep('totp')
          return
        }

        onLoginSuccess?.()
      } else {
        await apiKeyLoginMutation.mutateAsync(apiKey)
        onLoginSuccess?.()
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : totpTranslations('genericError')
      setError(errorMessage)
    }
  }

  const handleTOTPVerifySuccess = (token: string, expireAt: string) => {
    const { setAuth } = useAuthStore.getState()
    setAuth(token, expireAt)
    onLoginSuccess?.()
  }

  const handleBackToLogin = () => {
    setLoginStep('credentials')
    setTempToken("")
    setError(null)
  }

  const isPending = loginMutation.isPending || apiKeyLoginMutation.isPending

  const handleModeChange = (value: string) => {
    setMode(value as LoginMode)
    setError(null)
  }

  if (loginStep === 'totp') {
    return (
      <TOTPVerify
        tempToken={tempToken}
        onVerifySuccess={handleTOTPVerifySuccess}
        onBack={handleBackToLogin}
        isVerifying={isVerifyingTOTP}
      />
    )
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

        <Tabs value={mode} onValueChange={handleModeChange}>
          <TabsList className="flex p-1 bg-muted rounded-2xl">
            <TabsHighlight className="rounded-xl bg-background shadow-sm">
              <TabsHighlightItem value="user" className="flex-1">
                <TabsTrigger
                  value="user"
                  className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-medium transition-colors data-[state=active]:text-foreground data-[state=inactive]:text-muted-foreground data-[state=inactive]:hover:text-foreground"
                >
                  <User className="w-4 h-4" />
                  {t('mode.user')}
                </TabsTrigger>
              </TabsHighlightItem>
              <TabsHighlightItem value="apikey" className="flex-1">
                <TabsTrigger
                  value="apikey"
                  className="w-full flex items-center justify-center gap-2 py-2.5 px-4 rounded-xl text-sm font-medium transition-colors data-[state=active]:text-foreground data-[state=inactive]:text-muted-foreground data-[state=inactive]:hover:text-foreground"
                >
                  <KeyRound className="w-4 h-4" />
                  {t('mode.apikey')}
                </TabsTrigger>
              </TabsHighlightItem>
            </TabsHighlight>
          </TabsList>

          <form onSubmit={handleSubmit} className="space-y-6 pt-2">
            <TabsContents className="p-3 -mx-3 py-6">
              <TabsContent value="user" className="space-y-6">
                <Field>
                  <FieldLabel htmlFor="username">{t('username')}</FieldLabel>
                  <Input
                    id="username"
                    type="text"
                    placeholder={t('usernamePlaceholder')}
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required={mode === 'user'}
                    disabled={isPending}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="password">{t('password')}</FieldLabel>
                  <Input
                    id="password"
                    type="password"
                    placeholder={t('passwordPlaceholder')}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required={mode === 'user'}
                    disabled={isPending}
                  />
                </Field>
              </TabsContent>
              <TabsContent value="apikey">
                <Field>
                  <FieldLabel htmlFor="apikey">{t('apikey')}</FieldLabel>
                  <Input
                    id="apikey"
                    type="password"
                    placeholder={t('apikeyPlaceholder')}
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    required={mode === 'apikey'}
                    disabled={isPending}
                  />
                </Field>
              </TabsContent>
            </TabsContents>

            {error && <FieldDescription className="text-destructive">{error}</FieldDescription>}

            <Button type="submit" disabled={isPending} className="w-full">
              {isPending ? t('button.loading') : t('button.submit')}
            </Button>
          </form>
        </Tabs>
      </div>
    </motion.div>
  )
}
