import * as React from "react"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground h-9 w-full min-w-0 rounded-[1.2rem] border border-border/40 bg-background/55 px-3 py-1 text-base shadow-nature-organic backdrop-blur-md transition-[border-color,box-shadow,background-color,transform] duration-300 outline-none file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm hover:border-primary/15 hover:shadow-[var(--waterhouse-shadow-soft)]",
        "focus-visible:border-ring focus-visible:ring-ring/30 focus-visible:ring-[4px] focus-visible:shadow-nature-glow focus-visible:bg-background/68",
        "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
        className
      )}
      {...props}
    />
  )
}

export { Input }
