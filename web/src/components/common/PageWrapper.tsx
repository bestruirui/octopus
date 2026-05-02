'use client';

import { motion, AnimatePresence } from 'motion/react';
import { ReactNode, isValidElement, Children } from 'react';
import { EASING } from '@/lib/animations/fluid-transitions';

interface PageWrapperProps {
  children: ReactNode;
  className?: string;
}

/**
 * 计算递减延迟 - 前几段先拉开层次，后续快速并拢。
 * Waterhouse 节奏比 Nature 更平滑，因此首段更轻、收敛更早。
 */
function getDiminishingDelay(index: number): number {
  if (index === 0) return 0;
  return Math.min(0.075 * Math.log2(index + 1), 0.3);
}

/**
 * 通用页面包装器——Waterhouse 段落节奏。
 * 使用递减延迟 + 浮游缓动，让段落像水雾舱体一样依次显形。
 */
export function PageWrapper({ children, className = 'space-y-6' }: PageWrapperProps) {
  const childArray = Children.toArray(children);

  return (
    <motion.div className={className}>
      <AnimatePresence>
        {childArray.map((child, index) => {
          const key = isValidElement(child) ? child.key : null;

          return (
            <motion.div
              key={key ?? index}
              initial={{ opacity: 0, y: 28, scale: 0.985 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{
                opacity: 0,
                y: 12,
                scale: 0.99,
                transition: { duration: 0.32, ease: EASING.waterhouseDrift }
              }}
              transition={{
                duration: 0.65,
                ease: EASING.waterhouseFloat,
                delay: getDiminishingDelay(index),
              }}
            >
              {child}
            </motion.div>
          );
        })}
      </AnimatePresence>
    </motion.div>
  );
}
