"use client";

import { useEffect, useState, useCallback } from "react";
import { RefreshCw, RotateCcw } from "lucide-react";
import { api, type Deploy, type Service } from "@/lib/api";
import { StatusBadge, SlotBadge } from "@/components/status-badge";
import { shortSHA, relativeTime, cn } from "@/lib/utils";

const LIVE = new Set(["pending", "building"]);

export function DeployList({ service }: { service: Service }) {
  const [deploys, setDeploys]       = useState<Deploy[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [rolling, setRolling]       = useState(false);
  const [rollMsg, setRollMsg]       = useState<string | null>(null);

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
    fetchDeploys();
  }, [fetchDeploys]);

  useEffect(() => {
    const hasLive = deploys.some((d) => LIVE.has(d.Status));
    if (!hasLive) return;
    const id = setInterval(fetchDeploys, 3000);
    return () => clearInterval(id);
  }, [deploys, fetchDeploys]);

  async function handleRollback() {
    setRolling(true);
    setRollMsg(null);
    try {
      await api.rollback(service.id);
      setRollMsg("Rollback triggered");
      await fetchDeploys();
    } catch (e) {
      setRollMsg(e instanceof Error ? e.message : "Rollback failed");
    } finally {
      setRolling(false);
    }
  }

  const activeDeploy = deploys.find((d) => d.Status === "active");
  const hasLive      = deploys.some((d) => LIVE.has(d.Status));
  const canRollback  = deploys.length >= 2 && activeDeploy != null && !hasLive;

  return (
    <div className="flex flex-col gap-3">
      {/* Section header */}
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-medium text-[#52525b] uppercase tracking-wider">
          Deploys
        </span>
        <div className="flex items-center gap-2">
          {canRollback && (
            <button
              onClick={handleRollback}
              disabled={rolling}
              className={cn(
                "flex items-center gap-1.5 px-2.5 py-1 text-[12px] rounded border",
                "border-[#a78bfa]/30 text-[#a78bfa] bg-[#a78bfa]/[0.06]",
                "hover:bg-[#a78bfa]/[0.12] transition-colors",
                "disabled:opacity-40 disabled:cursor-not-allowed"
              )}
            >
              <RotateCcw className="w-3 h-3" />
              {rolling ? "Rolling back..." : "Rollback"}
            </button>
          )}
          <button
            onClick={fetchDeploys}
            className="p-1 text-[#52525b] hover:text-[#a1a1aa] transition-colors rounded"
            title="Refresh"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {rollMsg && (
        <div
          className={cn(
            "px-3 py-2 text-[12px] rounded border",
            rollMsg.toLowerCase().includes("fail") || rollMsg.toLowerCase().includes("error")
              ? "text-[#ef4444] border-[#ef4444]/20 bg-[#ef4444]/[0.05]"
              : "text-[#6366f1] border-[#6366f1]/20 bg-[#6366f1]/[0.05]"
          )}
        >
          {rollMsg}
        </div>
      )}

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
          {/* Table head */}
          <div className="grid grid-cols-[7ch_1fr_auto_auto_auto] gap-4 px-4 py-2 bg-[#111113] border-b border-[#1c1c1f] text-[11px] font-medium text-[#52525b] uppercase tracking-wider">
            <span>SHA</span>
            <span>Message</span>
            <span>Status</span>
            <span>Slot</span>
            <span className="text-right">When</span>
          </div>

          {deploys.map((deploy, i) => {
            const isActive = deploy.Status === "active";
            const isLive   = LIVE.has(deploy.Status);
            return (
              <div
                key={deploy.ID}
                className={cn(
                  "grid grid-cols-[7ch_1fr_auto_auto_auto] gap-4 items-center px-4 py-2.5",
                  "border-b border-[#1c1c1f] last:border-b-0 text-[12px]",
                  isActive && "bg-[#6366f1]/[0.025]",
                  isLive   && "bg-[#f59e0b]/[0.025]",
                  i > 0 && !isActive && "opacity-60"
                )}
              >
                <code
                  className={cn(
                    "font-mono text-[12px]",
                    isActive ? "text-[#6366f1]" : "text-[#52525b]"
                  )}
                >
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
