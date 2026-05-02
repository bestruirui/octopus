import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border border-border/35 bg-background/52 px-2.5 py-1 text-xs font-medium whitespace-nowrap shadow-nature-organic backdrop-blur-md transition-[background-color,border-color,box-shadow] duration-300 [&>svg]:size-3 [&>svg]:pointer-events-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "border-primary/18 bg-primary/82 text-primary-foreground [a&]:hover:bg-primary/88 [a&]:hover:shadow-[var(--waterhouse-shadow-soft)]",
        secondary:
          "border-border/30 bg-secondary/72 text-secondary-foreground [a&]:hover:bg-secondary/82",
        destructive:
          "border-destructive/20 bg-destructive/78 text-white [a&]:hover:bg-destructive/86 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/66",
        outline:
          "text-foreground [a&]:hover:bg-accent/12 [a&]:hover:text-accent-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span"

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
