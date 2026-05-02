'use client';

import { useEffect, useRef, useCallback } from 'react';

interface Particle {
  x: number;
  y: number;
  size: number;
  speedX: number;
  speedY: number;
  opacity: number;
  opacityDir: number;
  life: number;
  maxLife: number;
}

/**
 * Canvas 粒子背景——模拟尘埃、花粉或气泡的布朗运动。
 * 使用类柏林噪声流动场产生空间相关的流体运动，模拟自然界的微观粒子漂浮。
 */
export function ParticleBackground({
  count = 40,
  minSize = 1,
  maxSize = 3,
  minOpacity = 0.1,
  maxOpacity = 0.35,
  className = '',
}: {
  count?: number;
  minSize?: number;
  maxSize?: number;
  minOpacity?: number;
  maxOpacity?: number;
  className?: string;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const particlesRef = useRef<Particle[]>([]);
  const rafRef = useRef<number>(0);
  const visibleRef = useRef(true);
  const reducedMotionRef = useRef(false);

  const initParticles = useCallback(
    (w: number, h: number) => {
      const particles: Particle[] = [];
      for (let i = 0; i < count; i++) {
        // 自然非均匀性：少数粒子明显更大或更小
        const sizeRandom = Math.random();
        let sizeMul = 1;
        if (sizeRandom < 0.05) {
          sizeMul = 2;       // 5% 为大粒子（花粉）
        } else if (sizeRandom < 0.15) {
          sizeMul = 0.5;     // 10% 为小粒子（尘埃）
        }
        const baseSize = minSize + Math.random() * (maxSize - minSize);
        particles.push({
          x: Math.random() * w,
          y: Math.random() * h,
          size: baseSize * sizeMul,
          speedX: (Math.random() - 0.5) * 0.3,
          speedY: (Math.random() - 0.5) * 0.3 - 0.1,
          opacity: minOpacity + Math.random() * (maxOpacity - minOpacity),
          opacityDir: Math.random() > 0.5 ? 1 : -1,
          life: Math.random() * 300,
          maxLife: 200 + Math.random() * 600,  // 更大的生命周期方差
        });
      }
      return particles;
    },
    [count, minSize, maxSize, minOpacity, maxOpacity]
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    reducedMotionRef.current = reducedMotionQuery.matches;

    let particles = initParticles(canvas.width, canvas.height);
    particlesRef.current = particles;
    let width = window.innerWidth;
    let height = window.innerHeight;
    let color = document.documentElement.classList.contains('dark')
      ? '166, 214, 201'
      : '132, 186, 165';

    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      width = window.innerWidth;
      height = window.innerHeight;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      particles = initParticles(width, height);
      particlesRef.current = particles;
    };

    const updateColor = () => {
      color = document.documentElement.classList.contains('dark')
        ? '166, 214, 201'
        : '132, 186, 165';
    };

    const drawStatic = () => {
      ctx.clearRect(0, 0, width, height);
      for (let i = 0; i < particles.length; i++) {
        const p = particles[i];
        ctx.beginPath();
        ctx.fillStyle = `rgba(${color}, ${p.opacity * 0.55})`;
        ctx.shadowColor = `rgba(${color}, ${p.opacity * 0.35})`;
        ctx.shadowBlur = p.size * 4;
        ctx.arc(p.x, p.y, p.size * 1.9, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.shadowBlur = 0;
    };

    const handleVisibilityChange = () => {
      visibleRef.current = document.visibilityState === 'visible';
      if (visibleRef.current) {
        rafRef.current = requestAnimationFrame(animate);
      } else {
        cancelAnimationFrame(rafRef.current);
      }
    };

    const handleReducedMotionChange = () => {
      reducedMotionRef.current = reducedMotionQuery.matches;
      if (reducedMotionRef.current) {
        cancelAnimationFrame(rafRef.current);
        drawStatic();
      } else if (visibleRef.current) {
        rafRef.current = requestAnimationFrame(animate);
      }
    };

    resize();
    updateColor();
    window.addEventListener('resize', resize);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    reducedMotionQuery.addEventListener('change', handleReducedMotionChange);
    const themeObserver = new MutationObserver(updateColor);
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });

    const animate = () => {
      if (!visibleRef.current || reducedMotionRef.current) {
        return;
      }

      ctx.clearRect(0, 0, width, height);

      // 流动场时间——每帧递增，驱动正弦噪声
      const flowTime = performance.now() * 0.0003;

      for (let i = 0; i < particles.length; i++) {
        const p = particles[i];

        // 类柏林噪声流动场：相邻粒子倾向于相似方向，产生流体涡流
        const angle =
          Math.sin(p.x * 0.008 + flowTime) *
          Math.cos(p.y * 0.01 + flowTime * 0.8) *
          Math.PI;
        const flowForce = 0.04;
        p.speedX += Math.cos(angle) * flowForce;
        p.speedY += Math.sin(angle) * flowForce;
        // 速度阻尼，防止加速失控
        p.speedX *= 0.985;
        p.speedY *= 0.985;
        // 速度钳制
        const maxSpeed = 0.4;
        const speed = Math.sqrt(p.speedX * p.speedX + p.speedY * p.speedY);
        if (speed > maxSpeed) {
          p.speedX = (p.speedX / speed) * maxSpeed;
          p.speedY = (p.speedY / speed) * maxSpeed;
        }

        p.x += p.speedX;
        p.y += p.speedY;

        // 透明度呼吸
        p.opacity += p.opacityDir * 0.002;
        if (p.opacity >= maxOpacity) p.opacityDir = -1;
        if (p.opacity <= minOpacity) p.opacityDir = 1;

        // 生命周期
        p.life++;
        if (p.life > p.maxLife) {
          p.x = Math.random() * width;
          p.y = height + 20;
          p.life = 0;
          p.maxLife = 200 + Math.random() * 600;
        }

        // 边界环绕
        if (p.x < -20) p.x = width + 20;
        if (p.x > width + 20) p.x = -20;
        if (p.y < -20) p.y = height + 20;

        // 使用 canvas shadow 保留柔光感，避免每帧每粒子创建径向渐变。
        ctx.beginPath();
        ctx.fillStyle = `rgba(${color}, ${p.opacity})`;
        ctx.shadowColor = `rgba(${color}, ${p.opacity * 0.45})`;
        ctx.shadowBlur = p.size * 5;
        ctx.arc(p.x, p.y, p.size * 1.8, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.shadowBlur = 0;

      rafRef.current = requestAnimationFrame(animate);
    };

    if (reducedMotionRef.current) {
      drawStatic();
    } else {
      rafRef.current = requestAnimationFrame(animate);
    }

    return () => {
      cancelAnimationFrame(rafRef.current);
      window.removeEventListener('resize', resize);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      reducedMotionQuery.removeEventListener('change', handleReducedMotionChange);
      themeObserver.disconnect();
    };
  }, [initParticles, maxOpacity, minOpacity]);

  return (
    <canvas
      ref={canvasRef}
      className={`pointer-events-none fixed inset-0 z-0 ${className}`}
      aria-hidden="true"
    />
  );
}
