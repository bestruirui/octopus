'use client';

import { cn } from '@/lib/utils';
import type { ReactNode } from 'react';

interface IslandShellProps {
    children: ReactNode;
    className?: string;
    showDecorations?: boolean;
}

/**
 * IslandShell - 封装 Waterhouse 设计系统中重复的 island 容器样式
 *
 * 提供统一的圆角、边框、背景、阴影和装饰性渐变背景
 */
export function IslandShell({ children, className, showDecorations = true }: IslandShellProps) {
    return (
        <div
            className={cn(
                'waterhouse-island relative flex h-full min-h-0 flex-col rounded-[2.35rem] border border-border/35 bg-card/54 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]',
                className
            )}
        >
            {showDecorations && (
                <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_14%,color-mix(in_oklch,var(--waterhouse-highlight)_20%,transparent)_0%,transparent_30%),radial-gradient(circle_at_84%_18%,color-mix(in_oklch,var(--primary)_14%,transparent)_0%,transparent_26%),linear-gradient(180deg,color-mix(in_oklch,white_16%,transparent),transparent_22%,color-mix(in_oklch,var(--waterhouse-highlight)_8%,transparent))]" />
            )}
            {children}
        </div>
    );
}
