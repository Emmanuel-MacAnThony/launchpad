import { cn } from "@/lib/utils";
import type { DeployStatus, Slot } from "@/lib/api";

const statusConfig: Record<DeployStatus, { label: string; dot: string; text: string; bg: string; border: string }> = {
  active:      { label: "active",      dot: "bg-[#6366f1] pulse-active",   text: "text-[#6366f1]", bg: "bg-[#6366f1]/[0.07]",  border: "border-[#6366f1]/25" },
  building:    { label: "building",    dot: "bg-[#f59e0b] animate-pulse",  text: "text-[#f59e0b]", bg: "bg-[#f59e0b]/[0.07]",  border: "border-[#f59e0b]/25" },
  pending:     { label: "pending",     dot: "bg-[#71717a]",                text: "text-[#71717a]", bg: "bg-[#71717a]/[0.07]",  border: "border-[#71717a]/25" },
  failed:      { label: "failed",      dot: "bg-[#ef4444]",                text: "text-[#ef4444]", bg: "bg-[#ef4444]/[0.07]",  border: "border-[#ef4444]/25" },
  rolled_back: { label: "rolled back", dot: "bg-[#a78bfa]",                text: "text-[#a78bfa]", bg: "bg-[#a78bfa]/[0.07]",  border: "border-[#a78bfa]/25" },
};

export function StatusBadge({ status, className }: { status: DeployStatus; className?: string }) {
  const c = statusConfig[status] ?? statusConfig.pending;
  return (
    <span className={cn("inline-flex items-center gap-1.5 px-2 py-0.5 text-[11px] font-mono rounded border", c.text, c.bg, c.border, className)}>
      <span className={cn("w-1.5 h-1.5 rounded-full shrink-0", c.dot)} />
      {c.label}
    </span>
  );
}

const slotConfig: Record<Slot, { text: string; bg: string; border: string }> = {
  blue:  { text: "text-[#60a5fa]", bg: "bg-[#60a5fa]/[0.07]", border: "border-[#60a5fa]/25" },
  green: { text: "text-[#2dd4bf]", bg: "bg-[#2dd4bf]/[0.07]", border: "border-[#2dd4bf]/25" },
};

export function SlotBadge({ slot, className }: { slot: Slot | null; className?: string }) {
  if (!slot) return <span className="text-[#52525b] text-[11px]">—</span>;
  const c = slotConfig[slot];
  return (
    <span className={cn("inline-flex items-center px-2 py-0.5 text-[11px] font-mono rounded border", c.text, c.bg, c.border, className)}>
      {slot}
    </span>
  );
}
