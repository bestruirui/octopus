'use client';

import { useState } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useLogin } from "@/api/endpoints/user"
import Logo from "@/components/modules/logo"

export function LoginForm({ onLoginSuccess }: { onLoginSuccess?: () => void }) {
  const t = useTranslations('login')
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)

  const loginMutation = useLogin()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      await loginMutation.mutateAsync({
        username,
        password,
        expire: 86400,
      })

      onLoginSuccess?.()
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : t('error.generic')
      setError(errorMessage)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center px-6 text-foreground">
      <div className="w-full max-w-sm space-y-8">
        <header className="flex flex-col items-center gap-3">
          <Logo />
          <h1 className="text-2xl font-bold">Octopus</h1>
        </header>

        <form onSubmit={handleSubmit} className="space-y-6">
          <Field>
            <FieldLabel htmlFor="username">{t('username')}</FieldLabel>
            <Input
              id="username"
              type="text"
              placeholder={t('usernamePlaceholder')}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              disabled={loginMutation.isPending}
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
              required
              disabled={loginMutation.isPending}
            />
          </Field>

          {error && <FieldDescription className="text-destructive">{error}</FieldDescription>}

          <Button type="submit" disabled={loginMutation.isPending} className="w-full">
            {loginMutation.isPending ? t('button.loading') : t('button.submit')}
          </Button>
        </form>
      </div>
    </div>
  )
}
