import React, { useState } from "react";
import { previewConfig, applyConfig } from "../api";

export default function ConfigPage() {
  const [preview, setPreview] = useState(null);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  const handlePreview = async () => {
    setBusy(true);
    setMsg("");
    try {
      const data = await previewConfig();
      setPreview(data);
    } catch (err) {
      setMsg(err.message || "Failed to preview config");
    } finally {
      setBusy(false);
    }
  };

  const handleApply = async () => {
    if (!window.confirm("Apply and restart Kumo?")) return;
    setBusy(true);
    setMsg("");
    try {
      const res = await applyConfig();
      setMsg(
        res.error
          ? "Apply failed: " + res.error
          : "Config applied and Kumo restarted."
      );
    } catch (err) {
      setMsg(err.message || "Failed to apply config");
    } finally {
      setBusy(false);
    }
  };

  const codeBox = (title, content) => (
    <div className="space-y-1">
      <div className="text-xs text-slate-400">{title}</div>
      <pre className="bg-slate-950 border border-slate-800 rounded-md p-2 text-[11px] overflow-auto max-h-60 whitespace-pre-wrap">
        {content || "<empty>"}
      </pre>
    </div>
  );

  return (
    <div className="space-y-4 text-sm">
      <h2 className="text-lg font-semibold">Kumo Config</h2>
      <div className="flex gap-2 text-xs">
        <button
          onClick={handlePreview}
          disabled={busy}
          className="px-3 py-1 rounded-md bg-slate-800 hover:bg-slate-700 disabled:opacity-60"
        >
          Preview Config
        </button>
        <button
          onClick={handleApply}
          disabled={busy}
          className="px-3 py-1 rounded-md bg-red-600/80 hover:bg-red-600 disabled:opacity-60"
        >
          Apply & Restart
        </button>
      </div>
      {msg && <div className="text-xs text-slate-300">{msg}</div>}
      {preview && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {codeBox("sources.toml", preview.sources_toml)}
          {codeBox("queues.toml", preview.queues_toml)}
          {codeBox("listener_domains.toml", preview.listener_domains_toml)}
          {codeBox("dkim_data.toml", preview.dkim_data_toml)}
          {codeBox("init.lua", preview.init_lua)}
        </div>
      )}
    </div>
  );
}
