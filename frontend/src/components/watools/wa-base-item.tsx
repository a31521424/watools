import { ReactNode } from "react";
import { CommandItem } from "@/components/ui/command";

export interface BaseItemProps {
  id: string;
  triggerId: string;
  name: string;
  icon: ReactNode;
  score: number;
  onSelect: () => void;
  children?: ReactNode;
}

export const WaBaseItem = ({ triggerId, icon, name, onSelect, children }: BaseItemProps) => {
  return (
    <CommandItem
      key={triggerId}
      value={triggerId}
      className='gap-x-4'
      onSelect={onSelect}
    >
      {icon}
      <div className="flex flex-col flex-1">
        <span>{name}</span>
        {children}
      </div>
    </CommandItem>
  );
};