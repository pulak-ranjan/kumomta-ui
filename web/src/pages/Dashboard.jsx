import React, { useEffect, useState } from "react";
import { getDashboardStats, getAIInsights } from "../api";

export default function Dashboard() {
  const [stats, setStats] = useState(null);
  const [insight, setInsight] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const s = await getDashboardStats();
        setStats(s);
      } catch (err) {
        setError("Failed to load stats");
      }
    })();
  }, []);

  const getAI = async () => {
    setLoading(true);
    setInsight("");
    try {
      const res = await getAIInsights();
      setInsight(res.insight);
    } catch (err) {
      setInsight("Error: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  if (!stats) return <div className="p-4 text-slate-400">Loading dashboard...</div>;

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold">System Dashboard</h2>
        <button 
          onClick={getAI}
          disabled={loading}
          className="bg-purple-600 hover:bg-purple-500 text-white px-3 py-1.5 rounded-md text-xs font-medium flex items-center gap-2 transition-colors"
        >
          {loading ? "Analyzing Logs..." : "✨ AI Log Analysis"}
        </button>
      </div>

      {/* AI Insight Box */}
      {insight && (
        <div className="bg-slate-900/80 border border-purple-500/30 p-4 rounded-lg text-sm text-slate-300 whitespace-pre-wrap shadow-lg shadow-purple-900/10">
          <div className="text-purple-400 font-semibold mb-2 text-xs uppercase tracking-wider">AI Analysis Result</div>
          {insight}
        </div>
      )}

      {/* Main Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Domains" value={stats.domains} icon="🌐" />
        <StatCard label="Senders" value={stats.senders} icon="📧" />
        <StatCard label="CPU Load" value={stats.cpu_load} color="text-sky-400" />
        <StatCard label="RAM Usage" value={stats.ram_usage} color="text-sky-400" />
      </div>

      {/* Infrastructure Health */}
      <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider mt-6 mb-3">Service Status</h3>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <ServiceCard name="KumoMTA" status={stats.kumo_status} />
        <ServiceCard name="Dovecot" status={stats.dovecot_status} />
        <ServiceCard name="Fail2Ban" status={stats.f2b_status} />
      </div>

      {/* Open Ports */}
      <div className="bg-slate-900 border border-slate-800 rounded-lg p-4 mt-4">
        <div className="text-xs uppercase text-slate-500 mb-2">Open Ports (Public/Local)</div>
        <div className="flex flex-wrap gap-2">
          {stats.open_ports ? (
            stats.open_ports.split(", ").map(port => (
              <span key={port} className="bg-slate-950 border border-slate-700 text-slate-300 px-2 py-1 rounded text-xs font-mono">
                {port}
              </span>
            ))
          ) : (
            <span className="text-slate-600 text-xs">Scanning...</span>
          )}
        </div>
      </div>
    </div>
  );
}

function StatCard({ label, value, color = "text-white", icon }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg p-4 relative overflow-hidden">
      <div className="relative z-10">
        <div className="text-xs uppercase text-slate-500 mb-1">{label}</div>
        <div className={`text-2xl font-mono ${color}`}>{value}</div>
      </div>
      {icon && <div className="absolute -right-2 -bottom-2 text-4xl opacity-10 grayscale">{icon}</div>}
    </div>
  );
}

function ServiceCard({ name, status }) {
  const isActive = status === "active";
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg p-4 flex items-center justify-between">
      <div>
        <div className="text-xs uppercase text-slate-500 mb-1">{name}</div>
        <div className={`text-lg font-medium ${isActive ? "text-green-400" : "text-red-400"}`}>
          {status || "Unknown"}
        </div>
      </div>
      <div className={`w-3 h-3 rounded-full ${isActive ? "bg-green-500 animate-pulse" : "bg-red-500"}`}></div>
    </div>
  );
}
