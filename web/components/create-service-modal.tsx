"use client";

import { useState } from "react";
import { X, Check } from "lucide-react";
import { api, type CreateServiceInput } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  onClose: () => void;
  onCreated: (serviceId: string) => void;
}

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

type StepStatus = "waiting" | "running" | "done" | "error";

// Maps the server's error message to which step it represents.
// Steps: 0=ssh, 1=docker, 2=nginx-present, 3=nginx-bootstrap, 4=ports, 5=save
function stepForError(msg: string): number {
  const m = msg.toLowerCase();
  if (m.includes("docker"))                                           return 1;
  if (m.includes("nginx is not installed"))                          return 2;
  if (m.includes("bootstrap") || m.includes("failed to bootstrap")) return 3;
  if (m.includes("port"))                                            return 4;
  if (m.includes("persist") || m.includes("id conflict"))           return 5;
  // SSH failures, invalid input, domain taken — all fail before any step confirms
  return 0;
}

/* ── small helpers ──────────────────────────────────────── */

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <span className="text-[11px] text-[#3f3f46] font-mono uppercase tracking-widest whitespace-nowrap">
          {title}
        </span>
        <div className="flex-1 h-px bg-[#1c1c1f]" />
      </div>
      {children}
    </div>
  );
}

function Field({
  label,
  placeholder,
  type = "text",
  value,
  onChange,
  span2 = false,
}: {
  label: string;
  placeholder: string;
  type?: string;
  value: string;
  onChange: (v: string) => void;
  span2?: boolean;
}) {
  return (
    <div className={cn(span2 && "col-span-2")}>
      <label className="block text-[11px] font-mono text-[#52525b] uppercase tracking-wider mb-1.5">
        {label}
      </label>
      <div className="flex items-center border border-[#27272a] rounded-md bg-[#0d0d10] focus-within:border-[#6366f1]/50 transition-colors">
        <span className="pl-3 pr-2 text-[#3f3f46] text-sm font-mono select-none">›</span>
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="flex-1 bg-transparent text-sm font-mono text-[#e4e4e7] py-2.5 pr-3 placeholder:text-[#3f3f46] focus:outline-none min-w-0"
        />
      </div>
    </div>
  );
}

function KeyField({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div className="col-span-2">
      <label className="block text-[11px] font-mono text-[#52525b] uppercase tracking-wider mb-1.5">
        ssh_private_key
      </label>
      <div className="border border-[#27272a] rounded-md bg-[#0d0d10] focus-within:border-[#6366f1]/50 transition-colors">
        <textarea
          rows={3}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={"-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----"}
          className="w-full bg-transparent text-sm font-mono text-[#e4e4e7] p-3 placeholder:text-[#3f3f46] focus:outline-none resize-none leading-relaxed"
        />
      </div>
    </div>
  );
}

/* ── main component ─────────────────────────────────────── */

