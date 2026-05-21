import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

/**
 * TOTP 状态
 */
export interface TOTPStatus {
    enabled: boolean;
}

/**
 * TOTP 初始化响应
 */
export interface TOTPInitResponse {
    secret: string;
    uri: string;
}

/**
 * 获取 TOTP 状态 Hook
 */
export function useTOTPStatus() {
    return useQuery({
        queryKey: ['totp', 'status'],
        queryFn: async () => {
            return apiClient.get<TOTPStatus>('/api/v1/user/totp/status');
        },
    });
}

/**
 * 初始化 TOTP Hook
 */
export function useTOTPInit() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async () => {
            return apiClient.post<TOTPInitResponse>('/api/v1/user/totp/init');
        },
        onSuccess: (data) => {
            logger.log('TOTP 初始化成功:', data);
        },
        onError: (error) => {
            logger.error('TOTP 初始化失败:', error);
        },
    });
}

/**
 * 验证并启用 TOTP Hook
 */
export function useTOTPVerifySetup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: { code: string }) => {
            return apiClient.post<string>('/api/v1/user/totp/verify-setup', data);
        },
        onSuccess: () => {
            logger.log('TOTP 验证并启用成功');
            queryClient.invalidateQueries({ queryKey: ['totp', 'status'] });
        },
        onError: (error) => {
            logger.error('TOTP 验证并启用失败:', error);
        },
    });
}

/**
 * 禁用 TOTP Hook
 */
export function useTOTPDisable() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: { code: string }) => {
            return apiClient.post<string>('/api/v1/user/totp/disable', data);
        },
        onSuccess: () => {
            logger.log('TOTP 禁用成功');
            queryClient.invalidateQueries({ queryKey: ['totp', 'status'] });
        },
        onError: (error) => {
            logger.error('TOTP 禁用失败:', error);
        },
    });
}
