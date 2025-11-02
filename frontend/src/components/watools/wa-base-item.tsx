import { ReactNode } from "react";
import { CommandItem } from "@/components/ui/command";

export interface BaseItemProps {
  id: string;
  triggerId: string;
  title: string;
  icon: ReactNode;
  usedCount?: number;
  subtitle?: string;
  badge?: string;
  onSelect: () => void;
  children?: ReactNode;
}

export const WaBaseItem = ({ triggerId, icon, title, subtitle, badge, onSelect, children }: BaseItemProps) => {
  return (
    <CommandItem
      key={triggerId}
      value={triggerId}
      className='gap-x-4 py-3'
      onSelect={onSelect}
    >
      <div className="shrink-0">
        {icon}
      </div>
      <div className="flex flex-1 items-center justify-between min-w-0">
        <div className="flex flex-col min-w-0">
          <span className="text-sm font-medium truncate">{title}</span>
          {subtitle && (
            <span className="text-xs text-muted-foreground truncate">{subtitle}</span>
          )}
          {children}
        </div>
        {badge && (
          <span className="text-xs text-muted-foreground bg-border px-2 py-1 rounded shrink-0 ml-2">
            {badge}
          </span>
        )}
      </div>
    </CommandItem>
  );
};