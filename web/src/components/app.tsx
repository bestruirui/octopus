
'use client';

import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from "motion/react"
import { useAuth } from '@/api/endpoints/user';
import { LoginForm } from '@/components/modules/login';
import { APIKeyDashboard } from '@/components/modules/apikey-dashboard';
import { ContentLoader } from '@/route/content-loader';
import { NavBar, useNavStore } from '@/components/modules/navbar';
import { useTranslations } from 'next-intl'
import Logo, { LOGO_DRAW_END_MS } from '@/components/modules/logo';
import { Toolbar } from '@/components/modules/toolbar';
import { ENTRANCE_VARIANTS } from '@/lib/animations/fluid-transitions';
import { useQueryClient } from '@tanstack/react-query';
import { CONTENT_MAP } from '@/route';
import { apiClient } from '@/api/client';
import { logger } from '@/lib/logger';

function timeout(ms: number) {
    return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

export function AppContainer() {
    const { isAuthenticated, isAPIKeyAuth, isLoading: authLoading } = useAuth();
    const { activeItem, direction } = useNavStore();
    const t = useTranslations('navbar');
    const queryClient = useQueryClient();

    // Logo 动画完成状态
    const [logoAnimationComplete, setLogoAnimationComplete] = useState(false);
    const [bootstrapComplete, setBootstrapComplete] = useState(false);
    const bootstrapStartedRef = useRef(false);

    // 首屏最早的 server-rendered loader：一旦客户端开始渲染，就淡出移除
    useEffect(() => {
        const el = document.getElementById('initial-loader');
        if (!el) return;

        el.classList.add('octo-hide');
        const timer = setTimeout(() => el.remove(), 220);
        return () => clearTimeout(timer);
    }, []);

    useEffect(() => {
        const timer = setTimeout(() => setLogoAnimationComplete(true), LOGO_DRAW_END_MS);
        return () => clearTimeout(timer);
    }, []);

    useEffect(() => {
        if (authLoading) return;
        if (!isAuthenticated) {
            setBootstrapComplete(true);
            return;
        }

        if (bootstrapStartedRef.current) return;
        bootstrapStartedRef.current = true;

        let cancelled = false;

        (async () => {
            try {
                const prefetches: Array<Promise<unknown>> = [];

                // API Key 认证模式：预取 dashboard stats
                if (isAPIKeyAuth) {
                    prefetches.push(
                        queryClient.prefetchQuery({
                            queryKey: ['apikey', 'dashboard', 'stats'],
                            queryFn: async () => apiClient.get('/api/v1/apikey/stats'),
                        })
                    );
                } else {
                    // 普通用户认证模式：预取对应页面数据
                    const component = CONTENT_MAP[activeItem];
                    if (component?.preload) {
                        prefetches.push(component.preload());
                    }

                    switch (activeItem) {
                        case 'home': {
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['stats', 'total'],
                                    queryFn: async () => apiClient.get('/api/v1/stats/total'),
                                })
                            );
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['stats', 'daily'],
                                    queryFn: async () => apiClient.get('/api/v1/stats/daily'),
                                })
                            );
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['stats', 'hourly'],
                                    queryFn: async () => apiClient.get('/api/v1/stats/hourly'),
                                })
                            );
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['channels', 'list'],
                                    queryFn: async () => apiClient.get('/api/v1/channel/list'),
                                })
                            );
                            break;
                        }
                        case 'channel': {
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['channels', 'list'],
                                    queryFn: async () => apiClient.get('/api/v1/channel/list'),
                                })
                            );
                            break;
                        }
                        case 'group': {
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['groups', 'list'],
                                    queryFn: async () => apiClient.get('/api/v1/group/list'),
                                })
                            );
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['models', 'channel'],
                                    queryFn: async () => apiClient.get('/api/v1/model/channel'),
                                })
                            );
                            break;
                        }
                        case 'model': {
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['models', 'list'],
                                    queryFn: async () => apiClient.get('/api/v1/model/list'),
                                })
                            );
                            break;
                        }
                        case 'setting': {
                            prefetches.push(
                                queryClient.prefetchQuery({
                                    queryKey: ['apikeys', 'list'],
                                    queryFn: async () => apiClient.get('/api/v1/apikey/list'),
                                })
                            );
                            break;
                        }
                        default:
                            break;
                    }
                }

                await Promise.race([
                    Promise.allSettled(prefetches),
                    timeout(5000),
                ]);
            } catch (e) {
                logger.warn('bootstrap prefetch failed:', e);
            } finally {
                if (!cancelled) setBootstrapComplete(true);
            }
        })();

        return () => {
            cancelled = true;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [authLoading, isAuthenticated]);

    // 加载状态
    const isLoading =
        authLoading ||
        !logoAnimationComplete ||
        (isAuthenticated && !bootstrapComplete);

    // 加载页面
    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Logo size={120} animate />
            </div>
        );
    }

    // API Key 认证模式 - 显示 API Key Dashboard
    if (isAPIKeyAuth) {
        return (
            <AnimatePresence mode="wait">
                <APIKeyDashboard key="apikey-dashboard" />
            </AnimatePresence>
        );
    }

    // 登录页面
    if (!isAuthenticated) {
        return (
            <AnimatePresence mode="wait">
                <LoginForm key="login" />
            </AnimatePresence>
        );
    }

    // 主界面
    return (
        <motion.div
            key="main-app"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.3 }}
            className="mx-auto flex h-dvh max-w-6xl flex-col overflow-hidden px-3 md:grid md:grid-cols-[auto_1fr] md:gap-6 md:px-6"
        >
            <NavBar />
            <main className="flex min-h-0 w-full min-w-0 flex-1 flex-col">
                <header className="my-6 flex flex-none items-center gap-x-2 px-2">
                    <Logo size={48} />
                    <div className="flex-1 overflow-hidden">
                        <AnimatePresence mode="wait" custom={direction}>
                            <motion.div
                                key={activeItem}
                                custom={direction}
                                variants={{
                                    initial: (direction: number) => ({
                                        y: 32 * direction,
                                        opacity: 0
                                    }),
                                    animate: {
                                        y: 0,
                                        opacity: 1
                                    },
                                    exit: (direction: number) => ({
                                        y: -32 * direction,
                                        opacity: 0
                                    })
                                }}
                                initial="initial"
                                animate="animate"
                                exit="exit"
                                transition={{ duration: 0.3 }}
                                className="flex items-center"
                            >
                                <span className="text-3xl font-bold mt-1">{t(activeItem)}</span>
                            </motion.div>
                        </AnimatePresence>
                    </div>
                    <div className="ml-auto">
                        <Toolbar />
                    </div>
                </header>
                <AnimatePresence mode="wait" initial={false}>
                    <motion.div
                        key={activeItem}
                        variants={ENTRANCE_VARIANTS.content}
                        initial="initial"
                        animate="animate"
                        exit={{
                            opacity: 0,
                            scale: 0.98,
                        }}
                        transition={{ duration: 0.25 }}
                        className="h-full min-h-0 flex-1"
                    >
                        <ContentLoader activeRoute={activeItem} />
                    </motion.div>
                </AnimatePresence>
            </main>
        </motion.div>
    );
}
