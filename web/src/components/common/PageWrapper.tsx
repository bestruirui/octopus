'use client';

import { motion, AnimatePresence } from 'motion/react';
import { ReactNode, isValidElement, Children } from 'react';
import { EASING } from '@/lib/animations/fluid-transitions';

interface PageWrapperProps {
  children: ReactNode;
  className?: string;
}

/**
 * 计算递减延迟 - 前几个元素延迟明显，后续快速收敛
 * 使用对数函数让延迟在前几个元素后迅速趋于平稳
 * 例如: [0, 0.08, 0.14, 0.18, 0.21, 0.24, ...] 最大约 0.4s
 */
function getDiminishingDelay(index: number): number {
  if (index === 0) return 0;
  // 对数递减：延迟增长逐渐变慢，最大延迟约 0.4s
  return Math.min(0.08 * Math.log2(index + 1), 0.4);
}

/**
 * 通用页面包装器，为页面内容添加流体动画效果
 * 使用递减延迟策略，避免元素过多时动画时间过长
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
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{
                opacity: 0,
                scale: 0.95,
                transition: { duration: 0.3 }
              }}
              transition={{
                duration: 0.5,
                ease: EASING.easeOutExpo,
                delay: getDiminishingDelay(index),
              }}
              layout
            >
              {child}
            </motion.div>
          );
        })}
      </AnimatePresence>
    </motion.div>
  );
}
