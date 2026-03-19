import * as React from "react"

import { cn } from "@/lib/utils"

const SHEET_ANIMATION_MS = 240

const Sheet = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    open?: boolean
    onOpenChange?: (open: boolean) => void
  }
>(({ className, open, onOpenChange, children, ...props }, ref) => {
  const [mounted, setMounted] = React.useState(Boolean(open))
  const [visible, setVisible] = React.useState(Boolean(open))

  React.useEffect(() => {
    let animationFrame = 0
    let timeoutId: ReturnType<typeof setTimeout> | null = null

    if (open) {
      setMounted(true)
      animationFrame = window.requestAnimationFrame(() => {
        setVisible(true)
      })
    } else if (mounted) {
      setVisible(false)
      timeoutId = setTimeout(() => {
        setMounted(false)
      }, SHEET_ANIMATION_MS)
    }

    return () => {
      window.cancelAnimationFrame(animationFrame)
      if (timeoutId) {
        clearTimeout(timeoutId)
      }
    }
  }, [mounted, open])

  if (!mounted) return null

  return (
    <div className="absolute inset-0 z-50 flex overflow-hidden">
      {/* Backdrop */}
      <div
        className={cn(
          "absolute inset-0 bg-black/20 backdrop-blur-[1px] transition-opacity duration-200 ease-out",
          visible ? "opacity-100" : "opacity-0"
        )}
        onClick={() => onOpenChange?.(false)}
      />
      {/* Sheet */}
      <div
        ref={ref}
        className={cn(
          "absolute right-0 top-0 flex h-full w-[400px] max-w-full flex-col overflow-hidden bg-white shadow-2xl transition-transform duration-200",
          "ease-[cubic-bezier(0.22,1,0.36,1)]",
          visible ? "translate-x-0" : "translate-x-[104%]",
          className
        )}
        {...props}
      >
        {children}
      </div>
    </div>
  )
})
Sheet.displayName = "Sheet"

const SheetHeader = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("flex flex-col space-y-2 border-b p-6", className)}
    {...props}
  />
))
SheetHeader.displayName = "SheetHeader"

const SheetTitle = React.forwardRef<
  HTMLHeadingElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h2
    ref={ref}
    className={cn("text-lg font-semibold", className)}
    {...props}
  />
))
SheetTitle.displayName = "SheetTitle"

const SheetDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p
    ref={ref}
    className={cn("text-sm text-gray-500", className)}
    {...props}
  />
))
SheetDescription.displayName = "SheetDescription"

const SheetContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("flex-1 overflow-y-auto p-6", className)}
    {...props}
  />
))
SheetContent.displayName = "SheetContent"

const SheetFooter = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("flex items-center justify-end gap-2 border-t p-6", className)}
    {...props}
  />
))
SheetFooter.displayName = "SheetFooter"

export { Sheet, SheetHeader, SheetTitle, SheetDescription, SheetContent, SheetFooter }
