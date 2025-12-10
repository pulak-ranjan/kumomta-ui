import React, { useEffect, useState } from "react";
import { 
  Thermometer, Play, Pause, RefreshCw, AlertCircle 
} from "lucide-react";
import { getWarmupList, updateWarmup } from "../api";
import { cn } from "../lib/utils";

export default function WarmupPage() {
  const [senders, setSenders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = async () => {
    setLoading(true);
    try {
      const data = await getWarmupList();
      setSenders(data || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const toggle = async (id, currentStatus) => {
    // If turning on, default to 'standard', else keep existing logic
    await updateWarmup(id, !currentStatus, "standard");
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">IP Warmup</h1>
          <p className="text-muted-foreground">Automated daily rate limiting for new IPs.</p>
        </div>
        <button onClick={load} className="p-2 hover:bg-muted rounded-md border">
          <RefreshCw className={cn("w-5 h-5", loading && "animate-spin")} />
        </button>
      </div>

      {error && (
        <div className="bg-destructive/10 text-destructive p-4 rounded-md flex items-center gap-2">
          <AlertCircle className="w-5 h-5" /> {error}
        </div>
      )}

      <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
              <tr>
                <th className="px-6 py-3">Sender Identity</th>
                <th className="px-6 py-3">Status</th>
                <th className="px-6 py-3">Plan</th>
                <th className="px-6 py-3">Progress</th>
                <th className="px-6 py-3">Current Limit</th>
                <th className="px-6 py-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {senders.map((s) => (
                <tr key={s.sender_id} className="hover:bg-muted/50 transition-colors">
                  <td className="px-6 py-4">
                    <div className="font-medium">{s.email}</div>
                    <div className="text-xs text-muted-foreground">{s.domain}</div>
                  </td>
                  <td className="px-6 py-4">
                    <span className={cn("px-2 py-1 rounded-full text-xs font-bold flex w-fit items-center gap-1", 
                      s.enabled ? "bg-orange-500/10 text-orange-600" : "bg-muted text-muted-foreground")}>
                      {s.enabled ? <Thermometer className="w-3 h-3" /> : null}
                      {s.enabled ? "WARMING" : "OFF"}
                    </span>
                  </td>
                  <td className="px-6 py-4 capitalize">{s.enabled ? s.plan : "-"}</td>
                  <td className="px-6 py-4">
                    {s.enabled ? (
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold">Day {s.day}</span>
                      </div>
                    ) : "-"}
                  </td>
                  <td className="px-6 py-4">
                    <span className="font-mono bg-background border px-2 py-1 rounded">
                      {s.current_rate || "Unlimited"}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    <button
                      onClick={() => toggle(s.sender_id, s.enabled)}
                      className={cn("p-2 rounded-md transition-colors border", 
                        s.enabled ? "hover:bg-red-500/10 border-red-200 text-red-600" : "hover:bg-green-500/10 border-green-200 text-green-600")}
                      title={s.enabled ? "Pause Warmup" : "Start Warmup"}
                    >
                      {s.enabled ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                    </button>
                  </td>
                </tr>
              ))}
              {senders.length === 0 && !loading && (
                <tr>
                  <td colSpan="6" className="text-center py-8 text-muted-foreground">
                    No senders found. Add senders in Domains first.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
