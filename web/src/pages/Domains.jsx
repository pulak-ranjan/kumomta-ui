import React, { useEffect, useState } from "react";
import { 
  Plus, 
  Upload, 
  Trash2, 
  Edit2, 
  Copy, 
  Check, 
  Server, 
  Globe, 
  Mail, 
  ShieldCheck, 
  AlertCircle,
  MoreHorizontal
} from "lucide-react";
import {
  listDomains,
  saveDomain,
  deleteDomain,
  saveSender,
  deleteSender,
  getSettings,
  getSystemIPs,
  importSenders
} from "../api";
import { cn } from "../lib/utils";

export default function Domains() {
  const [domains, setDomains] = useState([]);
  const [settings, setSettings] = useState(null);
  const [systemIPs, setSystemIPs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState("");
  const [editingDomain, setEditingDomain] = useState(null);
  const [showImport, setShowImport] = useState(false);
  const [senderForm, setSenderForm] = useState({
    domainID: null,
    id: 0,
    local_part: "",
    email: "",
    ip: "",
    smtp_password: ""
  });

  const load = async () => {
    setLoading(true);
    setMsg("");
    try {
      const [d, s, ips] = await Promise.all([
        listDomains(),
        getSettings(),
        getSystemIPs()
      ]);
      setDomains(Array.isArray(d) ? d : []);
      setSettings(s || null);
      setSystemIPs(Array.isArray(ips) ? ips : []);
    } catch (err) {
      setMsg(err.message || "Failed to load data");
      setDomains([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  // --- Actions ---
  const handleSaveDomain = async (e) => {
    e.preventDefault();
    try {
      await saveDomain(editingDomain);
      setEditingDomain(null);
      await load();
    } catch (err) { setMsg(err.message); }
  };

  const handleDeleteDomain = async (id) => {
    if (!confirm("Delete domain and all senders?")) return;
    try { await deleteDomain(id); await load(); } catch (err) { setMsg(err.message); }
  };

  const handleImport = async (e) => {
    e.preventDefault();
    const file = e.target.file.files[0];
    if (!file) return;
    setLoading(true);
    try {
      const res = await importSenders(file);
      setMsg(res.message);
      setShowImport(false);
      await load();
    } catch (err) { setMsg(err.message); } finally { setLoading(false); }
  };

  const handleSaveSender = async (e) => {
    e.preventDefault();
    try {
      await saveSender(senderForm.domainID, senderForm);
      setSenderForm({ domainID: null, id: 0, local_part: "", email: "", ip: "", smtp_password: "" });
      await load();
    } catch (err) { setMsg(err.message); }
  };

  const handleDeleteSender = async (id) => {
    if (!confirm("Delete sender?")) return;
    try { await deleteSender(id); await load(); } catch (err) { setMsg(err.message); }
  };

  // --- Copy Helper ---
  const [copied, setCopied] = useState("");
  const copy = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(""), 2000);
  };

  // --- DNS Logic ---
  const dnsHelpers = (d) => {
    const mainIp = settings?.main_server_ip || "SERVER_IP";
    const ips = new Set();
    d.senders?.forEach(s => s.ip && ips.add(s.ip));
    ips.add(mainIp);
    
    const ipParts = Array.from(ips).map(ip => `ip4:${ip}`).join(" ");
    const spfValue = `v=spf1 ${ipParts} ~all`;
    const root = d.name;
    
    return [
      { label: "A (Mail)", value: `${d.mail_host} 3600 IN A ${mainIp}` },
      { label: "A (Bounce)", value: `${d.bounce_host} 3600 IN A ${mainIp}` },
      { label: "MX", value: `${root} 3600 IN MX 10 ${d.mail_host}.` },
      { label: "SPF", value: `${root} 3600 IN TXT "${spfValue}"` }
    ];
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Domains</h2>
          <p className="text-muted-foreground">Manage your sending domains and identities.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowImport(true)} className="flex items-center gap-2 bg-secondary text-secondary-foreground hover:bg-secondary/80 px-4 py-2 rounded-md text-sm font-medium transition-colors">
            <Upload className="w-4 h-4" /> Import CSV
          </button>
          <button onClick={() => setEditingDomain({ id: 0, name: "", mail_host: "", bounce_host: "" })} className="flex items-center gap-2 bg-primary text-primary-foreground hover:bg-primary/90 px-4 py-2 rounded-md text-sm font-medium transition-colors">
            <Plus className="w-4 h-4" /> New Domain
          </button>
        </div>
      </div>

      {msg && <div className="bg-destructive/10 text-destructive p-3 rounded-md text-sm">{msg}</div>}

      {loading ? (
        <div className="text-center py-12 text-muted-foreground">Loading domains...</div>
      ) : domains.length === 0 ? (
        <div className="text-center py-12 border border-dashed rounded-xl bg-card">
          <Globe className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium">No domains found</h3>
          <p className="text-muted-foreground text-sm mt-1">Add your first domain to get started.</p>
        </div>
      ) : (
        <div className="grid gap-6">
          {domains.map((d) => (
            <div key={d.id} className="bg-card border rounded-xl overflow-hidden shadow-sm">
              {/* Domain Header */}
              <div className="p-4 border-b bg-muted/30 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/10 rounded-lg text-primary">
                    <Globe className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-lg">{d.name}</h3>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1"><Server className="w-3 h-3" /> {d.mail_host}</span>
                      <span className="flex items-center gap-1"><Server className="w-3 h-3" /> {d.bounce_host}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => setEditingDomain({ ...d })} className="p-2 hover:bg-background rounded-md text-muted-foreground hover:text-foreground transition-colors"><Edit2 className="w-4 h-4" /></button>
                  <button onClick={() => handleDeleteDomain(d.id)} className="p-2 hover:bg-destructive/10 rounded-md text-muted-foreground hover:text-destructive transition-colors"><Trash2 className="w-4 h-4" /></button>
                </div>
              </div>

              <div className="p-4 grid lg:grid-cols-2 gap-6">
                {/* DNS Helpers */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">DNS Records</h4>
                  <div className="space-y-2">
                    {dnsHelpers(d).map((rec, i) => (
                      <div key={i} className="group flex items-center justify-between gap-2 p-2 rounded-md bg-muted/50 border border-transparent hover:border-border transition-colors">
                        <div className="min-w-0">
                          <div className="text-[10px] text-muted-foreground font-mono uppercase">{rec.label}</div>
                          <div className="text-xs font-mono truncate">{rec.value}</div>
                        </div>
                        <button onClick={() => copy(rec.value, `dns-${d.id}-${i}`)} className="text-muted-foreground hover:text-primary">
                          {copied === `dns-${d.id}-${i}` ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                        </button>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Senders */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Senders</h4>
                    <button onClick={() => setSenderForm({ domainID: d.id, id: 0, local_part: "", email: "", ip: "", smtp_password: "" })} className="text-xs flex items-center gap-1 text-primary hover:underline">
                      <Plus className="w-3 h-3" /> Add Sender
                    </button>
                  </div>
                  
                  {d.senders?.length === 0 ? (
                    <div className="text-sm text-muted-foreground italic">No senders configured.</div>
                  ) : (
                    <div className="space-y-2 max-h-[200px] overflow-y-auto pr-1">
                      {d.senders.map((s) => (
                        <div key={s.id} className="flex items-center justify-between p-2 rounded-md border bg-background text-sm">
                          <div className="flex items-center gap-3">
                            <Mail className="w-4 h-4 text-muted-foreground" />
                            <div>
                              <div className="font-medium">{s.email}</div>
                              <div className="text-xs text-muted-foreground flex gap-2">
                                <span>IP: {s.ip || "Default"}</span>
                                {s.has_dkim && <span className="text-green-600 flex items-center gap-0.5"><ShieldCheck className="w-3 h-3" /> DKIM</span>}
                              </div>
                            </div>
                          </div>
                          <div className="flex gap-1">
                            <button onClick={() => setSenderForm({ domainID: d.id, ...s })} className="p-1.5 hover:bg-muted rounded text-muted-foreground"><Edit2 className="w-3 h-3" /></button>
                            <button onClick={() => handleDeleteSender(s.id)} className="p-1.5 hover:bg-destructive/10 hover:text-destructive rounded text-muted-foreground"><Trash2 className="w-3 h-3" /></button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modals omitted for brevity, but structure follows the Login/Register glass card style */}
      {/* Edit Domain Modal */}
      {editingDomain && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-lg shadow-lg p-6 space-y-4">
            <h3 className="text-lg font-semibold">{editingDomain.id ? "Edit Domain" : "New Domain"}</h3>
            <form onSubmit={handleSaveDomain} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Domain Name</label>
                <input className="w-full h-10 px-3 rounded-md border bg-background" value={editingDomain.name} onChange={e => setEditingDomain({...editingDomain, name: e.target.value})} required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Mail Host</label>
                  <input className="w-full h-10 px-3 rounded-md border bg-background" value={editingDomain.mail_host} onChange={e => setEditingDomain({...editingDomain, mail_host: e.target.value})} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Bounce Host</label>
                  <input className="w-full h-10 px-3 rounded-md border bg-background" value={editingDomain.bounce_host} onChange={e => setEditingDomain({...editingDomain, bounce_host: e.target.value})} />
                </div>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setEditingDomain(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" className="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Sender Modal */}
      {senderForm.domainID && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-lg shadow-lg p-6 space-y-4">
            <h3 className="text-lg font-semibold">{senderForm.id ? "Edit Sender" : "New Sender"}</h3>
            <form onSubmit={handleSaveSender} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Local Part</label>
                <input className="w-full h-10 px-3 rounded-md border bg-background" value={senderForm.local_part} onChange={e => setSenderForm({...senderForm, local_part: e.target.value})} placeholder="e.g. news" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Email Address</label>
                <input className="w-full h-10 px-3 rounded-md border bg-background" value={senderForm.email} onChange={e => setSenderForm({...senderForm, email: e.target.value})} placeholder="news@example.com" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">IP Address</label>
                <select className="w-full h-10 px-3 rounded-md border bg-background" value={senderForm.ip} onChange={e => setSenderForm({...senderForm, ip: e.target.value})}>
                  <option value="">-- Default Server IP --</option>
                  {systemIPs.map(ip => <option key={ip.id} value={ip.value}>{ip.value} ({ip.interface})</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">SMTP Password</label>
                <input type="password" className="w-full h-10 px-3 rounded-md border bg-background" value={senderForm.smtp_password} onChange={e => setSenderForm({...senderForm, smtp_password: e.target.value})} placeholder="Leave blank to keep current" />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setSenderForm({ domainID: null, id: 0, local_part: "", email: "", ip: "", smtp_password: "" })} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" className="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Import Modal */}
      {showImport && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-lg shadow-lg p-6 space-y-4">
            <h3 className="text-lg font-semibold">Bulk Import</h3>
            <p className="text-sm text-muted-foreground">Upload a CSV with headers: <code>domain, localpart, ip, password</code></p>
            <form onSubmit={handleImport} className="space-y-4">
                <input type="file" name="file" accept=".csv" className="w-full text-sm text-muted-foreground file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-semibold file:bg-primary file:text-primary-foreground hover:file:bg-primary/90" required />
                <div className="flex justify-end gap-2">
                    <button type="button" onClick={() => setShowImport(false)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                    <button type="submit" className="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">Import</button>
                </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
