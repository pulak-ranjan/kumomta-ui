import React, { useState } from "react";
import { 
  ServerCog, 
  Play, 
  Eye, 
  FileCode, 
  Save, 
  AlertTriangle 
} from "lucide-react";
import { previewConfig, applyConfig } from "../api";
import { cn } from "../lib/utils";

export default function ConfigPage() {
  const [preview, setPreview] = useState(null);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);

  const handlePreview = async () => {
    setBusy(true);
    setMsg("");
    setError(false);
    try {
      const data = await previewConfig();
      setPreview(data);
    } catch (err) {
      setMsg(err.message || "Failed to preview config");
      setError(true);
    } finally {
      setBusy(false);
    }
  };

  const handleApply = async () => {
    if (!window.confirm("This will overwrite configuration files and restart the KumoMTA service. Continue?")) return;
    setBusy(true);
    setMsg("");
    setError(false);
    try {
      const res = await applyConfig();
      if (res.error) {
        setMsg("Apply Failed: " + res.error);
        setError(true);
      } else {
        setMsg("Configuration applied successfully. Service restarted.");
      }
    } catch (err) {
      setMsg(err.message || "Failed to apply config");
      setError(true);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Configuration</h1>
          <p className="text-muted-foreground">Generate and apply KumoMTA policy files.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handlePreview}
            disabled={busy}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors"
          >
            <Eye className="w-4 h-4" /> Preview
          </button>
          <button
            onClick={handleApply}
            disabled={busy}
            className="flex items-center gap-2 h-10 px-4 rounded-md bg-destructive text-destructive-foreground hover:bg-destructive/90 text-sm font-medium transition-colors shadow-sm"
          >
            <Play className="w-4 h-4" /> Apply & Restart
          </button>
        </div>
      </div>

      {msg && (
        <div className={cn(
          "flex items-center gap-2 p-4 rounded-lg text-sm font-medium animate-in fade-in slide-in-from-top-2",
          error ? "bg-destructive/10 text-destructive border border-destructive/20" : "bg-green-500/10 text-green-600 border border-green-500/20"
        )}>
          {error ? <AlertTriangle className="w-4 h-4" /> : <Save className="w-4 h-4" />}
          {msg}
        </div>
      )}

      {!preview ? (
        <div className="flex flex-col items-center justify-center p-16 border-2 border-dashed rounded-xl bg-muted/10">
          <ServerCog className="w-16 h-16 text-muted-foreground/30 mb-4" />
          <h3 className="text-lg font-semibold text-foreground">No Configuration Loaded</h3>
          <p className="text-muted-foreground text-sm mt-1">Click "Preview" to generate the current configuration snapshot.</p>
        </div>
      ) : (
        <div className="grid xl:grid-cols-2 gap-6">
          <ConfigBox title="init.lua" content={preview.init_lua} />
          <ConfigBox title="sources.toml" content={preview.sources_toml} />
          <ConfigBox title="queues.toml" content={preview.queues_toml} />
          <ConfigBox title="listener_domains.toml" content={preview.listener_domains_toml} />
          <ConfigBox title="dkim_data.toml" content={preview.dkim_data_toml} />
        </div>
      )}
    </div>
  );
}

function ConfigBox({ title, content }) {
  return (
    <div className="bg-zinc-950 border border-zinc-800 rounded-xl overflow-hidden shadow-sm flex flex-col">
      <div className="bg-zinc-900/50 border-b border-zinc-800 p-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FileCode className="w-4 h-4 text-blue-400" />
          <span className="text-sm font-medium text-zinc-300">{title}</span>
        </div>
        <span className="text-[10px] text-zinc-500 font-mono">LUA/TOML</span>
      </div>
      <div className="relative group flex-1">
        <pre className="p-4 text-xs font-mono text-zinc-300 overflow-auto whitespace-pre-wrap max-h-[300px] custom-scrollbar">
          {content || <span className="text-zinc-600 italic"># Empty file</span>}
        </pre>
      </div>
    </div>
  );
}
