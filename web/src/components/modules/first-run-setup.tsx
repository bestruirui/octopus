'use client';

import { useEffect, useMemo, useState } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ShieldAlert, RefreshCw } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { apiClient } from '@/api/client';
import type { BootstrapCreateAdminRequest, BootstrapStatusResponse } from '@/api/endpoints/bootstrap';
import { HttpStatus, type ApiError } from '@/api/types';
import { toast } from '@/components/common/Toast';
import Logo from '@/components/modules/logo';
import { Button } from '@/components/ui/button';
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { ParticleBackground } from '@/components/nature';
import { useIsMobile } from '@/hooks/use-mobile';

function isApiError(error: unknown): error is ApiError {
  return typeof error === 'object' && error !== null && 'code' in error && 'message' in error;
}

export function FirstRunSetup() {
  const t = useTranslations('bootstrap');
  const queryClient = useQueryClient();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorText, setErrorText] = useState<string | null>(null);
  const [setupComplete, setSetupComplete] = useState(false);
  const isMobile = useIsMobile();

  const { data, isLoading, refetch, error } = useQuery({
    queryKey: ['bootstrap', 'status'],
    queryFn: async () => apiClient.get<BootstrapStatusResponse>('/api/v1/bootstrap/status', undefined, false),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (setupComplete && data?.initialized) {
      window.location.assign('/');
    }
  }, [setupComplete, data?.initialized]);

  const createAdminMutation = useMutation({
    mutationFn: async (payload: BootstrapCreateAdminRequest) =>
      apiClient.post<{ initialized: boolean }>('/api/v1/bootstrap/create-admin', payload, undefined, false),
    onSuccess: async () => {
      setErrorText(null);
      setSetupComplete(true);
      toast.success(t('actions.submitSuccess'));
      await queryClient.invalidateQueries({ queryKey: ['bootstrap', 'status'] });
      await refetch();
    },
    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : t('error.generic');
      setErrorText(message);
      toast.error(message);
    },
  });

  const errorMessage = useMemo(() => {
    if (!error) return null;
    if (isApiError(error)) {
      if (error.code === HttpStatus.INTERNAL_SERVER_ERROR) {
        return t('error.server');
      }
      return error.message || t('error.generic');
    }
    if (error instanceof Error && error.message) {
      return error.message;
    }
    return t('error.generic');
  }, [error, t]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorText(null);
    await createAdminMutation.mutateAsync({
      username,
      password,
    });
  };

  const isPending = createAdminMutation.isPending;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
      className="relative min-h-screen flex items-center justify-center px-6 py-10 text-foreground overflow-hidden"
    >
      {/* Nature: 粒子背景 */}
      {!isMobile && <ParticleBackground count={40} minOpacity={0.08} maxOpacity={0.25} />}
      
      {/* 氛围光 */}
      <div className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-primary/10 blur-[120px] opacity-50" />

      <div className="relative z-10 w-full max-w-2xl">
        <div className="waterhouse-island-fx flex flex-col gap-8 p-8 md:p-10 border-border/35 bg-background/50 shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] rounded-[2.5rem]">
          <header className="flex flex-col md:flex-row items-center md:items-start gap-5 md:gap-6 border-b border-border/30 pb-6">
            <div className="waterhouse-pod grid size-16 shrink-0 place-items-center overflow-hidden rounded-[1.65rem] border-border/35 bg-background/58 shadow-waterhouse-soft">
              <Logo size={48} />
            </div>
            <div className="flex flex-col items-center md:items-start gap-1.5 text-center md:text-left mt-1 md:mt-0">
              <h1 className="text-2xl md:text-3xl font-bold tracking-tight">{t('title')}</h1>
              <p className="text-sm text-muted-foreground/80">{t('subtitle')}</p>
            </div>
          </header>

          <div className="space-y-6">
            <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-900 dark:text-amber-100 shadow-inner">
              <div className="flex items-start gap-3">
                <ShieldAlert className="mt-0.5 size-5 shrink-0 text-amber-500" />
                <div className="space-y-1.5">
                  <p className="font-semibold">{t('notice.title')}</p>
                  <p className="text-amber-900/80 dark:text-amber-100/80 leading-relaxed">{t('notice.description')}</p>
                </div>
              </div>
            </div>

            <AnimatePresence mode="wait">
              {isLoading ? (
                <motion.div
                  key="loading"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="flex items-center justify-center gap-3 py-8 text-sm text-muted-foreground"
                >
                  <RefreshCw className="size-5 animate-spin text-primary" />
                  <span className="tracking-wide">{t('checking')}</span>
                </motion.div>
              ) : data?.initialized && !setupComplete ? (
                <motion.div
                  key="initialized"
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-5 text-center text-sm font-medium text-emerald-900 dark:text-emerald-100 shadow-sm"
                >
                  {t('initialized')}
                </motion.div>
              ) : (
                <motion.form
                  key="form"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="space-y-5"
                  onSubmit={handleSubmit}
                >
                  <Field>
                    <FieldLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1" htmlFor="bootstrap-username">{t('form.username')}</FieldLabel>
                    <Input
                      id="bootstrap-username"
                      type="text"
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      placeholder={t('form.usernamePlaceholder')}
                      className="h-12 rounded-xl bg-background/40 border-border/30 focus-visible:ring-primary/40 focus-visible:border-primary/50"
                      disabled={isPending}
                      required
                    />
                  </Field>

                  <Field>
                    <FieldLabel className="text-xs font-semibold uppercase tracking-wider text-muted-foreground/70 ml-1" htmlFor="bootstrap-password">{t('form.password')}</FieldLabel>
                    <Input
                      id="bootstrap-password"
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder={t('form.passwordPlaceholder')}
                      className="h-12 rounded-xl bg-background/40 border-border/30 focus-visible:ring-primary/40 focus-visible:border-primary/50"
                      disabled={isPending}
                      required
                    />
                    <FieldDescription className="ml-1 mt-1 text-muted-foreground/60">{t('form.passwordHint')}</FieldDescription>
                  </Field>

                  <p className="text-sm text-muted-foreground/70 ml-1">
                    {data?.message || t('description')}
                  </p>

                  {errorText && (
                    <motion.div initial={{ opacity: 0, y: -5 }} animate={{ opacity: 1, y: 0 }}>
                      <FieldDescription className="text-destructive font-medium text-xs bg-destructive/5 p-3 rounded-xl border border-destructive/10">
                        {errorText}
                      </FieldDescription>
                    </motion.div>
                  )}

                  <div className="pt-4 flex flex-col sm:flex-row gap-3">
                    <Button 
                      type="submit" 
                      className="flex-1 h-12 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90 shadow-nature-glow transition-all active:scale-[0.98]" 
                      disabled={isPending}
                    >
                      {isPending ? t('actions.submitting') : t('actions.submit')}
                    </Button>
                    <Button 
                      type="button"
                      onClick={() => void refetch()} 
                      variant="outline" 
                      className="h-12 px-6 rounded-xl border-border/30 bg-background/30 hover:bg-background/50 text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <RefreshCw className={`mr-2 size-4 ${isLoading ? 'animate-spin' : ''}`} />
                      {t('actions.refresh')}
                    </Button>
                  </div>
                </motion.form>
              )}
            </AnimatePresence>

            {errorMessage && (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="mt-4 rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive shadow-sm">
                {errorMessage}
              </motion.div>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  );
}
