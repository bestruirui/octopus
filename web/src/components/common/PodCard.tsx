'use client';

import { cn } from '@/lib/utils';
import type { ReactNode } from 'react';

interface PodCardProps {
    children: ReactNode;
    className?: string;
    variant?: 'default' | 'dashed' | 'empty';
}

/**
 * PodCard - 封装 Waterhouse 设计系统中重复的卡片样式模式
 *
 * 默认变体：白色半透明背景 + 圆角 + 边框 + 模糊
 * dashed 变体：虚线边框，用于空状态占位
 * empty 变体：更透明的背景，用于内容缺失状态
 */
export function PodCard({ children, className, variant = 'default' }: PodCardProps) {
    return (
        <div
            className={cn(
                'relative overflow-hidden rounded-[1.8rem] shadow-waterhouse-soft',
                variant === 'default' && 'border border-border/30 bg-background/34 backdrop-blur-md',
                variant === 'dashed' && 'border border-dashed border-border/35 bg-background/28 backdrop-blur-md',
                variant === 'empty' && 'border border-border/25 bg-background/20 backdrop-blur-sm',
                className
            )}
        >
            {children}
        </div>
    );
}
