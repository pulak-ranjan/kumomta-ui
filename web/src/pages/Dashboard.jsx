import React, { useEffect, useState } from "react";
import { getStatus } from "../api";

export default function Dashboard() {
  const [status, setStatus] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const s = await getStatus();
        setStatus(s);
      } catch (err) {
        setError(err.message || "Failed to load status");
      }
    })();
  }, []);

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Dashboard</h2>
      {error && <div className="text-xs text-red-400">{error}</div>}
      {status ? (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-sm">
          {["api", "kumomta", "dovecot", "fail2ban"].map((key) => (
            <div
              key={key}
              className="bg-slate-900 border border-slate-800 rounded-lg p-3"
            >
              <div className="text-xs uppercase text-slate-400">{key}</div>
              <div
                className={
                  status[key] === "active" || status[key] === "ok"
                    ? "text-green-400"
                    : "text-red-400"
                }
              >
                {status[key] || "unknown"}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="text-sm text-slate-400">Loading status...</div>
      )}
    </div>
  );
}
