'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { EASING } from '@/lib/animations/fluid-transitions';

interface Ripple {
  id: number;
  x: number;
  y: number;
}

/**
 * 涟漪效果——光标移动时在背景上留下淡淡的水波纹轨迹。
 * 模拟水滴落在静水面的扩散效果。
 */
export function RippleEffect({
  maxRipples = 20,
  throttleMs = 80,
  className = '',
}: {
  maxRipples?: number;
  throttleMs?: number;
  className?: string;
}) {
  const [ripples, setRipples] = useState<Ripple[]>([]);
  const idRef = useRef(0);
  const lastTimeRef = useRef(0);
  const rafRef = useRef<number>(0);
  const pendingRippleRef = useRef<Ripple | null>(null);
  const enabledRef = useRef(true);

  const flushRipple = useCallback(() => {
    rafRef.current = 0;
    const ripple = pendingRippleRef.current;
    pendingRippleRef.current = null;
    if (!ripple || !enabledRef.current) return;

    setRipples((prev) => {
      const next = [...prev, ripple];
      if (next.length > maxRipples) {
        return next.slice(next.length - maxRipples);
      }
      return next;
    });
  }, [maxRipples]);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!enabledRef.current) return;

      const now = performance.now();
      if (now - lastTimeRef.current < throttleMs) return;
      lastTimeRef.current = now;

      pendingRippleRef.current = {
        id: idRef.current++,
        x: e.clientX,
        y: e.clientY,
      };

      if (rafRef.current === 0) {
        rafRef.current = requestAnimationFrame(flushRipple);
      }
    },
    [flushRipple, throttleMs]
  );

  useEffect(() => {
    const reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updateEnabled = () => {
      enabledRef.current =
        !reducedMotionQuery.matches && document.visibilityState === 'visible';
      if (!enabledRef.current) {
        pendingRippleRef.current = null;
        setRipples([]);
        if (rafRef.current !== 0) {
          cancelAnimationFrame(rafRef.current);
          rafRef.current = 0;
        }
      }
    };

    updateEnabled();
    window.addEventListener('mousemove', handleMouseMove, { passive: true });
    document.addEventListener('visibilitychange', updateEnabled);
    reducedMotionQuery.addEventListener('change', updateEnabled);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('visibilitychange', updateEnabled);
      reducedMotionQuery.removeEventListener('change', updateEnabled);
      if (rafRef.current !== 0) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, [handleMouseMove]);

  useEffect(() => {
    if (ripples.length === 0) return;
    const timer = setTimeout(() => {
      setRipples((prev) => prev.slice(1));
    }, 2000);
    return () => clearTimeout(timer);
  }, [ripples]);

  return (
    <div
      className={`pointer-events-none fixed inset-0 z-0 overflow-hidden ${className}`}
      aria-hidden="true"
    >
      <AnimatePresence>
        {ripples.map((ripple) => (
          <motion.div
            key={ripple.id}
            initial={{ scale: 0.2, opacity: 0.22 }}
            animate={{ scale: 3.6, opacity: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 1.9, ease: EASING.natureRipple }}
            className="absolute rounded-full border border-primary/20 bg-[radial-gradient(circle,rgba(255,255,255,0.16)_0%,rgba(255,255,255,0)_62%)] shadow-[0_0_0_1px_rgba(255,255,255,0.08)_inset]"
            style={{
              left: ripple.x - 20,
              top: ripple.y - 20,
              width: 40,
              height: 40,
            }}
          />
        ))}
      </AnimatePresence>
    </div>
  );
}
