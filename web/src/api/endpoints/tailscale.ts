import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';

export type TailscalePhase =
    | 'unsupported'
    | 'not_installed'
    | 'needs_login'
    | 'stopped'
    | 'running'
    | 'conflict'
    | 'credentials_needed'
    | 'error';

export interface TailscaleStatus {
    supported: boolean;
    installed: boolean;
    daemon_running: boolean;
    logged_in: boolean;
    running: boolean;
    funnel_active: boolean;
    config_conflict: boolean;
    default_credentials: boolean;
    phase: TailscalePhase;
    binary_path?: string;
    target_url: string;
    public_url?: string;
    api_url?: string;
    approval_url?: string;
    last_error?: string;
}

const statusQueryKey = ['tailscale', 'status'] as const;

export function useTailscaleStatus() {
    return useQuery({
        queryKey: statusQueryKey,
        queryFn: () => apiClient.get<TailscaleStatus>('/api/v1/tailscale/status'),
        refetchInterval: 10000,
    });
}

function useTailscaleAction(path: 'start' | 'stop') {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => apiClient.post<TailscaleStatus>(`/api/v1/tailscale/${path}`),
        onSuccess: (status) => queryClient.setQueryData(statusQueryKey, status),
        onSettled: () => queryClient.invalidateQueries({ queryKey: statusQueryKey }),
    });
}

export function useStartTailscaleFunnel() {
    return useTailscaleAction('start');
}

export function useStopTailscaleFunnel() {
    return useTailscaleAction('stop');
}
