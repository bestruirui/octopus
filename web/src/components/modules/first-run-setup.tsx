'use client';

import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ShieldAlert, TerminalSquare, Copy, RefreshCw } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { apiClient } from '@/api/client';
import type { BootstrapStatusResponse } from '@/api/endpoints/bootstrap';
import { HttpStatus, type ApiError } from '@/api/types';
import { toast } from '@/components/common/Toast';
import Logo from '@/components/modules/logo';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';

const setupCommands = [
  'export OCTOPUS_INITIAL_ADMIN_USERNAME="admin"',
  'export OCTOPUS_INITIAL_ADMIN_PASSWORD="change-this-password-long"',
  'export OCTOPUS_AUTH_JWT_SECRET="replace-with-a-long-random-secret"',
  'go run main.go start',
];

function isApiError(error: unknown): error is ApiError {
  return typeof error === 'object' && error !== null && 'code' in error && 'message' in error;
}

export function FirstRunSetup() {
  const t = useTranslations('bootstrap');
  const [copied, setCopied] = useState(false);

  const { data, isLoading, refetch, error } = useQuery({
    queryKey: ['bootstrap', 'status'],
    queryFn: async () => apiClient.get<BootstrapStatusResponse>('/api/v1/bootstrap/status', undefined, false),
    retry: false,
    staleTime: 0,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1500);
    return () => window.clearTimeout(timer);
  }, [copied]);

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

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(setupCommands.join('\n'));
      setCopied(true);
      toast.success(t('actions.copied'));
    } catch {
      toast.error(t('actions.copyFailed'));
    }
  };

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
          ) : data?.initialized ? (
            <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-4 text-sm text-emerald-900 dark:text-emerald-100">
              {t('initialized')}
            </div>
          ) : (
            <div className="space-y-3">
              <div className="rounded-2xl border bg-muted/40 p-4">
                <div className="mb-3 flex items-center gap-2 text-sm font-medium">
                  <TerminalSquare className="size-4" />
                  <span>{t('commands.title')}</span>
                </div>
                <pre className="overflow-x-auto rounded-xl bg-background p-4 text-xs leading-6 text-muted-foreground">
                  <code>{setupCommands.join('\n')}</code>
                </pre>
              </div>

              <ul className="list-disc space-y-2 pl-5 text-sm text-muted-foreground">
                <li>{t('steps.username')}</li>
                <li>{t('steps.password')}</li>
                <li>{t('steps.secret')}</li>
                <li>{t('steps.restart')}</li>
              </ul>

              <p className="text-sm text-muted-foreground">
                {data?.message || t('description')}
              </p>
            </div>
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
          <Button onClick={() => void handleCopy()} className="rounded-xl" disabled={copied}>
            <Copy className="mr-2 size-4" />
            {copied ? t('actions.copied') : t('actions.copy')}
          </Button>
        </CardFooter>
      </Card>
    </div>
  );
}
