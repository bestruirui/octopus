import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

/**
 * 工作分组信息
 */
export interface RouteGroup {
    id?: number;
    route_id?: number;
    group_id: number;
    category: string;
    description?: string;
    keywords?: string;
}

/**
 * 路由信息
 */
export interface Route {
    id?: number;
    name: string;
    primary_group_id: number;
    dispatch_group_id?: number;
    description?: string;
    work_groups?: RouteGroup[];
}

/**
 * 新增工作分组请求
 */
export interface RouteGroupAddRequest {
    group_id: number;
    category: string;
    description?: string;
    keywords?: string;
}

/**
 * 更新工作分组请求
 */
export interface RouteGroupUpdateRequest {
    id: number;
    group_id?: number;
    category?: string;
    description?: string;
    keywords?: string;
}

/**
 * 创建路由请求
 */
export interface RouteCreateRequest {
    name: string;
    primary_group_id: number;
    dispatch_group_id?: number;
    description?: string;
    work_groups?: RouteGroupAddRequest[];
}

/**
 * 更新路由请求 - 仅包含变更的数据
 */
export interface RouteUpdateRequest {
    id: number;
    name?: string;
    primary_group_id?: number;
    dispatch_group_id?: number;
    description?: string;
    work_groups_to_add?: RouteGroupAddRequest[];
    work_groups_to_update?: RouteGroupUpdateRequest[];
    work_groups_to_delete?: number[];
}

/**
 * 获取路由列表 Hook
 */
export function useRouteList() {
    return useQuery({
        queryKey: ['routes', 'list'],
        queryFn: async () => {
            return apiClient.get<Route[]>('/api/v1/route/list');
        },
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 创建路由 Hook
 */
export function useCreateRoute() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: RouteCreateRequest) => {
            return apiClient.post<Route>('/api/v1/route/create', data);
        },
        onSuccess: (data) => {
            logger.log('路由创建成功:', data);
            queryClient.invalidateQueries({ queryKey: ['routes', 'list'] });
        },
        onError: (error) => {
            logger.error('路由创建失败:', error);
        },
    });
}

/**
 * 更新路由 Hook - 仅发送变更的数据
 */
export function useUpdateRoute() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: RouteUpdateRequest) => {
            return apiClient.post<Route>('/api/v1/route/update', data);
        },
        onSuccess: (data) => {
            logger.log('路由更新成功:', data);
            queryClient.invalidateQueries({ queryKey: ['routes', 'list'] });
        },
        onError: (error) => {
            logger.error('路由更新失败:', error);
        },
    });
}

/**
 * 删除路由 Hook
 */
export function useDeleteRoute() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/route/delete/${id}`);
        },
        onSuccess: () => {
            logger.log('路由删除成功');
            queryClient.invalidateQueries({ queryKey: ['routes', 'list'] });
        },
        onError: (error) => {
            logger.error('路由删除失败:', error);
        },
    });
}
