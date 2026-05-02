'use client';

import { motion, useReducedMotion } from 'motion/react';
import { EASING } from '@/lib/animations/fluid-transitions';

/**
 * 有机 Blob 形状——模拟细胞、水滴或不规则生物体。
 * 使用 border-radius 关键帧动画实现永动的有机形态变换。
 */
export function OrganicBlob({
  size = 300,
  color = 'oklch(0.68 0.08 150 / 0.4)',
  blur = 60,
  className = '',
  style,
}: {
  size?: number;
  color?: string;
  blur?: number;
  className?: string;
  style?: React.CSSProperties;
}) {
  const prefersReducedMotion = useReducedMotion();

  return (
    <motion.div
      aria-hidden="true"
      className={`pointer-events-none fixed z-0 ${className}`}
      style={{
        width: size,
        height: size,
        background: `radial-gradient(circle at 40% 35%, ${color} 0%, transparent 70%)`,
        filter: `blur(${blur}px)`,
        borderRadius: '60% 40% 30% 70% / 60% 30% 70% 40%',
        ...style,
      }}
      animate={
        prefersReducedMotion
          ? {
              borderRadius: '60% 40% 30% 70% / 60% 30% 70% 40%',
              x: 0,
              y: 0,
              rotate: 0,
              scale: 1,
            }
          : {
              borderRadius: [
                '60% 40% 30% 70% / 60% 30% 70% 40%',
                '30% 60% 70% 40% / 50% 60% 30% 60%',
                '40% 60% 30% 70% / 70% 30% 60% 40%',
                '70% 30% 50% 50% / 40% 70% 30% 60%',
                '60% 40% 30% 70% / 60% 30% 70% 40%',
              ],
              x: [0, 15, -10, -15, 0],
              y: [0, -10, 15, -5, 0],
              rotate: [0, 3, -2, 4, 0],
              scale: [1, 1.04, 0.98, 1.03, 1],
            }
      }
      transition={
        prefersReducedMotion
          ? { duration: 0 }
          : {
              duration: 18,
              repeat: Infinity,
              ease: EASING.waterhouseDrift,
            }
      }
    />
  );
}
