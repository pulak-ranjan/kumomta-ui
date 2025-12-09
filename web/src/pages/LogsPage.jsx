import React, { useEffect, useState } from "react";
import { FileText, RefreshCw, Terminal, Shield, Mail } from "lucide-react";
import { getLogs } from "../api";
import { cn } from "../lib/utils";

export default function LogsPage() {
  const [service, setService] = useState("kumomta");
  const [logs, setLogs] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  const load = async (svc) => {
    setBusy(true);
    setMsg("");
    try {
      const res = await getLogs(svc, 100);
      setLogs(res.logs || "");
    } catch (err) {
      setMsg(err.message || "Failed to load logs");
      setLogs("");
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    load(service);
  }, [service]);

  const services = [
    { id: 'kumomta', label: 'KumoMTA', icon: Mail },
    { id: 'dovecot', label: 'Dovecot', icon: FileText },
    { id: 'fail2ban', label: 'Fail2Ban', icon: Shield },
  ];

  return (
    <div className="space-y-6 h-[calc(100vh-140px)] flex flex-col">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 flex-shrink-0">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">System Logs</h1>
          <p className="text-muted-foreground">View real-time journalctl output.</p>
        </div>
        <button 
          onClick={() => load(service)}
          disabled={busy}
          className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors"
        >
          <RefreshCw className={cn("w-4 h-4", busy && "animate-spin")} />
          Refresh
        </button>
      </div>

      <div className="flex items-center space-x-1 bg-muted p-1 rounded-lg w-fit flex-shrink-0">
        {services.map((svc) => {
          const Icon = svc.icon;
          const isActive = service === svc.id;
          return (
            <button
              key={svc.id}
              onClick={() => setService(svc.id)}
              className={cn(
                "flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-all",
                isActive 
                  ? "bg-background text-foreground shadow-sm" 
                  : "text-muted-foreground hover:text-foreground hover:bg-background/50"
              )}
            >
              <Icon className="w-4 h-4" />
              {svc.label}
            </button>
          );
        })}
      </div>

      {msg && (
        <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm flex-shrink-0">
          {msg}
        </div>
      )}

      <div className="flex-1 bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden shadow-sm flex flex-col min-h-0">
        <div className="bg-zinc-900/50 border-b border-zinc-800 p-3 flex items-center gap-2">
          <Terminal className="w-4 h-4 text-zinc-400" />
          <span className="text-xs font-mono text-zinc-400">journalctl -u {service} -n 100</span>
        </div>
        <pre className="flex-1 p-4 text-xs font-mono text-zinc-300 overflow-auto whitespace-pre-wrap leading-relaxed custom-scrollbar">
          {logs || <span className="text-zinc-600 italic">No logs available or service not running...</span>}
        </pre>
      </div>
    </div>
  );
}
