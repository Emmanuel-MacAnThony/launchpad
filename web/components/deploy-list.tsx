"use client";

import { useEffect, useState, useCallback } from "react";
import { RefreshCw, RotateCcw, ChevronDown, ChevronRight } from "lucide-react";
import { api, type Deploy, type Service } from "@/lib/api";
import { StatusBadge, SlotBadge } from "@/components/status-badge";
import { shortSHA, relativeTime, cn } from "@/lib/utils";

const LIVE = new Set(["pending", "building"]);

const SLOT_COLOR = { blue: "#60a5fa", green: "#2dd4bf" } as const;

function duration(a: string, b?: string | null): string {
  if (!b) return "—";
  const ms = new Date(b).getTime() - new Date(a).getTime();
  if (ms < 0) return "—";
  const s = Math.round(ms / 1000);
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
}

function latestForSlot(deploys: Deploy[], slot: "blue" | "green"): Deploy | null {
  return deploys
    .filter((d) => d.Slot === slot)
    .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime())[0] ?? null;
}

const SLOT_RGB = { blue: "96,165,250", green: "45,212,191" } as const;

function SlotCard({
  slot, deploy, isActive, canRollback, rolling, onRollback,
}: {
  slot: "blue" | "green";
  deploy: Deploy | null;
  isActive: boolean;
  canRollback: boolean;
  rolling: boolean;
  onRollback: () => void;
}) {
  const color = SLOT_COLOR[slot];
  const rgb   = SLOT_RGB[slot];

  return (
    <div
      className={cn("relative flex-1 overflow-hidden min-w-0", isActive && "slot-activate")}
      style={{
        border: `1px solid ${isActive ? "rgba(255,255,255,0.12)" : "rgba(255,255,255,0.06)"}`,
        backgroundColor: "#0f0f12",
        backgroundImage: [
          `radial-gradient(ellipse 120% 60% at 50% 0%, rgba(${rgb},${isActive ? "0.12" : "0.05"}) 0%, transparent 70%)`,
          "linear-gradient(rgba(255,255,255,0.032) 1px, transparent 1px)",
          "linear-gradient(90deg, rgba(255,255,255,0.032) 1px, transparent 1px)",
        ].join(", "),
        backgroundSize: "100% 100%, 24px 24px, 24px 24px",
        boxShadow: isActive
          ? "inset 0 1px 0 rgba(255,255,255,0.08), 0 4px 24px rgba(0,0,0,0.5), 0 16px 48px rgba(0,0,0,0.4)"
          : "inset 0 1px 0 rgba(255,255,255,0.04), 0 4px 16px rgba(0,0,0,0.4)",
      }}
    >

      <div className="p-6 flex flex-col gap-5">

        {/* Header: slot name + live indicator */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <SlotBadge slot={slot} />
            {isActive ? (
              <span className="flex items-center gap-1.5 text-[11px] font-semibold tracking-wide uppercase"
                style={{ color }}>
                <span className="w-1.5 h-1.5 rounded-full pulse-active" style={{ backgroundColor: color }} />
                Live
              </span>
            ) : (
              <span className="text-[11px] text-[#3f3f46] uppercase tracking-wide font-medium">Standby</span>
            )}
          </div>
          {isActive && canRollback && (
            <button
              onClick={onRollback}
              disabled={rolling}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1 text-[11px] rounded border font-medium transition-all",
                rolling
                  ? "border-[#a78bfa]/60 text-[#a78bfa] bg-[#a78bfa]/[0.15] cursor-not-allowed"
                  : "border-[#a78bfa]/35 text-[#a78bfa] bg-[#a78bfa]/[0.08] cursor-pointer hover:bg-[#a78bfa]/[0.16] hover:border-[#a78bfa]/55",
              )}
            >
              <RotateCcw className={cn("w-3 h-3 shrink-0", rolling && "animate-spin")} />
              <span>{rolling ? "Rolling back…" : "Rollback"}</span>
            </button>
          )}
        </div>

        {/* Divider */}
        <div className="h-px" style={{ background: `linear-gradient(90deg, rgba(${rgb},0.3), rgba(255,255,255,0.04) 60%, transparent)` }} />

        {deploy ? (
          <>
            {/* SHA + status */}
            <div className="flex items-center justify-between gap-2">
              <code
                className="text-[15px] font-mono font-bold tracking-tight"
                style={{ color, textShadow: `0 0 20px rgba(${rgb},0.5)` }}
              >
                {deploy.CommitSHA.slice(0, 7)}
              </code>
              <StatusBadge status={deploy.Status} />
            </div>

            {/* Commit message */}
            <p
              className="text-[12px] text-[#71717a] leading-relaxed line-clamp-2 min-h-[2.5em]"
              title={deploy.CommitMessage}
            >
              {deploy.CommitMessage || "No commit message"}
            </p>

            {/* Footer: time + duration */}
            <div className="flex items-center justify-between pt-1 border-t border-white/[0.04]">
              <span className="text-[11px] text-[#3f3f46]">
                {relativeTime(deploy.FinishedAt ?? deploy.CreatedAt)}
              </span>
              {deploy.StartedAt && deploy.FinishedAt && (
                <span className="text-[11px] text-[#3f3f46] font-mono">
                  {duration(deploy.StartedAt, deploy.FinishedAt)}
                </span>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center py-6">
            <span className="text-[12px] text-[#3f3f46]">No deployment yet</span>
          </div>
        )}
      </div>
    </div>
  );
}

function DeployDetail({ deploy }: { deploy: Deploy }) {
  const color = SLOT_COLOR[deploy.Slot as "blue" | "green"] ?? "#a1a1aa";
  return (
    <div className="px-4 py-3 bg-[#0a0a0c] border-b border-[#1c1c1f] grid grid-cols-2 gap-x-8 gap-y-2">
      <div>
        <span className="text-[10px] text-[#3f3f46] uppercase tracking-wider">Full SHA</span>
        <p className="text-[12px] font-mono mt-0.5 break-all" style={{ color }}>{deploy.CommitSHA}</p>
      </div>
      <div>
        <span className="text-[10px] text-[#3f3f46] uppercase tracking-wider">Duration</span>
        <p className="text-[12px] font-mono text-[#a1a1aa] mt-0.5">
          {duration(deploy.CreatedAt, deploy.FinishedAt)}
        </p>
      </div>
      <div>
        <span className="text-[10px] text-[#3f3f46] uppercase tracking-wider">Started</span>
        <p className="text-[12px] text-[#a1a1aa] mt-0.5">{relativeTime(deploy.CreatedAt)}</p>
      </div>
      {deploy.FinishedAt && (
        <div>
          <span className="text-[10px] text-[#3f3f46] uppercase tracking-wider">Finished</span>
          <p className="text-[12px] text-[#a1a1aa] mt-0.5">{relativeTime(deploy.FinishedAt)}</p>
        </div>
      )}
      {deploy.CommitMessage && (
        <div className="col-span-2">
          <span className="text-[10px] text-[#3f3f46] uppercase tracking-wider">Commit</span>
          <p className="text-[12px] text-[#a1a1aa] mt-0.5 break-all">{deploy.CommitMessage}</p>
        </div>
      )}
    </div>
  );
}

export function DeployList({
  service,
  onServiceRefresh,
}: {
  service: Service;
  onServiceRefresh?: () => void | Promise<void>;
}) {
  const [deploys, setDeploys]       = useState<Deploy[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [rolling, setRolling]       = useState(false);
  const [rollMsg, setRollMsg]       = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchDeploys = useCallback(async () => {
    try {
      const data = await api.listDeploys(service.id);
      setDeploys(data ?? []);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed to load deploys");
    } finally {
      setLoading(false);
    }
  }, [service.id]);

  useEffect(() => {
    setLoading(true);
    setDeploys([]);
    setError(null);
    setRollMsg(null);
    setExpandedId(null);
    fetchDeploys();
  }, [fetchDeploys]);

  useEffect(() => {
    const hasLive = deploys.some((d) => LIVE.has(d.Status));
    if (!hasLive) return;
    // While a build is in flight, poll deploys and the service together so the
    // Live card flips the moment activation lands, without a manual refresh.
    const id = setInterval(() => {
      fetchDeploys();
      onServiceRefresh?.();
    }, 3000);
    return () => clearInterval(id);
  }, [deploys, fetchDeploys, onServiceRefresh]);

  async function handleRollback() {
    setRolling(true);
    setRollMsg(null);
    try {
      await api.rollback(service.id);
      // Refresh deploys AND the service — active_slot lives on the service, and
      // the cards derive which slot is Live from it. Skipping this leaves the
      // old slot flagged active until a manual page refresh.
      await Promise.all([fetchDeploys(), onServiceRefresh?.()]);
    } catch (e) {
      setRollMsg(e instanceof Error ? e.message : "Rollback failed");
    } finally {
      setRolling(false);
    }
  }

  // service.active_slot is authoritative — it reflects nginx's current target.
  // deploy.Status === "active" is NOT updated on rollback, so it can lie after a swap.
  const activeSlot   = service.active_slot as "blue" | "green" | undefined;
  const hasLive      = deploys.some((d) => LIVE.has(d.Status));
  const blueLatest   = latestForSlot(deploys, "blue");
  const greenLatest  = latestForSlot(deploys, "green");
  // Rollback uses GetLatestOnSlot for both slots — no status=active requirement.
  // Both slots must have at least one deploy, and nothing can be building.
  const canRollback = activeSlot != null && blueLatest != null && greenLatest != null && !hasLive;

  return (
    <div className="flex flex-col gap-4">

      {/* Two slot cards */}
      <div className="flex gap-3">
        <SlotCard
          slot="blue"
          deploy={blueLatest}
          isActive={activeSlot === "blue"}
          canRollback={canRollback}
          rolling={rolling}
          onRollback={handleRollback}
        />
        <SlotCard
          slot="green"
          deploy={greenLatest}
          isActive={activeSlot === "green"}
          canRollback={canRollback}
          rolling={rolling}
          onRollback={handleRollback}
        />
      </div>

      {rollMsg && (
        <div className={cn(
          "px-3 py-2 text-[12px] rounded border",
          rollMsg.toLowerCase().includes("fail") || rollMsg.toLowerCase().includes("error")
            ? "text-[#ef4444] border-[#ef4444]/20 bg-[#ef4444]/[0.05]"
            : "text-[#6366f1] border-[#6366f1]/20 bg-[#6366f1]/[0.05]"
        )}>
          {rollMsg}
        </div>
      )}

      {/* Deploy history */}
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-medium text-[#52525b] uppercase tracking-wider">
          History
        </span>
        <button
          onClick={fetchDeploys}
          className="p-1 text-[#52525b] hover:text-[#a1a1aa] transition-colors rounded cursor-pointer"
          title="Refresh"
        >
          <RefreshCw className="w-3.5 h-3.5" />
        </button>
      </div>

      {loading && (
        <div className="text-[12px] text-[#52525b] py-4 text-center">Loading...</div>
      )}
      {error && (
        <div className="text-[12px] text-[#ef4444] border border-[#ef4444]/20 bg-[#ef4444]/[0.05] px-3 py-2 rounded">
          {error}
        </div>
      )}
      {!loading && !error && deploys.length === 0 && (
        <div className="text-[12px] text-[#52525b] py-8 text-center border border-dashed border-[#27272a] rounded-lg">
          No deploys yet — push to trigger one
        </div>
      )}

      {deploys.length > 0 && (
        <div className="rounded-lg border border-[#1c1c1f] overflow-hidden">
          <div className="grid grid-cols-[16px_7ch_1fr_auto_auto_auto] gap-3 px-4 py-2 bg-[#111113] border-b border-[#1c1c1f] text-[11px] font-medium text-[#52525b] uppercase tracking-wider">
            <span />
            <span>SHA</span>
            <span>Message</span>
            <span>Status</span>
            <span>Slot</span>
            <span className="text-right">When</span>
          </div>

          {deploys.map((deploy, i) => {
            const isActive   = deploy.Status === "active";
            const isLive     = LIVE.has(deploy.Status);
            const isExpanded = expandedId === deploy.ID;
            return (
              <div key={deploy.ID}>
                <div
                  onClick={() => setExpandedId(isExpanded ? null : deploy.ID)}
                  className={cn(
                    "grid grid-cols-[16px_7ch_1fr_auto_auto_auto] gap-3 items-center px-4 py-2.5",
                    "border-b border-[#1c1c1f] last:border-b-0 text-[12px]",
                    "cursor-pointer select-none transition-colors",
                    isActive ? "bg-[#6366f1]/[0.04] hover:bg-[#6366f1]/[0.07]" :
                    isLive   ? "bg-[#f59e0b]/[0.04] hover:bg-[#f59e0b]/[0.07]" :
                               "hover:bg-[#111113]",
                    i > 0 && !isActive && !isLive && "opacity-55 hover:opacity-100"
                  )}
                >
                  <span className="text-[#3f3f46]">
                    {isExpanded
                      ? <ChevronDown  className="w-3 h-3" />
                      : <ChevronRight className="w-3 h-3" />
                    }
                  </span>
                  <code className={cn("font-mono text-[12px]", isActive ? "text-[#6366f1]" : "text-[#52525b]")}>
                    {shortSHA(deploy.CommitSHA)}
                  </code>
                  <span className="truncate text-[#a1a1aa]" title={deploy.CommitMessage}>
                    {deploy.CommitMessage || "—"}
                  </span>
                  <StatusBadge status={deploy.Status} />
                  <SlotBadge   slot={deploy.Slot} />
                  <span className="text-[#52525b] whitespace-nowrap text-right">
                    {relativeTime(deploy.CreatedAt)}
                  </span>
                </div>
                {isExpanded && <DeployDetail deploy={deploy} />}
              </div>
            );
          })}
        </div>
      )}

      {hasLive && (
        <div className="flex items-center gap-2 text-[12px] text-[#f59e0b]">
          <span className="w-1.5 h-1.5 rounded-full bg-[#f59e0b] animate-pulse" />
          Build in progress — polling every 3s
        </div>
      )}
    </div>
  );
}
