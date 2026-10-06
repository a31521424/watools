import * as React from "react"

import { cn } from "@/lib/utils"

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "default" | "destructive" | "outline" | "ghost"
  size?: "default" | "sm" | "lg"
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", ...props }, ref) => {
    return (
      <button
        className={cn(
          "inline-flex items-center justify-center rounded-lg text-sm font-medium transition-colors",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
          "disabled:pointer-events-none disabled:opacity-50",
          {
            "bg-primary text-primary-foreground hover:brightness-95": variant === "default",
            "bg-destructive text-white hover:brightness-95": variant === "destructive",
            "border border-border bg-card hover:bg-secondary": variant === "outline",
            "hover:bg-secondary": variant === "ghost",
          },
          {
            "h-9 px-4 py-2": size === "default",
            "h-8 px-3 text-sm": size === "sm",
            "h-12 px-6": size === "lg",
          },
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button }
