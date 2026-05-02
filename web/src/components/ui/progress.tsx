"use client"

import * as React from "react"
import * as ProgressPrimitive from "@radix-ui/react-progress"

import { cn } from "@/lib/utils"

function Progress({
  className,
  value,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root>) {
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      className={cn(
        "relative h-2.5 w-full overflow-hidden rounded-full border border-border/30 bg-background/55 shadow-[inset_0_1px_1px_rgba(255,255,255,0.08)] backdrop-blur-sm",
        className
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className="h-full w-full flex-1 rounded-full bg-[linear-gradient(90deg,color-mix(in_oklch,var(--primary)_88%,white_12%)_0%,color-mix(in_oklch,var(--accent)_72%,var(--primary)_28%)_100%)] shadow-[0_0_18px_-8px_var(--primary)] transition-all duration-700 ease-out"
        style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
      />
    </ProgressPrimitive.Root>
  )
}

export { Progress }
