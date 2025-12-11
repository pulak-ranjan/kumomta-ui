import React, { useEffect, useState } from "react";
import { 
  Thermometer, Play, Pause, RefreshCw, AlertCircle, Settings, X, Save, Loader2 
} from "lucide-react";
import { getWarmupList, updateWarmup } from "../api";
import { cn } from "../lib/utils";

export default function WarmupPage() {
  const [senders, setSenders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modal State
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState({ enabled: false, plan: "standard" });
  const [saving, setSaving] = useState(false);

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

  // Open the configuration modal
  const openEdit = (s) => {
    setEditing(s);
    setForm({ 
      enabled: s.enabled, 
      plan: s.plan || "standard" 
    });
  };

  // Save changes from modal
  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await updateWarmup(editing.sender_id, form.enabled, form.plan);
      setEditing(null);
      load();
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  // Quick toggle (preserves current plan)
  const quickToggle = async (s) => {
    try {
      await updateWarmup(s.sender_id, !s.enabled, s.plan || "standard");
      load();
    } catch (err) {
      setError(err.message);
    }
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
                  <td className="px-6 py-4 capitalize">
                    {s.enabled ? s.plan : <span className="text-muted-foreground opacity-50">{s.plan || "Standard"}</span>}
                  </td>
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
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => openEdit(s)}
                        className="p-2 rounded-md hover:bg-muted border text-muted-foreground transition-colors"
                        title="Configure Plan"
                      >
                        <Settings className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => quickToggle(s)}
                        className={cn("p-2 rounded-md transition-colors border", 
                          s.enabled ? "hover:bg-red-500/10 border-red-200 text-red-600" : "hover:bg-green-500/10 border-green-200 text-green-600")}
                        title={s.enabled ? "Pause Warmup" : "Start Warmup"}
                      >
                        {s.enabled ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                      </button>
                    </div>
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

      {/* Configuration Modal */}
      {editing && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-lg shadow-lg p-6 space-y-4">
            <div className="flex justify-between items-center border-b pb-4">
              <div>
                <h3 className="text-lg font-semibold">Configure Warmup</h3>
                <p className="text-xs text-muted-foreground">{editing.email}</p>
              </div>
              <button onClick={() => setEditing(null)}><X className="w-4 h-4" /></button>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div className="space-y-3">
                <label className="text-sm font-medium">Warmup Plan</label>
                <div className="grid grid-cols-1 gap-2">
                  {['conservative', 'standard', 'aggressive'].map((plan) => (
                    <label 
                      key={plan}
                      className={cn(
                        "flex items-center justify-between p-3 rounded-md border cursor-pointer transition-all",
                        form.plan === plan ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-muted"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <input 
                          type="radio" 
                          name="plan" 
                          value={plan}
                          checked={form.plan === plan}
                          onChange={e => setForm({...form, plan: e.target.value})}
                          className="hidden"
                        />
                        <span className="capitalize font-medium">{plan}</span>
                      </div>
                      {plan === 'conservative' && <span className="text-xs text-muted-foreground">Starts at 10/hr</span>}
                      {plan === 'standard' && <span className="text-xs text-muted-foreground">Starts at 25/hr</span>}
                      {plan === 'aggressive' && <span className="text-xs text-muted-foreground">Starts at 50/hr</span>}
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex items-center gap-2 p-3 bg-muted/30 rounded-md">
                <input 
                  type="checkbox" 
                  id="enableSwitch"
                  checked={form.enabled}
                  onChange={e => setForm({...form, enabled: e.target.checked})}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                />
                <label htmlFor="enableSwitch" className="text-sm font-medium cursor-pointer">
                  Enable Warmup Schedule
                </label>
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setEditing(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" disabled={saving} className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">
                  {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  Save Configuration
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
