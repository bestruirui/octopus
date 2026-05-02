'use client';

import { motion, useReducedMotion } from 'motion/react';
import type { ReactNode } from 'react';
import { EASING } from '@/lib/animations/fluid-transitions';

/**
 * 呼吸元素——即使没有用户操作，元素也保持极其缓慢的"呼吸"运动。
 * 模拟生物体的脉动节律，避免界面死寂。
 */
export function BreathingElement({
  children,
  intensity = 1.008,
  duration = 8,
  className = '',
}: {
  children: ReactNode;
  intensity?: number;
  duration?: number;
  className?: string;
}) {
  const prefersReducedMotion = useReducedMotion();

  return (
    <motion.div
      className={className}
      animate={
        prefersReducedMotion
          ? {
              scale: 1,
              opacity: 1,
              y: 0,
            }
          : {
              scale: [1, intensity, 1],
              opacity: [1, 0.96, 1],
              y: [0, -2, 0],
            }
      }
      transition={
        prefersReducedMotion
          ? { duration: 0 }
          : {
              duration,
              repeat: Infinity,
              ease: EASING.waterhouseSurface,
            }
      }
    >
      {children}
    </motion.div>
  );
}
