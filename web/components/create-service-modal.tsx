"use client";

import { useState, useEffect } from "react";
import { X, Check } from "lucide-react";
import { api, type CreateServiceInput } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  onClose: () => void;
  onCreated: (serviceId: string) => void;
}

type Field = {
  key: keyof CreateServiceInput;
  label: string;
  placeholder: string;
  type?: "text" | "number" | "password";
};

const FIELDS: Field[] = [
  { key: "name",             label: "name",             placeholder: "my-api",                          type: "text" },
  { key: "repo_url",         label: "repo_url",         placeholder: "https://github.com/org/repo.git", type: "text" },
  { key: "domain",           label: "domain",           placeholder: "api.example.com",                 type: "text" },
  { key: "health_check_url", label: "health_check_url", placeholder: "http://api.example.com/health",   type: "text" },
  { key: "webhook_secret",   label: "webhook_secret",   placeholder: "super-secret",                    type: "password" },
  { key: "host",             label: "host",             placeholder: "1.2.3.4 or hostname",             type: "text" },
  { key: "ssh_user",         label: "ssh_user",         placeholder: "ubuntu",                          type: "text" },
  { key: "ssh_private_key",  label: "ssh_private_key",  placeholder: "-----BEGIN OPENSSH PRIVATE KEY-----", type: "text" },
  { key: "blue_port",        label: "blue_port",        placeholder: "3001",                            type: "number" },
  { key: "green_port",       label: "green_port",       placeholder: "3002",                            type: "number" },
  { key: "container_port",   label: "container_port",   placeholder: "8080",                            type: "number" },
  { key: "compose_service",  label: "compose_service",  placeholder: "app",                             type: "text" },
];

const EMPTY: Record<keyof CreateServiceInput, string> = {
  name: "", repo_url: "", domain: "", health_check_url: "",
  webhook_secret: "", host: "", ssh_user: "", ssh_private_key: "",
  blue_port: "", green_port: "", container_port: "", compose_service: "",
};

const STEPS = [
  "ssh connection",
  "docker installed",
  "nginx installed",
  "nginx config",
  "port availability",
  "registering service",
];

// Each step advances every ~900ms while the request is in flight.
// The last step stays in "running" until the API responds.
const STEP_INTERVAL_MS = 900;

type StepStatus = "waiting" | "running" | "done" | "error";

