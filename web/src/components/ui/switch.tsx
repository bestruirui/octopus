"use client"

import * as React from "react"
import * as SwitchPrimitive from "@radix-ui/react-switch"

import { cn } from "@/lib/utils"

function Switch({
  className,
  ...props
}: React.ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "peer inline-flex h-[1.45rem] w-10 shrink-0 items-center rounded-full border border-border/40 bg-background/55 shadow-nature-organic backdrop-blur-md transition-[background-color,border-color,box-shadow,transform] duration-300 outline-none data-[state=checked]:border-primary/20 data-[state=checked]:bg-primary/80 data-[state=unchecked]:bg-background/55 focus-visible:border-ring focus-visible:ring-ring/30 focus-visible:ring-[4px] focus-visible:shadow-nature-glow hover:shadow-[var(--waterhouse-shadow-soft)] disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          "pointer-events-none block size-[1.05rem] rounded-full bg-background/90 ring-0 shadow-sm transition-all duration-300 ease-out data-[state=checked]:translate-x-[calc(100%+4px)] data-[state=unchecked]:translate-x-[2px] dark:data-[state=checked]:bg-primary-foreground"
        )}
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }
