"use client"

import { useMemo } from "react"
import { motion } from "motion/react"
import { cn } from "@/lib/utils"
import { useNavStore, type NavItem } from "@/components/modules/navbar"
import { ROUTES } from "@/route/config"
import { usePreload } from "@/route/use-preload"
import { ENTRANCE_VARIANTS } from "@/lib/animations/fluid-transitions"
import { MagneticWrapper } from "@/components/nature"
import { useTranslations } from "next-intl"

export function NavBar() {
    const { activeItem, orderedItems, visibleItems, setActiveItem } = useNavStore()
    const { preload } = usePreload()
    const t = useTranslations('navbar')
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
                    "waterhouse-pod fixed bottom-5 left-1/2 flex max-w-[calc(100vw-1.5rem)] -translate-x-1/2 items-center gap-1.5 overflow-visible p-2.5",
                    "rounded-[2.15rem] border-sidebar-border/40 bg-sidebar/72 text-sidebar-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]",
                    "md:sticky md:top-6 md:left-auto md:bottom-auto md:h-[calc(100dvh-3rem)] md:max-w-none md:translate-x-0 md:flex-col md:gap-3 md:p-3.5"
                )}
                variants={ENTRANCE_VARIANTS.navbar}
                initial="initial"
                animate="animate"
            >
                <div className="pointer-events-none absolute inset-1 rounded-[1.85rem] border border-white/20 opacity-70 md:rounded-[2rem]" />
                <div className="pointer-events-none absolute left-1/2 top-1/2 h-12 w-[82%] -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary/10 blur-2xl md:h-[72%] md:w-12" />
                {orderedRoutes.map((route, index) => {
                    const isActive = activeItem === route.id
                    return (
                        <MagneticWrapper key={route.id} intensity={0.2} scale={1.04} className="z-20">
                            <motion.button
                                type="button"
                                aria-label={t(route.id as NavItem)}
                                onClick={() => setActiveItem(route.id as NavItem)}
                                onMouseEnter={() => preload(route.id)}
                                className={cn(
                                    "group relative z-20 grid size-11 place-items-center rounded-[1.55rem] border transition-[color,background-color,border-color,box-shadow,transform] duration-300 md:size-12",
                                    isActive
                                        ? "border-primary/20 text-sidebar-primary-foreground shadow-[0_18px_32px_-22px_color-mix(in_oklch,var(--primary)_55%,black)]"
                                        : "border-sidebar-border/25 bg-sidebar-accent/22 text-sidebar-foreground/58 hover:border-primary/20 hover:bg-background/42 hover:text-sidebar-foreground"
                                )}
                                initial={{ opacity: 0, scale: 0.8 }}
                                animate={{
                                    opacity: 1,
                                    scale: 1,
                                    transition: {
                                        delay: index * 0.05,
                                        duration: 0.3,
                                    }
                                }}
                                whileHover={{ scale: 1.04, zIndex: 30 }}
                                whileTap={{ scale: 0.92 }}
                            >
                                {isActive && (
                                    <motion.div
                                        layoutId="navbar-indicator"
                                        className="absolute -inset-1 z-0 bg-sidebar-primary"
                                        style={{
                                            borderRadius: '62% 38% 58% 42% / 46% 58% 42% 54%',
                                            filter: 'url(#nature-gooey)',
                                        }}
                                        transition={{
                                            type: "spring",
                                            stiffness: 200,
                                            damping: 24,
                                            mass: 0.8,
                                        }}
                                    />
                                )}
                                <span
                                    className={cn(
                                        "relative z-10 grid size-8 place-items-center rounded-[1.1rem] transition-transform duration-300 group-hover:scale-105",
                                        isActive ? "bg-white/18" : "bg-background/20"
                                    )}
                                >
                                    <route.icon className="size-5" strokeWidth={2} />
                                </span>
                                {isActive && (
                                    <span className="absolute -right-0.5 top-1/2 z-10 size-2 -translate-y-1/2 rounded-full bg-primary/70 shadow-[0_0_14px_var(--primary)] md:-right-1" />
                                )}
                            </motion.button>
                        </MagneticWrapper>
                    )
                })}
            </motion.nav>
        </div>
    )
}