export function CreateServiceModal({ onClose, onCreated }: Props) {
  const [values, setValues] = useState<Record<keyof CreateServiceInput, string>>(EMPTY);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stepStatuses, setStepStatuses] = useState<StepStatus[]>(
    STEPS.map(() => "waiting")
  );

  function set(key: keyof CreateServiceInput) {
    return (v: string) => setValues((prev) => ({ ...prev, [key]: v }));
  }

  async function handleSubmit(e: React.SyntheticEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    // Step 0 is "running" while the request is in flight — nothing else advances.
    setStepStatuses(STEPS.map((_, i) => (i === 0 ? "running" : "waiting")));
    setLoading(true);

    try {
      const svc = await api.createService({
        name:             values.name.trim(),
        repo_url:         values.repo_url.trim(),
        domain:           values.domain.trim(),
        health_check_url: values.health_check_url.trim(),
        webhook_secret:   values.webhook_secret.trim(),
        host:             values.host.trim(),
        ssh_user:         values.ssh_user.trim(),
        ssh_private_key:  values.ssh_private_key.trim(),
        blue_port:        Number(values.blue_port),
        green_port:       Number(values.green_port),
        container_port:   Number(values.container_port),
        compose_service:  values.compose_service.trim() || "app",
      });

      // Cascade all steps to "done" only on real success.
      for (let i = 0; i < STEPS.length; i++) {
        setTimeout(() => {
          setStepStatuses((prev) => { const next = [...prev]; next[i] = "done"; return next; });
        }, i * 150);
      }
      setTimeout(() => onCreated(svc.id), STEPS.length * 150 + 200);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "registration failed";
      const failed = stepForError(msg);
      // Mark every step before the failed one as done, the failed one as error.
      setStepStatuses(STEPS.map((_, i) => (i < failed ? "done" : i === failed ? "error" : "waiting")));
      setError(msg);
      // stay in progress view — the error is shown there with a "try again" button
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6">
      <div
        className="absolute inset-0 bg-black/75 backdrop-blur-sm"
        onClick={loading ? undefined : onClose}
      />

      <div className="relative w-full max-w-2xl bg-[#09090b] border border-[#27272a] rounded-lg shadow-2xl flex flex-col max-h-[90vh]">

        {/* Title bar */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#1c1c1f] bg-[#0d0d10] shrink-0 rounded-t-lg">
          <div className="flex items-center gap-3">
            <span className="text-[#a1a1aa] text-base font-mono">register service</span>
          </div>
          {!loading && (
            <button
              onClick={onClose}
              className="p-2 rounded-md text-[#52525b] hover:text-[#e4e4e7] hover:bg-[#27272a] transition-colors cursor-pointer"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>

        {loading ? (
          /* ── progress view ─────────────────────────────── */
          <div className="px-8 py-8 flex-1">
            <p className="text-sm text-[#52525b] font-mono mb-6">
              connecting to your server...
            </p>
            <div className="space-y-5">
              {STEPS.map((label, i) => (
                <StepRow key={label} label={label} status={stepStatuses[i]} index={i} />
              ))}
            </div>

            {error && (
              <>
                <div className="mt-7 px-4 py-3 text-sm font-mono text-[#ef4444] border border-[#ef4444]/20 bg-[#ef4444]/5 rounded-md">
                  error: {error}
                </div>
                <div className="flex justify-end mt-4">
                  <button
                    onClick={() => { setLoading(false); setError(null); setStepStatuses(STEPS.map(() => "waiting")); }}
                    className="px-4 py-2 text-sm font-mono text-[#71717a] hover:text-[#a1a1aa] transition-colors cursor-pointer"
                  >
                    ← try again
                  </button>
                </div>
              </>
            )}
          </div>
        ) : (
          /* ── form view ──────────────────────────────────── */
          <form
            onSubmit={handleSubmit}
            className="flex-1 overflow-y-auto no-scrollbar px-6 py-6 space-y-6"
          >
            {/* Application */}
            <Section title="Application">
              <div className="grid grid-cols-2 gap-4">
                <Field label="name"             placeholder="my-api"                          value={values.name}             onChange={set("name")} />
                <Field label="repo_url"         placeholder="https://github.com/org/repo.git" value={values.repo_url}         onChange={set("repo_url")} />
                <Field label="domain"           placeholder="api.example.com"                 value={values.domain}           onChange={set("domain")} />
                <Field label="health_check_url" placeholder="http://api.example.com/health"   value={values.health_check_url} onChange={set("health_check_url")} />
              </div>
            </Section>

            {/* Server */}
            <Section title="Server">
              <div className="grid grid-cols-2 gap-4">
                <Field label="host"     placeholder="1.2.3.4 or hostname" value={values.host}     onChange={set("host")} />
                <Field label="ssh_user" placeholder="ubuntu"              value={values.ssh_user} onChange={set("ssh_user")} />
                <KeyField value={values.ssh_private_key} onChange={set("ssh_private_key")} />
              </div>
            </Section>

            {/* Ports */}
            <Section title="Ports">
              <div className="grid grid-cols-3 gap-4">
                <Field label="blue_port"      placeholder="3001" type="number" value={values.blue_port}      onChange={set("blue_port")} />
                <Field label="green_port"     placeholder="3002" type="number" value={values.green_port}     onChange={set("green_port")} />
                <Field label="container_port" placeholder="8080" type="number" value={values.container_port} onChange={set("container_port")} />
              </div>
            </Section>

            {/* Config */}
            <Section title="Config">
              <div className="grid grid-cols-2 gap-4">
                <Field label="webhook_secret"  placeholder="super-secret" type="password" value={values.webhook_secret}  onChange={set("webhook_secret")} span2={false} />
                <Field label="compose_service" placeholder="app"                          value={values.compose_service} onChange={set("compose_service")} span2={false} />
              </div>
              <p className="text-xs text-[#3f3f46] font-mono mt-1 leading-relaxed">
                <span className="text-[#52525b]">compose_service</span> is the service name in your docker-compose.yml that launchpad pins the port on
              </p>
            </Section>

            {/* Actions */}
            <div className="flex justify-end gap-3 pt-1 pb-1">
              <button
                type="button"
                onClick={onClose}
                className="px-5 py-2.5 text-sm font-mono rounded-md border border-[#27272a] bg-[#111113] text-[#71717a] hover:text-[#e4e4e7] hover:border-[#3f3f46] transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-5 py-2.5 text-sm font-mono rounded-md border border-[#6366f1]/50 bg-[#6366f1]/10 text-[#6366f1] hover:bg-[#6366f1]/20 hover:border-[#6366f1]/70 transition-colors cursor-pointer"
              >
                Register
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

/* ── step row + indicator ───────────────────────────────── */

function StepRow({ label, status, index }: { label: string; status: StepStatus; index: number }) {
  const visible = status !== "waiting";
  return (
    <div
      className={cn(
        "flex items-center gap-4 transition-all duration-200",
        visible ? "step-slide-in opacity-100" : "opacity-20"
      )}
      style={{ animationDelay: visible ? `${index * 40}ms` : undefined }}
    >
      <StepIndicator status={status} />
      <span className={cn(
        "text-sm font-mono transition-colors duration-200",
        status === "done"    ? "text-[#6366f1]"  :
        status === "error"   ? "text-[#ef4444]"  :
        status === "running" ? "text-[#e4e4e7]"  :
                               "text-[#52525b]"
      )}>
        {label}
      </span>
    </div>
  );
}

function StepIndicator({ status }: { status: StepStatus }) {
  if (status === "waiting") return (
    <div className="w-6 h-6 rounded-full border border-[#27272a] shrink-0 flex items-center justify-center">
      <div className="w-1.5 h-1.5 rounded-full bg-[#27272a]" />
    </div>
  );

  if (status === "running") return (
    <div className="w-6 h-6 shrink-0 relative flex items-center justify-center">
      <div className="absolute inset-0 rounded-full border border-[#6366f1]/20" />
      <div className="absolute inset-0 rounded-full border-2 border-transparent animate-spin" style={{ borderTopColor: "#6366f1" }} />
    </div>
  );

  if (status === "done") return (
    <div className="w-6 h-6 rounded-full bg-[#6366f1] shrink-0 flex items-center justify-center check-pop">
      <Check className="w-3.5 h-3.5 text-white stroke-[2.5]" />
    </div>
  );

  return (
    <div className="w-6 h-6 rounded-full border border-[#ef4444] bg-[#ef4444]/10 shrink-0 flex items-center justify-center">
      <X className="w-3.5 h-3.5 text-[#ef4444]" />
    </div>
  );
}
