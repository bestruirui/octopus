'use client';

import { useRef, useState, type ReactNode } from 'react';
import { motion, useReducedMotion } from 'motion/react';
import { SPRING } from '@/lib/animations/fluid-transitions';

/**
 * 磁力包装器——鼠标靠近时元素像磁流体一样被光标吸引、拉伸。
 * 包裹按钮、卡片等交互元素，产生有机的吸引力反馈。
 */
export function MagneticWrapper({
  children,
  intensity = 0.3,
  scale = 1.05,
  className = '',
}: {
  children: ReactNode;
  intensity?: number;
  scale?: number;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const shouldReduceMotion = useReducedMotion();
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [rotation, setRotation] = useState(0);
  const [isHovered, setIsHovered] = useState(false);

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!ref.current) return;
    const rect = ref.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;
    const deltaX = (e.clientX - centerX) * intensity;
    const deltaY = (e.clientY - centerY) * intensity;
    const normalizedX = rect.width === 0 ? 0 : (e.clientX - centerX) / (rect.width / 2);
    setPosition({ x: deltaX, y: deltaY });
    setRotation(Math.max(-4, Math.min(4, normalizedX * 3.5)));
  };

  const handleMouseEnter = () => setIsHovered(true);
  const handleMouseLeave = () => {
    setIsHovered(false);
    setPosition({ x: 0, y: 0 });
    setRotation(0);
  };

  return (
    <motion.div
      ref={ref}
      className={`inline-block ${className}`}
      onMouseMove={handleMouseMove}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      animate={{
        x: shouldReduceMotion ? 0 : position.x,
        y: shouldReduceMotion ? 0 : position.y,
        rotate: shouldReduceMotion ? 0 : rotation,
        scale: isHovered ? scale : 1,
      }}
      transition={SPRING.waterhouseMagnetic}
      style={{ willChange: 'transform' }}
    >
      {children}
    </motion.div>
  );
}
