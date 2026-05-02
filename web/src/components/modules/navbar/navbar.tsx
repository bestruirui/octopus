"use client"

import { useMemo } from "react"
import { motion, useReducedMotion } from "motion/react"
import { cn } from "@/lib/utils"
import { useNavStore, type NavItem } from "@/components/modules/navbar"
import { ROUTES } from "@/route/config"
import { usePreload } from "@/route/use-preload"
import { ENTRANCE_VARIANTS } from "@/lib/animations/fluid-transitions"
import { MagneticWrapper } from "@/components/nature"
import { useTranslations } from "next-intl"
import { useIsMobile } from "@/hooks/use-mobile"

export function NavBar() {
    const { activeItem, orderedItems, visibleItems, setActiveItem } = useNavStore()
    const { preload } = usePreload()
    const t = useTranslations('navbar')
    const isMobile = useIsMobile()
    const reduceMotion = useReducedMotion()
    const lightweightMotion = isMobile || reduceMotion
    const visibleRouteSet = useMemo(() => new Set(visibleItems), [visibleItems])
    const routeById = useMemo(
        () => new Map(ROUTES.map((route) => [route.id as NavItem, route])),
        []
    )
    const orderedRoutes = useMemo(
        () =>
            orderedItems
                .filter((item) => visibleRouteSet.has(item))
                .map((item) => routeById.get(item))
                .filter((route) => route !== undefined),
        [orderedItems, routeById, visibleRouteSet]
    )

    return (
        <div className="relative z-50 md:min-h-full">
            <motion.nav
                aria-label={t('ariaLabel')}
                className={cn(
                    "waterhouse-pod fixed bottom-5 left-1/2 flex max-w-[calc(100vw-1.5rem)] -translate-x-1/2 items-center gap-1 overflow-x-auto p-2.5 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
                    "rounded-[2.15rem] border-sidebar-border/40 bg-sidebar/72 text-sidebar-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]",
                    "md:sticky md:top-6 md:left-auto md:bottom-auto md:h-[calc(100dvh-3rem)] md:max-w-none md:translate-x-0 md:flex-col md:gap-3 md:overflow-visible md:p-3.5"
                )}
                variants={lightweightMotion ? undefined : ENTRANCE_VARIANTS.navbar}
                initial={lightweightMotion ? false : "initial"}
                animate={lightweightMotion ? undefined : "animate"}
            >
                <div className="pointer-events-none absolute inset-1 rounded-[1.85rem] border border-white/20 opacity-70 md:rounded-[2rem]" />
                <div className="pointer-events-none absolute left-1/2 top-1/2 h-12 w-[82%] -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/10 blur-2xl md:h-[72%] md:w-12" />
                {/* 移动端：滚动边缘渐变指示器，提示用户有更多导航项可滚动 */}
                <div className="pointer-events-none absolute inset-y-0 left-0 z-30 w-8 rounded-l-[2.15rem] bg-gradient-to-r from-sidebar/90 to-transparent md:hidden" />
                <div className="pointer-events-none absolute inset-y-0 right-0 z-30 w-8 rounded-r-[2.15rem] bg-gradient-to-l from-sidebar/90 to-transparent md:hidden" />
                {orderedRoutes.map((route, index) => {
                    const isActive = activeItem === route.id
                    return (
                        <MagneticWrapper key={route.id} intensity={lightweightMotion ? 0 : 0.2} scale={lightweightMotion ? 1 : 1.04} className="z-20">
                            <motion.button
                                type="button"
                                aria-label={t(route.id as NavItem)}
                                onClick={() => setActiveItem(route.id as NavItem)}
                                onMouseEnter={() => {
                                    if (!isMobile) {
                                        preload(route.id)
                                    }
                                }}
                                className={cn(
                                    "group relative z-20 grid size-9 place-items-center rounded-[1.55rem] border transition-[color,background-color,border-color,box-shadow,transform] duration-300 md:size-12",
                                    isActive
                                        ? "border-primary/20 text-sidebar-primary-foreground shadow-[0_18px_32px_-22px_color-mix(in_oklch,var(--primary)_55%,black)]"
                                        : "border-sidebar-border/25 bg-sidebar-accent/22 text-sidebar-foreground/58 hover:border-primary/20 hover:bg-background/42 hover:text-sidebar-foreground"
                                )}
                                initial={lightweightMotion ? false : { opacity: 0, scale: 0.8 }}
                                animate={lightweightMotion ? undefined : {
                                    opacity: 1,
                                    scale: 1,
                                    transition: {
                                        delay: index * 0.05,
                                        duration: 0.3,
                                    }
                                }}
                                whileHover={lightweightMotion ? undefined : { scale: 1.04, zIndex: 30 }}
                                whileTap={lightweightMotion ? { scale: 0.97 } : { scale: 0.92 }}
                            >
                                {isActive && (
                                    <motion.div
                                        layoutId="navbar-indicator"
                                        className="absolute -inset-1 z-0 bg-sidebar-primary"
                                        style={{
                                            borderRadius: '62% 38% 58% 42% / 46% 58% 42% 54%',
                                            filter: lightweightMotion ? undefined : 'url(#nature-gooey)',
                                        }}
                                        transition={{
                                            type: "spring",
                                            stiffness: lightweightMotion ? 280 : 200,
                                            damping: lightweightMotion ? 32 : 24,
                                            mass: lightweightMotion ? 0.6 : 0.8,
                                        }}
                                    />
                                )}
                                <span
                                    className={cn(
                                        "relative z-10 grid size-6 place-items-center rounded-[1.1rem] transition-transform duration-300 group-hover:scale-105 md:size-8",
                                        isActive ? "bg-white/18" : "bg-background/20"
                                    )}
                                >
                                    <route.icon className="size-4 md:size-5" strokeWidth={2} />
                                </span>
                                {isActive && (
                                    <span className="absolute -right-0.5 top-1/2 z-10 size-1.5 -translate-y-1/2 rounded-full bg-primary/70 shadow-[0_0_14px_var(--primary)] md:size-2 md:-right-1" />
                                )}
                            </motion.button>
                        </MagneticWrapper>
                    )
                })}
            </motion.nav>
        </div>
    )
}
