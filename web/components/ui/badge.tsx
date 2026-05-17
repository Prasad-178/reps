import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-[0.06em] transition-colors",
  {
    variants: {
      variant: {
        default: "border-transparent bg-[var(--secondary)] text-[var(--secondary-foreground)]",
        primary: "border-transparent bg-[color-mix(in_oklch,var(--primary)_18%,transparent)] text-[var(--primary)]",
        success: "border-transparent bg-[color-mix(in_oklch,var(--success)_18%,transparent)] text-[var(--success)]",
        warning: "border-transparent bg-[color-mix(in_oklch,var(--warning)_18%,transparent)] text-[var(--warning)]",
        danger:  "border-transparent bg-[color-mix(in_oklch,var(--destructive)_18%,transparent)] text-[var(--destructive)]",
        outline: "border-[var(--border)] text-[var(--foreground)]",
      },
    },
    defaultVariants: { variant: "default" },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}