export function CreateServiceModal({ onClose, onCreated }: Props) {
  const [values, setValues] = useState<Record<keyof CreateServiceInput, string>>(EMPTY);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stepStatuses, setStepStatuses] = useState<StepStatus[]>(
    STEPS.map(() => "waiting")
  );

  // Animate steps 0..N-2 on a timer while the request is in flight.
  // The last step stays "running" until the API call resolves.
  useEffect(() => {
    if (!loading) return;

    setStepStatuses((prev) => {
      const next = [...prev];
      next[0] = "running";
      return next;
    });

    const timeouts: ReturnType<typeof setTimeout>[] = [];
    for (let i = 0; i < STEPS.length - 1; i++) {
      const t = setTimeout(() => {
        setStepStatuses((prev) => {
          const next = [...prev];
          next[i] = "done";
          next[i + 1] = "running";
          return next;
        });
      }, (i + 1) * STEP_INTERVAL_MS);
      timeouts.push(t);
    }

    return () => timeouts.forEach(clearTimeout);
  }, [loading]);

  function set(key: keyof CreateServiceInput, value: string) {
    setValues((v) => ({ ...v, [key]: value }));
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setStepStatuses(STEPS.map(() => "waiting"));
    setLoading(true);

    try {
      const svc = await api.createService({
        name: values.name.trim(),
        repo_url: values.repo_url.trim(),
        domain: values.domain.trim(),
        health_check_url: values.health_check_url.trim(),
        webhook_secret: values.webhook_secret.trim(),
        host: values.host.trim(),
        ssh_user: values.ssh_user.trim(),
        ssh_private_key: values.ssh_private_key.trim(),
        blue_port: Number(values.blue_port),
        green_port: Number(values.green_port),
        container_port: Number(values.container_port),
        compose_service: values.compose_service.trim() || "app",
      });

      // Flash all steps green, then navigate
      setStepStatuses(STEPS.map(() => "done"));
      setTimeout(() => onCreated(svc.id), 700);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "registration failed";
      setError(msg);
      setStepStatuses((prev) => {
        const next = [...prev];
        const idx = next.lastIndexOf("running");
        if (idx !== -1) next[idx] = "error";
        return next;
      });
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={loading ? undefined : onClose}
      />

      <div className="relative w-full max-w-lg bg-[#0a0b0d] border border-[#1a2318] rounded overflow-hidden shadow-2xl">
        {/* Title bar */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[#1a2318] bg-[#080a08]">
          <div className="flex items-center gap-2">
            <span className="text-[#00ff41] text-xs font-mono">$</span>
            <span className="text-[#a8c8a8] text-xs font-mono">launchpad register-service</span>
          </div>
          {!loading && (
            <button
              onClick={onClose}
              className="text-[#4a6048] hover:text-[#00ff41] transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>

        {loading ? (
          /* ── progress view ───────────────────────────────── */
          <div className="p-6">
            <div className="space-y-4">
              {STEPS.map((label, i) => (
                <StepRow key={label} label={label} status={stepStatuses[i]} index={i} />
              ))}
            </div>

            {error && (
              <>
                <div className="mt-5 px-3 py-2 text-xs font-mono text-[#ef4444] border border-[#ef4444]/20 bg-[#ef4444]/5 rounded">
                  $ error: {error}
                </div>
                <div className="flex justify-end mt-3">
                  <button
                    onClick={() => { setLoading(false); setError(null); }}
                    className="px-4 py-2 text-xs font-mono text-[#4a6048] hover:text-[#a8c8a8] transition-colors"
                  >
                    ← try again
                  </button>
                </div>
              </>
            )}
          </div>
        ) : (
          /* ── form view ───────────────────────────────────── */
          <form onSubmit={handleSubmit} className="p-4 max-h-[80vh] overflow-y-auto">
            <div className="space-y-3">
              {FIELDS.map((f) => (
                <div key={f.key} className="flex items-center gap-3">
                  <label className="text-[#4a6048] text-xs font-mono w-32 shrink-0 text-right">
                    {f.label}
                  </label>
                  <div className="flex-1 flex items-center border border-[#1a2318] rounded bg-[#0a0c0a] focus-within:border-[#00ff41]/40">
                    <span className="px-2 text-[#2a3a28] text-xs font-mono select-none">&gt;</span>
                    <input
                      type={f.type ?? "text"}
                      value={values[f.key]}
                      onChange={(e) => set(f.key, e.target.value)}
                      placeholder={f.placeholder}
                      className={cn(
                        "flex-1 bg-transparent text-xs font-mono text-[#a8c8a8] py-2 pr-3",
                        "placeholder:text-[#2a3a28] focus:outline-none"
                      )}
                    />
                  </div>
                </div>
              ))}
            </div>

            <p className="mt-4 text-[10px] text-[#2a3a28] font-mono leading-relaxed">
              the repo must have a <span className="text-[#4a6048]">docker-compose.yml</span> — set{" "}
              <span className="text-[#4a6048]">compose_service</span> to the service name launchpad should bind the port on (default: app)
            </p>

            <div className="mt-4 flex justify-end gap-3">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-xs font-mono text-[#4a6048] hover:text-[#a8c8a8] transition-colors"
              >
                cancel
              </button>
              <button
                type="submit"
                className={cn(
                  "flex items-center gap-2 px-4 py-2 text-xs font-mono rounded border",
                  "border-[#00ff41]/30 text-[#00ff41] hover:bg-[#00ff41]/10 transition-colors"
                )}
              >
                $ register
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

function StepRow({
  label,
  status,
  index,
}: {
  label: string;
  status: StepStatus;
  index: number;
}) {
  const visible = status !== "waiting";

  return (
    <div
      className={cn(
        "flex items-center gap-3 transition-all duration-200",
        visible ? "step-slide-in opacity-100" : "opacity-20"
      )}
      style={{ animationDelay: visible ? `${index * 40}ms` : undefined }}
    >
      <StepIndicator status={status} />
      <span
        className={cn(
          "text-xs font-mono transition-colors duration-200",
          status === "done"    ? "text-[#00ff41]"  :
          status === "error"   ? "text-[#ef4444]"  :
          status === "running" ? "text-[#a8c8a8]"  :
                                 "text-[#4a6048]"
        )}
      >
        {label}
      </span>
    </div>
  );
}

function StepIndicator({ status }: { status: StepStatus }) {
  if (status === "waiting") {
    return (
      <div className="w-5 h-5 rounded-full border border-[#2a3a28] shrink-0 flex items-center justify-center">
        <div className="w-1 h-1 rounded-full bg-[#2a3a28]" />
      </div>
    );
  }

  if (status === "running") {
    return (
      <div className="w-5 h-5 shrink-0 relative flex items-center justify-center">
        <div className="absolute inset-0 rounded-full border border-[#00ff41]/20" />
        <div
          className="absolute inset-0 rounded-full border-2 border-transparent animate-spin"
          style={{ borderTopColor: "#00ff41" }}
        />
      </div>
    );
  }

  if (status === "done") {
    return (
      <div className="w-5 h-5 rounded-full bg-[#00ff41] shrink-0 flex items-center justify-center check-pop">
        <Check className="w-3 h-3 text-black stroke-[3]" />
      </div>
    );
  }

  // error
  return (
    <div className="w-5 h-5 rounded-full border border-[#ef4444] bg-[#ef4444]/10 shrink-0 flex items-center justify-center">
      <X className="w-3 h-3 text-[#ef4444]" />
    </div>
  );
}
