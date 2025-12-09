import React, { useEffect, useState } from "react";
import { 
  Network, 
  Plus, 
  Trash2, 
  Globe, 
  Server, 
  RefreshCw,
  ListPlus
} from "lucide-react";
import { getSystemIPs, addSystemIPs } from "../api";
import { cn } from "../lib/utils";

export default function IPsPage() {
  const [ips, setIPs] = useState([]);
  const [form, setForm] = useState({ cidr: "", list: "" });
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      const list = await getSystemIPs();
      setIPs(list || []);
    } catch (err) {
      setMsg("Failed to load IPs");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const onAdd = async (e) => {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      const res = await addSystemIPs(form.cidr, form.list);
      setMsg(`Successfully added ${res.added || 0} IPs.`);
      setForm({ cidr: "", list: "" });
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to add IPs");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">IP Inventory</h1>
          <p className="text-muted-foreground">Manage server IP addresses for rotation.</p>
        </div>
        <button onClick={load} className="p-2 hover:bg-muted rounded-md transition-colors">
          <RefreshCw className={cn("w-5 h-5 text-muted-foreground", loading && "animate-spin")} />
        </button>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        {/* Add IPs Form */}
        <div className="lg:col-span-1">
          <div className="bg-card border rounded-xl p-5 shadow-sm sticky top-6">
            <h3 className="font-semibold mb-4 flex items-center gap-2">
              <ListPlus className="w-4 h-4" /> Add IPs
            </h3>
            <form onSubmit={onAdd} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">CIDR Range</label>
                <div className="relative">
                  <Globe className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input
                    value={form.cidr}
                    onChange={(e) => setForm({ ...form, cidr: e.target.value })}
                    placeholder="192.168.1.0/24"
                    className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  />
                </div>
              </div>
              
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">Or</span>
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Bulk List (One per line)</label>
                <textarea
                  value={form.list}
                  onChange={(e) => setForm({ ...form, list: e.target.value })}
                  rows={6}
                  placeholder={"10.0.0.1\n10.0.0.2"}
                  className="w-full rounded-md border bg-background p-3 text-sm font-mono focus:ring-2 focus:ring-ring resize-none"
                />
              </div>

              {msg && (
                <div className={cn("p-3 rounded-md text-sm", msg.includes("Failed") ? "bg-destructive/10 text-destructive" : "bg-green-500/10 text-green-600")}>
                  {msg}
                </div>
              )}

              <button
                type="submit"
                disabled={busy}
                className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
              >
                {busy ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                Add to Inventory
              </button>
            </form>
          </div>
        </div>

        {/* List IPs */}
        <div className="lg:col-span-2">
          <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
            <div className="p-4 border-b bg-muted/30 flex justify-between items-center">
              <h3 className="font-semibold text-sm flex items-center gap-2">
                <Server className="w-4 h-4 text-muted-foreground" />
                Available IPs <span className="bg-primary/10 text-primary px-2 py-0.5 rounded-full text-xs">{ips.length}</span>
              </h3>
            </div>
            
            <div className="max-h-[600px] overflow-y-auto">
              {loading ? (
                <div className="p-8 text-center text-muted-foreground">Loading inventory...</div>
              ) : ips.length === 0 ? (
                <div className="p-12 text-center border-dashed">
                  <Network className="w-12 h-12 mx-auto text-muted-foreground/50 mb-3" />
                  <p className="text-muted-foreground">No IPs found in inventory.</p>
                </div>
              ) : (
                <div className="divide-y">
                  {ips.map((ip, i) => (
                    <div key={i} className="flex items-center justify-between p-3 hover:bg-muted/50 transition-colors group">
                      <div className="flex items-center gap-3">
                        <div className="p-2 rounded-md bg-secondary text-secondary-foreground">
                          <Network className="w-4 h-4" />
                        </div>
                        <div>
                          <div className="font-mono text-sm font-medium">{ip.value}</div>
                          <div className="text-xs text-muted-foreground flex gap-2">
                            <span>{ip.interface || "eth0"}</span>
                            <span>•</span>
                            <span>{ip.netmask || "/32"}</span>
                          </div>
                        </div>
                      </div>
                      <button className="p-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md opacity-0 group-hover:opacity-100 transition-all">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
