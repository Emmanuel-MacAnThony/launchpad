import { cn } from "@/lib/utils";
import type { DeployStatus, Slot } from "@/lib/api";

const statusConfig: Record<
  DeployStatus,
  { label: string; className: string; dot?: string }
> = {
  active: {
    label: "active",
    className: "text-[#00ff41] border-[#00ff41]/30 bg-[#00ff41]/5",
    dot: "bg-[#00ff41] pulse-active",
  },
  building: {
    label: "building",
    className: "text-[#f59e0b] border-[#f59e0b]/30 bg-[#f59e0b]/5",
    dot: "bg-[#f59e0b]",
  },
  pending: {
    label: "pending",
    className: "text-[#6b7280] border-[#6b7280]/30 bg-[#6b7280]/5",
    dot: "bg-[#6b7280]",
  },
  failed: {
    label: "failed",
    className: "text-[#ef4444] border-[#ef4444]/30 bg-[#ef4444]/5",
    dot: "bg-[#ef4444]",
  },
  rolled_back: {
    label: "rolled_back",
    className: "text-[#a78bfa] border-[#a78bfa]/30 bg-[#a78bfa]/5",
    dot: "bg-[#a78bfa]",
  },
};

export function StatusBadge({
  status,
  className,
}: {
  status: DeployStatus;
  className?: string;
}) {
  const cfg = statusConfig[status] ?? statusConfig.pending;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 text-xs font-mono border rounded",
        cfg.className,
        className
      )}
    >
      <span className={cn("w-1.5 h-1.5 rounded-full shrink-0", cfg.dot)} />
      {cfg.label}
    </span>
  );
}

const slotConfig: Record<Slot, { className: string }> = {
  blue: { className: "text-[#60a5fa] border-[#60a5fa]/30 bg-[#60a5fa]/5" },
  green: { className: "text-[#00ff41] border-[#00ff41]/30 bg-[#00ff41]/5" },
};

export function SlotBadge({
  slot,
  className,
}: {
  slot: Slot | null;
  className?: string;
}) {
  if (!slot) return <span className="text-[#4a6048] text-xs">—</span>;
  const cfg = slotConfig[slot];
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 text-xs font-mono border rounded",
        cfg.className,
        className
      )}
    >
      {slot}
    </span>
  );
}
