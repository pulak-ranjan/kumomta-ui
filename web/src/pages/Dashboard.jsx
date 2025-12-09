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

  if (!stats) return <div className="p-4">Loading stats...</div>;

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold">Dashboard</h2>
        <button
          onClick={getAI}
          disabled={loading}
          className="bg-purple-600 hover:bg-purple-500 text-white px-3 py-1 rounded-md text-sm flex items-center gap-2"
        >
          {loading ? "Thinking..." : "✨ AI Insights"}
        </button>
      </div>

      {/* Insight Box */}
      {insight && (
        <div className="bg-slate-800 border border-purple-500/50 p-4 rounded-lg text-sm text-slate-200 whitespace-pre-wrap animate-fade-in">
          {insight}
        </div>
      )}

      {/* Main Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Domains" value={stats.domains} />
        <StatCard label="Senders" value={stats.senders} />
        <StatCard label="Total Sent" value="-" note="(Check logs)" />
        <StatCard label="Bounces" value="-" note="(Check logs)" />
      </div>

      {/* Server Health */}
      <h3 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">
        Server Health
      </h3>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatCard
          label="CPU Load (1m)"
          value={stats.cpu_load}
          color="text-sky-400"
        />
        <StatCard
          label="RAM Usage"
          value={stats.ram_usage}
          color="text-sky-400"
        />
        <StatCard
          label="KumoMTA"
          value={stats.kumo_status}
          color={
            stats.kumo_status.includes("Running")
              ? "text-green-400"
              : "text-red-400"
          }
        />
      </div>
    </div>
  );
}

function StatCard({ label, value, note, color = "text-white" }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
      <div className="text-xs uppercase text-slate-500 mb-1">{label}</div>
      <div className={`text-xl font-mono ${color}`}>{value}</div>
      {note && <div className="text-[10px] text-slate-600 mt-1">{note}</div>}
    </div>
  );
}
