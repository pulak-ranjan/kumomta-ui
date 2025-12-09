import React, { useEffect, useState } from "react";
import { Key, Copy, Check, RefreshCw, Plus, Filter } from "lucide-react";
import { listDKIMRecords, generateDKIM, listDomains } from "../api";
import { cn } from "../lib/utils";

export default function DKIMPage() {
  const [records, setRecords] = useState([]);
  const [domains, setDomains] = useState([]);
  const [selectedDomain, setSelectedDomain] = useState("");
  const [msg, setMsg] = useState("");
  const [form, setForm] = useState({ domain: "", local_part: "" });
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      const [recs, doms] = await Promise.all([listDKIMRecords(), listDomains()]);
      setRecords(Array.isArray(recs) ? recs : []);
      setDomains(Array.isArray(doms) ? doms : []);
    } catch (err) {
      setMsg(err.message || "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const onGenerate = async (e) => {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      await generateDKIM(form.domain, form.local_part || undefined);
      setMsg("DKIM keys generated. Refreshing records...");
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to generate DKIM");
    } finally {
      setBusy(false);
    }
  };

  const [copied, setCopied] = useState("");
  const copy = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(""), 2000);
  };

  const filteredRecords = selectedDomain ? records.filter(r => r.domain === selectedDomain) : records;

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">DKIM Management</h2>
          <p className="text-muted-foreground">Manage DomainKeys Identified Mail signatures.</p>
        </div>
      </div>

      {msg && <div className="bg-primary/10 text-primary p-3 rounded-md text-sm">{msg}</div>}

      <div className="grid lg:grid-cols-3 gap-6">
        {/* Generator Panel */}
        <div className="lg:col-span-1">
          <div className="bg-card border rounded-xl p-5 shadow-sm sticky top-6">
            <h3 className="font-semibold mb-4 flex items-center gap-2">
              <Plus className="w-4 h-4" /> Generate Keys
            </h3>
            <form onSubmit={onGenerate} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Domain</label>
                <input 
                  value={form.domain} 
                  onChange={e => setForm({...form, domain: e.target.value})} 
                  className="w-full h-10 px-3 rounded-md border bg-background text-sm"
                  placeholder="example.com"
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Local Part (Selector)</label>
                <input 
                  value={form.local_part} 
                  onChange={e => setForm({...form, local_part: e.target.value})} 
                  className="w-full h-10 px-3 rounded-md border bg-background text-sm"
                  placeholder="default (optional)"
                />
                <p className="text-[10px] text-muted-foreground">Leave blank to generate for all existing senders.</p>
              </div>
              <button 
                type="submit" 
                disabled={busy} 
                className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
              >
                {busy ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
                Generate Key
              </button>
            </form>
          </div>
        </div>

        {/* List Panel */}
        <div className="lg:col-span-2 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="font-semibold flex items-center gap-2">
              <Key className="w-4 h-4 text-muted-foreground" /> 
              Active Keys ({filteredRecords.length})
            </h3>
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-muted-foreground" />
              <select 
                className="h-8 rounded-md border bg-background text-xs px-2"
                value={selectedDomain}
                onChange={e => setSelectedDomain(e.target.value)}
              >
                <option value="">All Domains</option>
                {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
              </select>
            </div>
          </div>

          {loading ? (
            <div className="text-center py-10 text-muted-foreground">Loading records...</div>
          ) : filteredRecords.length === 0 ? (
            <div className="text-center py-10 border border-dashed rounded-xl">
              <p className="text-muted-foreground">No DKIM records found.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {filteredRecords.map((r, i) => {
                const fullLine = `${r.dns_name} 3600 IN TXT "${r.dns_value}"`;
                return (
                  <div key={i} className="bg-card border rounded-lg p-4 shadow-sm hover:shadow-md transition-all">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <div className="font-medium text-foreground">{r.domain}</div>
                        <div className="text-xs text-muted-foreground font-mono">selector: {r.selector}</div>
                      </div>
                      <div className="flex gap-1">
                        <button onClick={() => copy(r.dns_name, `name-${i}`)} className="p-1.5 hover:bg-muted rounded text-muted-foreground hover:text-foreground" title="Copy Name">
                          {copied === `name-${i}` ? <Check className="w-3 h-3 text-green-500" /> : <span className="text-[10px] font-bold">Name</span>}
                        </button>
                        <button onClick={() => copy(r.dns_value, `val-${i}`)} className="p-1.5 hover:bg-muted rounded text-muted-foreground hover:text-foreground" title="Copy Value">
                          {copied === `val-${i}` ? <Check className="w-3 h-3 text-green-500" /> : <span className="text-[10px] font-bold">Val</span>}
                        </button>
                        <button onClick={() => copy(fullLine, `full-${i}`)} className="p-1.5 hover:bg-muted rounded text-muted-foreground hover:text-foreground" title="Copy Full Record">
                          {copied === `full-${i}` ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}
                        </button>
                      </div>
                    </div>
                    
                    <div className="bg-muted/50 p-2 rounded border font-mono text-[10px] text-muted-foreground break-all">
                      {r.dns_value}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
