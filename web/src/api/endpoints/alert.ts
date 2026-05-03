import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

export interface AlertRule {
    id: number;
    name: string;
    enabled: boolean;
    condition_type: string;
    threshold: number;
    condition_json?: string;
    notif_channel_id: number;
    cooldown_sec: number;
    scope_channel_id?: number;
    scope_api_key_id?: number;
}

export interface AlertNotifChannel {
    id: number;
    name: string;
    type: string;
    url: string;
    secret?: string;
    headers?: string;
    config?: string;
}

export const NOTIF_CHANNEL_TYPES = ['webhook', 'gotify', 'email'] as const;
export type NotifChannelType = (typeof NOTIF_CHANNEL_TYPES)[number];

export interface GotifyConfig {
    server_url: string;
    token: string;
    priority?: number;
}

export interface EmailConfig {
    smtp_host: string;
    smtp_port: number;
    username: string;
    password: string;
    from: string;
    to: string;
    use_tls: boolean;
}

export interface AlertHistory {
    id: number;
    rule_id: number;
    rule_name: string;
    state: number;
    message: string;
    detail_json?: string;
    time: number;
}

export function useAlertRuleList() {
    return useQuery({
        queryKey: ['alerts', 'rules'],
        queryFn: async () => apiClient.get<AlertRule[]>('/api/v1/alert/rule/list'),
        refetchInterval: 30000,
    });
}

export function useCreateAlertRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<AlertRule>) => {
            return apiClient.post<AlertRule>('/api/v1/alert/rule/create', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'rules'] });
        },
        onError: (error) => logger.error('Create alert rule failed:', error),
    });
}

export function useUpdateAlertRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: AlertRule) => {
            return apiClient.post<null>('/api/v1/alert/rule/update', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'rules'] });
        },
        onError: (error) => logger.error('Update alert rule failed:', error),
    });
}

export function useDeleteAlertRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/alert/rule/delete/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'rules'] });
        },
        onError: (error) => logger.error('Delete alert rule failed:', error),
    });
}

export function useAlertNotifChannelList() {
    return useQuery({
        queryKey: ['alerts', 'channels'],
        queryFn: async () => apiClient.get<AlertNotifChannel[]>('/api/v1/alert/notif/list'),
        refetchInterval: 30000,
    });
}

export function useCreateNotifChannel() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: Partial<AlertNotifChannel>) => {
            return apiClient.post<AlertNotifChannel>('/api/v1/alert/notif/create', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'channels'] });
        },
        onError: (error) => logger.error('Create notif channel failed:', error),
    });
}

export function useUpdateNotifChannel() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (data: AlertNotifChannel) => {
            return apiClient.post<null>('/api/v1/alert/notif/update', data);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'channels'] });
        },
        onError: (error) => logger.error('Update notif channel failed:', error),
    });
}

export function useDeleteNotifChannel() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/alert/notif/delete/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['alerts', 'channels'] });
        },
        onError: (error) => logger.error('Delete notif channel failed:', error),
    });
}

export function useAlertHistory() {
    return useQuery({
        queryKey: ['alerts', 'history'],
        queryFn: async () => apiClient.get<AlertHistory[]>('/api/v1/alert/history?limit=50'),
        refetchInterval: 30000,
    });
}
