import React, { useEffect, useState } from "react";
import { getLogs } from "../api";

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

  return (
    <div className="space-y-4 text-sm">
      <h2 className="text-lg font-semibold">Logs</h2>
      <div className="flex gap-2 text-xs">
        {["kumomta", "dovecot", "fail2ban"].map((svc) => (
          <button
            key={svc}
            onClick={() => setService(svc)}
            className={`px-3 py-1 rounded-md ${
              service === svc
                ? "bg-sky-500 text-slate-50"
                : "bg-slate-800 text-slate-200 hover:bg-slate-700"
            }`}
          >
            {svc}
          </button>
        ))}
        <button
          onClick={() => load(service)}
          disabled={busy}
          className="ml-auto px-3 py-1 rounded-md bg-slate-800 hover:bg-slate-700 disabled:opacity-60"
        >
          Refresh
        </button>
      </div>
      {msg && <div className="text-xs text-red-400">{msg}</div>}
      <pre className="bg-slate-950 border border-slate-800 rounded-md p-2 text-[11px] overflow-auto max-h-[480px] whitespace-pre-wrap">
        {logs || "<no logs>"}
      </pre>
    </div>
  );
}
