"use client";

import { useEffect, useState, useCallback } from "react";
import { RefreshCw, RotateCcw, Terminal } from "lucide-react";
import { api, type Deploy, type Service } from "@/lib/api";
import { StatusBadge, SlotBadge } from "@/components/status-badge";
import { shortSHA, relativeTime, cn } from "@/lib/utils";

const LIVE_STATUSES = new Set(["pending", "building"]);

export function DeployList({ service }: { service: Service }) {
  const [deploys, setDeploys] = useState<Deploy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rolling, setRolling] = useState(false);
  const [rollbackMsg, setRollbackMsg] = useState<string | null>(null);

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
    setRollbackMsg(null);
    fetchDeploys();
  }, [fetchDeploys]);

  // Poll while any deploy is in a live state
  useEffect(() => {
    const hasLive = deploys.some((d) => LIVE_STATUSES.has(d.Status));
    if (!hasLive) return;
    const id = setInterval(fetchDeploys, 3000);
    return () => clearInterval(id);
  }, [deploys, fetchDeploys]);

  async function handleRollback() {
    setRolling(true);
    setRollbackMsg(null);
    try {
      await api.rollback(service.id);
      setRollbackMsg("$ rollback triggered");
      await fetchDeploys();
    } catch (e) {
      setRollbackMsg(
        `$ error: ${e instanceof Error ? e.message : "rollback failed"}`
      );
    } finally {
      setRolling(false);
    }
  }

  // Show rollback only if there are 2+ deploys and no deploy is currently live
  const activeDeploy = deploys.find((d) => d.Status === "active");
  const hasLive = deploys.some((d) => LIVE_STATUSES.has(d.Status));
  const canRollback =
    deploys.length >= 2 && activeDeploy != null && !hasLive;

  return (
    <div className="flex flex-col gap-4">
      {/* Header row */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-[#4a6048] uppercase tracking-widest">
          <Terminal className="w-3.5 h-3.5" />
          deploys
        </div>
        <div className="flex items-center gap-3">
          {canRollback && (
            <button
              onClick={handleRollback}
              disabled={rolling}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1 text-xs font-mono border rounded",
                "border-[#a78bfa]/30 text-[#a78bfa] hover:bg-[#a78bfa]/10",
                "transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              )}
            >
              <RotateCcw className="w-3 h-3" />
              {rolling ? "rolling back..." : "rollback"}
            </button>
          )}
          <button
            onClick={fetchDeploys}
            className="text-[#4a6048] hover:text-[#00ff41] transition-colors p-1"
            title="Refresh"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {rollbackMsg && (
        <div
          className={cn(
            "px-3 py-2 text-xs font-mono border rounded",
            rollbackMsg.includes("error")
              ? "text-[#ef4444] border-[#ef4444]/20 bg-[#ef4444]/5"
              : "text-[#00ff41] border-[#00ff41]/20 bg-[#00ff41]/5"
          )}
        >
          {rollbackMsg}
        </div>
      )}

      {loading && (
        <div className="text-xs text-[#4a6048] py-4 text-center">
          loading<span className="cursor-blink">_</span>
        </div>
      )}

      {error && (
        <div className="text-xs text-[#ef4444] border border-[#ef4444]/20 bg-[#ef4444]/5 px-3 py-2 rounded">
          $ error: {error}
        </div>
      )}

      {!loading && !error && deploys.length === 0 && (
        <div className="text-xs text-[#4a6048] py-6 text-center border border-dashed border-[#1a2318] rounded">
          no deploys yet — push to trigger one
        </div>
      )}

      {deploys.length > 0 && (
        <div className="flex flex-col border border-[#1a2318] rounded overflow-hidden">
          {/* Table header */}
          <div className="grid grid-cols-[7ch_1fr_auto_auto_auto] gap-3 px-4 py-2 text-[10px] text-[#4a6048] uppercase tracking-widest border-b border-[#1a2318] bg-[#0a0c0a]">
            <span>sha</span>
            <span>message</span>
            <span>status</span>
            <span>slot</span>
            <span>when</span>
          </div>

          {deploys.map((deploy, i) => {
            const isActive = deploy.Status === "active";
            const isLive = LIVE_STATUSES.has(deploy.Status);
            return (
              <div
                key={deploy.ID}
                className={cn(
                  "grid grid-cols-[7ch_1fr_auto_auto_auto] gap-3 items-center px-4 py-2.5 text-xs",
                  "border-b border-[#1a2318] last:border-b-0",
                  isActive && "bg-[#00ff41]/[0.02]",
                  isLive && "bg-[#f59e0b]/[0.02]",
                  i > 0 && "opacity-75"
                )}
              >
                <code
                  className={cn(
                    "font-mono tracking-tight",
                    isActive ? "text-[#00ff41]" : "text-[#4a6048]"
                  )}
                >
                  {shortSHA(deploy.CommitSHA)}
                </code>

                <span
                  className="truncate text-[#8aae8a]"
                  title={deploy.CommitMessage}
                >
                  {deploy.CommitMessage || "—"}
                </span>

                <StatusBadge status={deploy.Status} />
                <SlotBadge slot={deploy.Slot} />

                <span className="text-[#4a6048] whitespace-nowrap text-right">
                  {relativeTime(deploy.CreatedAt)}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Live indicator */}
      {hasLive && (
        <div className="flex items-center gap-2 text-xs text-[#f59e0b]">
          <span className="w-1.5 h-1.5 rounded-full bg-[#f59e0b] animate-pulse inline-block" />
          build in progress — polling every 3s
        </div>
      )}
    </div>
  );
}
