import React, { useEffect, useState } from "react";
import { 
  MailWarning, Filter, Trash2, User, Globe, AlertCircle, Key, Edit2, 
  X, Save, Loader2, Info, Server, Shield 
} from "lucide-react";
import { listBounces, listDomains, deleteBounce, saveBounce } from "../api";
import { cn } from "../lib/utils";

export default function BouncePage() {
  const [list, setList] = useState([]);
  const [domains, setDomains] = useState([]);
  const [filter, setFilter] = useState("");
  const [msg, setMsg] = useState("");
  const [loading, setLoading] = useState(true);
  
  // Edit State
  const [editing, setEditing] = useState(null);
  const [passForm, setPassForm] = useState({ password: "", notes: "" });
  const [busy, setBusy] = useState(false);

  // Info Modal State
  const [showInfo, setShowInfo] = useState(false);

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

  const handleDelete = async (id, username) => {
    if (!confirm(`Delete bounce account "${username}"?\nThis will remove the system user and all bounce emails.`)) return;
    try {
      await deleteBounce(id);
      load();
    } catch (err) {
      setMsg(err.message || "Failed to delete");
    }
  };

  const openEdit = (account) => {
    setEditing(account);
    setPassForm({ password: "", notes: account.notes || "" });
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const payload = {
        id: editing.id,
        username: editing.username,
        domain: editing.domain,
        notes: passForm.notes,
        password: passForm.password || undefined 
      };
      
      await saveBounce(payload);
      setEditing(null);
      setMsg("Account updated successfully");
      load();
    } catch (err) {
      setMsg(err.message || "Failed to update");
    } finally {
      setBusy(false);
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
        
        <div className="flex items-center gap-2">
          <button 
            onClick={() => setShowInfo(true)}
            className="flex items-center gap-2 h-10 px-4 rounded-md border bg-background hover:bg-muted text-sm font-medium transition-colors"
          >
            <Info className="w-4 h-4 text-blue-500" />
            Connection Info
          </button>

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
      </div>

      {msg && <div className="bg-blue-500/10 text-blue-600 p-3 rounded-md text-sm">{msg}</div>}

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
                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button 
                    onClick={() => openEdit(b)}
                    className="p-1.5 text-muted-foreground hover:text-primary hover:bg-muted rounded-md"
                    title="Edit Password/Notes"
                  >
                    <Edit2 className="w-4 h-4" />
                  </button>
                  <button 
                    onClick={() => handleDelete(b.id, b.username)}
                    className="p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md"
                    title="Delete Account"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
              
              <div className="space-y-1">
                <div className="font-mono font-medium text-lg truncate" title={b.username}>{b.username}</div>
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

      {/* Info Modal */}
      {showInfo && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-lg border rounded-lg shadow-lg p-6 space-y-4">
            <div className="flex justify-between items-center border-b pb-4">
              <h3 className="text-lg font-semibold flex items-center gap-2">
                <Server className="w-5 h-5 text-blue-500" /> Server Connection Info
              </h3>
              <button onClick={() => setShowInfo(false)}><X className="w-4 h-4" /></button>
            </div>
            
            <div className="space-y-4 text-sm">
              <p className="text-muted-foreground">Use these settings to connect your bounce processing software (e.g., MailWizz, Mautic) to this server.</p>
              
              <div className="grid gap-3">
                <div className="bg-muted/30 p-3 rounded-md border flex justify-between items-center">
                  <span className="font-medium">Hostname</span>
                  <span className="font-mono bg-background px-2 py-1 rounded">Your Server IP or bounce.domain.com</span>
                </div>
                <div className="bg-muted/30 p-3 rounded-md border flex justify-between items-center">
                  <span className="font-medium">Username</span>
                  <span className="font-mono bg-background px-2 py-1 rounded text-muted-foreground">e.g. b-news</span>
                </div>
                <div className="bg-muted/30 p-3 rounded-md border flex justify-between items-center">
                  <span className="font-medium">Password</span>
                  <span className="font-mono bg-background px-2 py-1 rounded text-muted-foreground">Set during creation</span>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="border rounded-md p-4 space-y-2">
                  <div className="flex items-center gap-2 font-semibold text-primary">
                    <Shield className="w-4 h-4" /> IMAP (SSL)
                  </div>
                  <div className="text-2xl font-bold">Port 993</div>
                  <div className="text-xs text-muted-foreground">Recommended for MailWizz</div>
                </div>
                <div className="border rounded-md p-4 space-y-2">
                  <div className="flex items-center gap-2 font-semibold text-primary">
                    <Shield className="w-4 h-4" /> POP3 (SSL)
                  </div>
                  <div className="text-2xl font-bold">Port 995</div>
                  <div className="text-xs text-muted-foreground">Alternative secure option</div>
                </div>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <button onClick={() => setShowInfo(false)} className="px-4 py-2 text-sm rounded-md bg-secondary hover:bg-secondary/80">Close</button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Modal (unchanged logic, just re-rendering for context) */}
      {editing && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-md border rounded-lg shadow-lg p-6 space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold">Edit Bounce Account</h3>
              <button onClick={() => setEditing(null)}><X className="w-4 h-4" /></button>
            </div>
            
            <div className="p-3 bg-muted rounded-md text-sm space-y-1">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Username:</span>
                <span className="font-mono">{editing.username}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Domain:</span>
                <span>{editing.domain}</span>
              </div>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">New Password</label>
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input 
                    type="password"
                    className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm" 
                    value={passForm.password}
                    onChange={e => setPassForm({...passForm, password: e.target.value})}
                    placeholder="Leave blank to keep current"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Notes</label>
                <textarea 
                  className="w-full rounded-md border bg-background p-3 text-sm h-20 resize-none"
                  value={passForm.notes}
                  onChange={e => setPassForm({...passForm, notes: e.target.value})}
                  placeholder="Optional notes..."
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button type="button" onClick={() => setEditing(null)} className="px-4 py-2 text-sm rounded-md hover:bg-muted">Cancel</button>
                <button type="submit" disabled={busy} className="flex items-center gap-2 px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90">
                  {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
