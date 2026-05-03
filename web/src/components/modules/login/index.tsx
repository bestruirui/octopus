'use client';

import { useState } from "react"
import { motion, AnimatePresence } from "motion/react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useLogin } from "@/api/endpoints/user"
import { useAPIKeyLogin } from "@/api/endpoints/apikey"
import Logo from "@/components/modules/logo"
import { KeyRound, User } from "lucide-react"
import { ParticleBackground } from "@/components/nature"
import { useIsMobile } from "@/hooks/use-mobile"
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

export function LoginForm({ onLoginSuccess }: { onLoginSuccess?: () => void }) {
  const t = useTranslations('login')
  const [mode, setMode] = useState<LoginMode>('user')
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [apiKey, setApiKey] = useState("")
  const [error, setError] = useState<string | null>(null)
  const isMobile = useIsMobile()

  const loginMutation = useLogin()
  const apiKeyLoginMutation = useAPIKeyLogin()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    try {
      if (mode === 'user') {
        await loginMutation.mutateAsync({
          username: username.trim(),
          password,
          expire: 1440,
        })
      } else {
        await apiKeyLoginMutation.mutateAsync(apiKey)
      }

      onLoginSuccess?.()
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : t('error.generic')
      setError(errorMessage)
    }
  }

  const isPending = loginMutation.isPending || apiKeyLoginMutation.isPending

  const handleModeChange = (value: string) => {
    setMode(value as LoginMode)
    setError(null)
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
      className="relative min-h-screen flex items-center justify-center px-6 text-foreground overflow-hidden"
    >
      {/* Nature: 粒子背景 */}
      {!isMobile && <ParticleBackground count={40} minOpacity={0.08} maxOpacity={0.25} />}
      
      {/* 氛围光 */}
      <div className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-primary/10 blur-[120px] opacity-50" />

      <div className="relative z-10 w-full max-w-sm">
        <div className="waterhouse-island-fx flex flex-col gap-8 p-8 md:p-10 border-border/35 bg-background/50 shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] rounded-[2.5rem]">
          <header className="flex flex-col items-center gap-4">
            <div className="waterhouse-pod grid size-16 shrink-0 place-items-center overflow-hidden rounded-[1.65rem] border-border/35 bg-background/58 shadow-waterhouse-soft">
              <Logo size={48} />
            </div>
            <div className="flex flex-col items-center gap-1">
              <h1 className="text-3xl font-bold tracking-tight">Octopus</h1>
              <p className="text-sm text-muted-foreground/80 tracking-wide uppercase font-medium">{t('welcome') || 'Welcome back'}</p>
            </div>
          </header>

          <Tabs value={mode} onValueChange={handleModeChange}>
            <TabsList className="flex p-1 bg-muted/50 rounded-2xl border border-border/20">
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

            <form onSubmit={handleSubmit} className="space-y-6 pt-4">
              <AnimatePresence mode="wait">
                <motion.div
                  key={mode}
                  initial={{ opacity: 0, x: 10 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -10 }}
                  transition={{ duration: 0.2, ease: "easeOut" }}
                >
                  <TabsContents>
                    <TabsContent value="user" className="space-y-5">
                      <Field>
                        <FieldLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1" htmlFor="username">{t('username')}</FieldLabel>
                        <Input
                          id="username"
                          type="text"
                          placeholder={t('usernamePlaceholder')}
                          value={username}
                          onChange={(e) => setUsername(e.target.value)}
                          className="h-12 rounded-xl bg-background/40 border-border/30 focus-visible:ring-primary/40 focus-visible:border-primary/50"
                          autoComplete="username"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          required={mode === 'user'}
                          disabled={isPending}
                        />
                      </Field>
                      <Field>
                        <FieldLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1" htmlFor="password">{t('password')}</FieldLabel>
                        <Input
                          id="password"
                          type="password"
                          placeholder={t('passwordPlaceholder')}
                          value={password}
                          onChange={(e) => setPassword(e.target.value)}
                          className="h-12 rounded-xl bg-background/40 border-border/30 focus-visible:ring-primary/40 focus-visible:border-primary/50"
                          autoComplete="current-password"
                          required={mode === 'user'}
                          disabled={isPending}
                        />
                      </Field>
                    </TabsContent>
                    <TabsContent value="apikey">
                      <Field>
                        <FieldLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1" htmlFor="apikey">{t('apikey')}</FieldLabel>
                        <Input
                          id="apikey"
                          type="password"
                          placeholder={t('apikeyPlaceholder')}
                          value={apiKey}
                          onChange={(e) => setApiKey(e.target.value)}
                          className="h-12 rounded-xl bg-background/40 border-border/30 focus-visible:ring-primary/40 focus-visible:border-primary/50"
                          autoComplete="off"
                          autoCapitalize="none"
                          autoCorrect="off"
                          spellCheck={false}
                          required={mode === 'apikey'}
                          disabled={isPending}
                        />
                      </Field>
                    </TabsContent>
                  </TabsContents>
                </motion.div>
              </AnimatePresence>

              {error && (
                <motion.div
                  initial={{ opacity: 0, y: -5 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="px-1"
                >
                  <FieldDescription className="text-destructive font-medium text-xs bg-destructive/5 p-2 rounded-lg border border-destructive/10">
                    {error}
                  </FieldDescription>
                </motion.div>
              )}

              <Button 
                type="submit" 
                disabled={isPending} 
                className="w-full h-12 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 shadow-nature-glow transition-all active:scale-[0.98]"
              >
                {isPending ? t('button.loading') : t('button.submit')}
              </Button>
            </form>
          </Tabs>
        </div>
      </div>
    </motion.div>
  )
}
