'use client';

import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ShieldAlert, RefreshCw } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { apiClient } from '@/api/client';
import type { BootstrapCreateAdminRequest, BootstrapStatusResponse } from '@/api/endpoints/bootstrap';
import { HttpStatus, type ApiError } from '@/api/types';
import { toast } from '@/components/common/Toast';
import Logo from '@/components/modules/logo';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';

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
    <div className="min-h-screen flex items-center justify-center px-6 py-10 text-foreground">
      <Card className="w-full max-w-2xl rounded-3xl border-border/60 shadow-lg">
        <CardHeader className="space-y-4">
          <div className="flex items-center gap-3">
            <Logo size={40} />
            <div>
              <CardTitle className="text-2xl">{t('title')}</CardTitle>
              <CardDescription>{t('subtitle')}</CardDescription>
            </div>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-900 dark:text-amber-100">
            <div className="flex items-start gap-3">
              <ShieldAlert className="mt-0.5 size-5 shrink-0" />
              <div className="space-y-2">
                <p className="font-medium">{t('notice.title')}</p>
                <p>{t('notice.description')}</p>
              </div>
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <RefreshCw className="size-4 animate-spin" />
              <span>{t('checking')}</span>
            </div>
          ) : data?.initialized && !setupComplete ? (
            <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-900 dark:text-emerald-100">
              {t('initialized')}
            </div>
          ) : (
            <form className="space-y-4" onSubmit={handleSubmit}>
              <Field>
                <FieldLabel htmlFor="bootstrap-username">{t('form.username')}</FieldLabel>
                <Input
                  id="bootstrap-username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder={t('form.usernamePlaceholder')}
                  disabled={isPending}
                  required
                />
              </Field>

              <Field>
                <FieldLabel htmlFor="bootstrap-password">{t('form.password')}</FieldLabel>
                <Input
                  id="bootstrap-password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('form.passwordPlaceholder')}
                  disabled={isPending}
                  required
                />
                <FieldDescription>{t('form.passwordHint')}</FieldDescription>
              </Field>

              <p className="text-sm text-muted-foreground">
                {data?.message || t('description')}
              </p>

              {errorText && <FieldDescription className="text-destructive">{errorText}</FieldDescription>}

              <Button type="submit" className="w-full rounded-xl" disabled={isPending}>
                {isPending ? t('actions.submitting') : t('actions.submit')}
              </Button>
            </form>
          )}

          {errorMessage && (
            <div className="rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
              {errorMessage}
            </div>
          )}
        </CardContent>

        <CardFooter className="flex flex-wrap gap-3">
          <Button onClick={() => void refetch()} variant="outline" className="rounded-xl">
            <RefreshCw className="mr-2 size-4" />
            {t('actions.refresh')}
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}
