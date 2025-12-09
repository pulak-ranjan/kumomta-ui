import React, { useEffect, useState } from "react";
import { MailWarning, Filter, Trash2, User, Globe, AlertCircle } from "lucide-react";
import { listBounces, listDomains } from "../api";
import { cn } from "../lib/utils";

export default function BouncePage() {
  const [list, setList] = useState([]);
  const [domains, setDomains] = useState([]);
  const [filter, setFilter] = useState("");
  const [msg, setMsg] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true);
    try {
      const [b, d] = await Promise.all([listBounces(), listDomains()]);
      setList(Array.isArray(b) ? b : []);
      setDomains(Array.isArray(d) ? d : []);
    } catch (err) {
      setMsg("Failed to load data");
      setList([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredList = filter ? list.filter(b => b.domain === filter) : list;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Bounce Accounts</h1>
          <p className="text-muted-foreground">System users handling incoming bounce messages.</p>
        </div>
        
        <div className="relative">
          <Filter className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <select 
            className="h-10 rounded-md border bg-background pl-9 pr-8 py-2 text-sm focus:ring-2 focus:ring-ring min-w-[200px]"
            value={filter}
            onChange={e => setFilter(e.target.value)}
          >
            <option value="">All Domains</option>
            {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
          </select>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-muted-foreground">Loading accounts...</div>
      ) : filteredList.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-12 bg-card border rounded-xl border-dashed">
          <MailWarning className="w-12 h-12 text-muted-foreground/50 mb-3" />
          <h3 className="text-lg font-medium">No Bounce Accounts</h3>
          <p className="text-muted-foreground text-sm">Bounce accounts are created automatically when you add senders.</p>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredList.map(b => (
            <div key={b.id} className="group bg-card border rounded-xl p-4 shadow-sm hover:shadow-md transition-all relative">
              <div className="flex justify-between items-start mb-2">
                <div className="p-2 bg-purple-500/10 text-purple-500 rounded-lg">
                  <User className="w-5 h-5" />
                </div>
                <button className="text-muted-foreground hover:text-destructive transition-colors opacity-0 group-hover:opacity-100">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
              
              <div className="space-y-1">
                <div className="font-mono font-medium text-lg">{b.username}</div>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Globe className="w-3 h-3" />
                  {b.domain}
                </div>
              </div>

              {b.notes && (
                <div className="mt-3 pt-3 border-t text-xs text-muted-foreground flex items-start gap-2">
                  <AlertCircle className="w-3 h-3 mt-0.5 shrink-0" />
                  <span className="line-clamp-2">{b.notes}</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
