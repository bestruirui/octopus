import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[1.25rem] border border-border/40 bg-background/55 text-sm font-medium text-foreground shadow-nature-organic backdrop-blur-md transition-[color,background-color,border-color,box-shadow,opacity,transform] duration-300 ease-out disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive active:scale-[0.96] hover:border-primary/20 hover:shadow-[var(--waterhouse-shadow-soft)]",
  {
    variants: {
      variant: {
        default:
          "border-primary/18 bg-primary/82 text-primary-foreground hover:bg-primary/88",
        destructive:
          "border-destructive/20 bg-destructive/78 text-white hover:bg-destructive/84 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/66",
        outline:
          "bg-background/55 text-foreground hover:bg-accent/18 hover:text-accent-foreground",
        secondary:
          "border-border/35 bg-secondary/72 text-secondary-foreground hover:bg-secondary/82",
        ghost:
          "border-transparent bg-transparent shadow-none backdrop-blur-0 hover:border-border/25 hover:bg-background/42 hover:text-accent-foreground",
        link: "border-transparent bg-transparent p-0 text-primary shadow-none backdrop-blur-0 hover:bg-transparent hover:text-primary hover:underline hover:shadow-none underline-offset-4",
      },
      size: {
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        sm: "h-8 rounded-[1rem] gap-1.5 px-3 has-[>svg]:px-2.5",
        lg: "h-10 rounded-[1.35rem] px-6 has-[>svg]:px-4",
        icon: "size-9 rounded-[1.15rem]",
        "icon-sm": "size-8 rounded-[1rem]",
        "icon-lg": "size-10 rounded-[1.35rem]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
