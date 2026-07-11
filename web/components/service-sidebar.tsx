"use client";

import { useEffect, useState, useCallback } from "react";
import { Rocket, Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { api, type Service } from "@/lib/api";
import { CreateServiceModal } from "@/components/create-service-modal";
import { cn } from "@/lib/utils";

interface Props {
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function ServiceSidebar({ selectedId, onSelect }: Props) {
  const [services, setServices]     = useState<Service[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [collapsed, setCollapsed]   = useState(false);

  const fetchServices = useCallback(() => {
    api
      .listServices()
      .then((data) => setServices(data ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : "failed to load"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchServices(); }, [fetchServices]);

  function handleCreated(id: string) {
    setShowCreate(false);
    fetchServices();
    onSelect(id);
  }

  return (
    <>
      <aside
        className={cn(
          "shrink-0 flex flex-col border-r border-[#1c1c1f] bg-[#09090b] overflow-hidden",
          "transition-[width] duration-250 ease-in-out",
          collapsed ? "w-[52px]" : "w-[220px]"
        )}
        style={{ height: "100%" }}
      >
        {/* Brand header — clicking toggles collapse */}
        <button
          onClick={() => setCollapsed((c) => !c)}
          className={cn(
            "flex items-center border-b border-[#1c1c1f] w-full shrink-0",
            "hover:bg-[#111113] transition-colors cursor-pointer",
            collapsed ? "justify-center px-0 py-4" : "gap-2.5 px-4 py-4"
          )}
        >
          <div className="w-6 h-6 rounded-md bg-[#6366f1]/10 border border-[#6366f1]/20 flex items-center justify-center shrink-0">
            <Rocket className="w-3.5 h-3.5 text-[#6366f1]" />
          </div>
          {!collapsed && (
            <span className="text-[13px] font-semibold text-[#f4f4f5] tracking-tight flex-1 text-left whitespace-nowrap">
              Launchpad
            </span>
          )}
        </button>

        {/* Nav body */}
        <div className="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar">

          {/* Section header row */}
          {!collapsed ? (
            <div className="flex items-center justify-between px-4 pt-4 pb-1.5">
              <span className="text-[10px] font-medium text-[#3f3f46] uppercase tracking-widest">
                Services
              </span>
              <button
                onClick={() => setShowCreate(true)}
                className="w-5 h-5 rounded flex items-center justify-center text-[#3f3f46] hover:text-[#6366f1] hover:bg-[#6366f1]/10 transition-colors"
                title="Register service"
              >
                <Plus className="w-3 h-3" />
              </button>
            </div>
          ) : (
            <div className="flex justify-center pt-3 pb-1">
              <button
                onClick={(e) => { e.stopPropagation(); setShowCreate(true); }}
                className="w-8 h-8 rounded flex items-center justify-center text-[#3f3f46] hover:text-[#6366f1] hover:bg-[#6366f1]/10 transition-colors"
                title="Register service"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>
            </div>
          )}

          {/* Loading / error / empty */}
          {loading && !collapsed && (
            <div className="px-4 py-2 text-[12px] text-[#3f3f46]">Loading...</div>
          )}
          {error && !collapsed && (
            <div className="px-4 py-2 text-[12px] text-[#ef4444]">{error}</div>
          )}
          {!loading && !error && services.length === 0 && !collapsed && (
            <div className="px-4 py-3">
              <p className="text-[12px] text-[#52525b] mb-2.5">No services yet.</p>
              <button
                onClick={() => setShowCreate(true)}
                className="w-full text-[12px] px-3 py-1.5 rounded border border-[#27272a] bg-[#111113] text-[#71717a] hover:text-[#f4f4f5] hover:border-[#3f3f46] transition-colors text-left"
              >
                + Register one
              </button>
            </div>
          )}

          {/* Service list — no outer padding; items span edge-to-edge */}
          <div className="mt-0.5">
            {services.map((svc) => {
              const isSelected = svc.id === selectedId;
              const isActive   = svc.active_slot != null;

              const dot = (
                <span
                  className={cn(
                    "rounded-full shrink-0 transition-colors",
                    collapsed ? "w-2 h-2" : "w-1.5 h-1.5",
                    isActive
                      ? isSelected
                        ? "bg-[#6366f1] pulse-active"
                        : "bg-[#6366f1]/60"
                      : "bg-[#27272a]"
                  )}
                />
              );

              if (collapsed) {
                return (
                  <button
                    key={svc.id}
                    onClick={() => onSelect(svc.id)}
                    title={svc.name}
                    className={cn(
                      "relative w-full flex items-center justify-center h-9 transition-colors",
                      isSelected ? "bg-[#6366f1]/[0.08]" : "hover:bg-[#111113]"
                    )}
                  >
                    {isSelected && (
                      <span className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r-full bg-[#6366f1]" />
                    )}
                    {dot}
                  </button>
                );
              }

              return (
                <button
                  key={svc.id}
                  onClick={() => onSelect(svc.id)}
                  className={cn(
                    "relative w-full flex items-center gap-2.5 pl-4 pr-3 py-2 text-left transition-colors",
                    isSelected
                      ? "bg-[#6366f1]/[0.08] text-[#f4f4f5]"
                      : "text-[#71717a] hover:bg-[#111113] hover:text-[#d4d4d8]"
                  )}
                >
                  {isSelected && (
                    <span className="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r-full bg-[#6366f1]" />
                  )}
                  {dot}
                  <span className="text-[13px] truncate font-mono capitalize">{svc.name}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Footer collapse toggle */}
        <div className="border-t border-[#1c1c1f] shrink-0">
          <button
            onClick={() => setCollapsed((c) => !c)}
            className={cn(
              "w-full flex items-center py-2.5 text-[#3f3f46] hover:text-[#71717a] hover:bg-[#111113] transition-colors",
              collapsed ? "justify-center" : "px-4 gap-2"
            )}
          >
            {collapsed ? (
              <ChevronRight className="w-3.5 h-3.5" />
            ) : (
              <>
                <ChevronLeft className="w-3.5 h-3.5" />
                <span className="text-[11px] whitespace-nowrap">Collapse</span>
              </>
            )}
          </button>
        </div>
      </aside>

      {showCreate && (
        <CreateServiceModal
          onClose={() => setShowCreate(false)}
          onCreated={handleCreated}
        />
      )}
    </>
  );
}
