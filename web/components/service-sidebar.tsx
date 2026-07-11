"use client";

import { useEffect, useState, useCallback } from "react";
import { Rocket, Plus } from "lucide-react";
import { api, type Service } from "@/lib/api";
import { CreateServiceModal } from "@/components/create-service-modal";
import { cn } from "@/lib/utils";

interface Props {
  selectedId: string | null;
  onSelect: (id: string) => void;
}

export function ServiceSidebar({ selectedId, onSelect }: Props) {
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const fetchServices = useCallback(() => {
    api
      .listServices()
      .then((data) => setServices(data ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : "failed to load"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

  function handleCreated(id: string) {
    setShowCreate(false);
    fetchServices();
    onSelect(id);
  }

  return (
    <>
      <aside
        className="w-56 shrink-0 flex flex-col border-r border-[#1a2318] bg-[#080a08]"
        style={{ height: "100%" }}
      >
        {/* Header */}
        <div className="px-4 py-4 border-b border-[#1a2318]">
          <div className="flex items-center gap-2">
            <Rocket className="w-4 h-4 text-[#00ff41]" />
            <span className="text-sm font-mono text-[#00ff41] tracking-tight">
              launchpad
            </span>
          </div>
          <div className="text-[10px] text-[#4a6048] mt-0.5 tracking-widest uppercase">
            ci/cd
          </div>
        </div>

        {/* Section label + add button */}
        <div className="flex items-center justify-between px-4 pt-4 pb-2">
          <span className="text-[10px] text-[#4a6048] uppercase tracking-widest">
            services
          </span>
          <button
            onClick={() => setShowCreate(true)}
            className="text-[#2a3a28] hover:text-[#00ff41] transition-colors"
            title="Register service"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto px-2">
          {loading && (
            <div className="px-2 py-3 text-xs text-[#4a6048]">
              loading<span className="cursor-blink">_</span>
            </div>
          )}

          {error && (
            <div className="px-2 py-2 text-xs text-[#ef4444]">
              err: {error}
            </div>
          )}

          {!loading && !error && services.length === 0 && (
            <div className="px-2 py-3 text-xs text-[#4a6048] leading-relaxed">
              no services yet
              <br />
              <button
                onClick={() => setShowCreate(true)}
                className="text-[#00ff41]/60 hover:text-[#00ff41] transition-colors underline underline-offset-2"
              >
                register one
              </button>
            </div>
          )}

          {services.map((svc) => {
            const isSelected = svc.id === selectedId;
            const isActive = svc.active_slot != null;
            return (
              <button
                key={svc.id}
                onClick={() => onSelect(svc.id)}
                className={cn(
                  "w-full flex items-center gap-2.5 px-3 py-2.5 rounded text-left text-xs",
                  "transition-colors duration-100",
                  isSelected
                    ? "bg-[#00ff41]/10 text-[#00ff41]"
                    : "text-[#6b8a6b] hover:bg-[#0f1a0f] hover:text-[#a8c8a8]"
                )}
              >
                {/* Active dot: green pulse if deployed, dim circle if never deployed */}
                <span
                  className={cn(
                    "w-2 h-2 rounded-full shrink-0",
                    isActive
                      ? isSelected
                        ? "bg-[#00ff41] pulse-active"
                        : "bg-[#00ff41]/60"
                      : isSelected
                      ? "bg-[#00ff41]/30"
                      : "bg-[#1a2318]"
                  )}
                />
                <span className="truncate font-mono">{svc.name}</span>
              </button>
            );
          })}
        </div>

        {/* Footer */}
        <div className="px-4 py-3 border-t border-[#1a2318]">
          <div className="text-[10px] text-[#2a3a28] font-mono">
            api: localhost:8090
          </div>
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
