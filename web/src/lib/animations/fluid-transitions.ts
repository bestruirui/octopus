import type { Variants } from 'motion/react';

// 缓动函数
export const EASING = {
    easeOutCubic: [0.25, 0.46, 0.45, 0.94] as const,
    easeOutExpo: [0.16, 1, 0.3, 1] as const,
    easeInOutCubic: [0.65, 0, 0.35, 1] as const,
    easeOutQuart: [0.25, 1, 0.5, 1] as const,
    /** Nature 涟漪扩散缓动——模拟水波衰减 */
    natureRipple: [0.12, 0.71, 0.33, 1] as const,
    /** Waterhouse 漂移缓动——雾层慢速滑移 */
    waterhouseDrift: [0.22, 1, 0.36, 1] as const,
    /** Waterhouse 浮游缓动——层级漂浮入场 */
    waterhouseFloat: [0.16, 1, 0.3, 1] as const,
    /** Waterhouse 舱壁形变——更柔的悬浮反馈 */
    waterhouseSurface: [0.2, 0.9, 0.32, 1] as const,
} as const;

// Spring 配置
export const SPRING = {
    smooth: {
        type: "spring" as const,
        stiffness: 80,
        damping: 20,
        mass: 1.2,
    },
    gentle: {
        type: "spring" as const,
        stiffness: 70,
        damping: 18,
        mass: 1.5,
    },
    bouncy: {
        type: "spring" as const,
        stiffness: 100,
        damping: 15,
        mass: 1,
    },
    /** Waterhouse 舱体悬浮——更稳的缓慢起落 */
    waterhouseHover: {
        type: "spring" as const,
        stiffness: 90,
        damping: 18,
        mass: 1.25,
    },
    /** Waterhouse 磁场吸附——位移/旋转更柔和 */
    waterhouseMagnetic: {
        type: "spring" as const,
        stiffness: 110,
        damping: 20,
        mass: 0.95,
    },
} as const;

/**
 * 磁性吸附进入动画
 */
export const ENTRANCE_VARIANTS = {
    // 导航栏进入——像气泡浮现
    navbar: {
        initial: {
            opacity: 0,
            scale: 0.4,
            filter: "blur(12px)",
        },
        animate: {
            opacity: 1,
            scale: 1,
            filter: "blur(0px)",
            transition: SPRING.waterhouseHover,
        },
    } as Variants,

    // 主内容进入——像液体融合
    content: {
        initial: {
            scale: 0.85,
            opacity: 0,
            filter: "blur(6px)",
        },
        animate: {
            scale: 1,
            opacity: 1,
            filter: "blur(0px)",
            transition: {
                duration: 0.6,
                ease: EASING.waterhouseFloat,
                delay: 0.08,
            },
        },
    } as Variants,

};

