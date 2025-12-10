import React, { useEffect, useState } from "react";
import { Key, Trash2, Plus, Copy, Check } from "lucide-react";
import { listKeys, createKey, deleteKey } from "../api";

export default function APIKeysPage() {
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [newName, setNewName] = useState("");

  // New key display modal
  const [newKeyData, setNewKeyData] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const data = await listKeys();
      setKeys(data || []);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      const res = await createKey(newName, "relay,verify");
      setNewKeyData(res); // Show the key once
      setShowForm(false);
      setNewName("");
      load();
    } catch (err) {
      alert(err.message);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm("Revoke this key? Apps using it will stop working.")) return;
    await deleteKey(id);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
          <p className="text-muted-foreground">Manage access for external tools (Verification, Relay).</p>
        </div>
        <button 
          onClick={() => setShowForm(!showForm)} 
          className="bg-primary text-primary-foreground px-4 py-2 rounded-md text-sm font-medium flex items-center gap-2 hover:bg-primary/90"
        >
          <Plus className="w-4 h-4" /> Create Key
        </button>
      </div>

      {/* New Key Modal */}
      {newKeyData && (
        <div className="bg-green-500/10 border border-green-500/20 p-6 rounded-xl animate-in fade-in slide-in-from-top-2">
          <h3 className="font-bold text-green-700 mb-2">Key Created Successfully</h3>
          <p className="text-sm text-green-800 mb-4">Copy this key now. You won't be able to see it again.</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-white dark:bg-black/20 p-3 rounded border font-mono text-sm break-all">
              {newKeyData.key}
            </code>
            <CopyButton text={newKeyData.key} />
          </div>
          <button onClick={() => setNewKeyData(null)} className="mt-4 text-sm underline text-green-700">Close</button>
        </div>
      )}

      {/* Create Form */}
      {showForm && !newKeyData && (
        <div className="bg-card border p-4 rounded-xl shadow-sm">
          <form onSubmit={handleCreate} className="flex gap-4 items-end">
            <div className="flex-1 space-y-2">
              <label className="text-sm font-medium">Application Name</label>
              <input 
                value={newName} 
                onChange={e => setNewName(e.target.value)} 
                placeholder="e.g. Email Verifier Tool"
                className="w-full h-10 px-3 rounded-md border bg-background"
                required 
              />
            </div>
            <button type="submit" className="h-10 px-4 bg-primary text-primary-foreground rounded-md text-sm font-medium">
              Generate
            </button>
          </form>
        </div>
      )}

      {/* List */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {keys.map(k => (
          <div key={k.id} className="bg-card border rounded-xl p-4 flex flex-col justify-between shadow-sm hover:shadow-md transition-shadow">
            <div>
              <div className="flex items-start justify-between mb-2">
                <div className="p-2 bg-primary/10 text-primary rounded-lg">
                  <Key className="w-5 h-5" />
                </div>
                <button onClick={() => handleDelete(k.id)} className="text-muted-foreground hover:text-destructive transition-colors">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
              <h3 className="font-semibold">{k.name}</h3>
              <p className="text-xs text-muted-foreground font-mono mt-1">Scopes: {k.scopes}</p>
            </div>
            <div className="mt-4 pt-4 border-t text-[10px] text-muted-foreground flex justify-between">
              <span>Created: {new Date(k.created_at).toLocaleDateString()}</span>
              <span>ID: {k.id}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function CopyButton({ text }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <button onClick={copy} className="p-3 bg-background border rounded hover:bg-muted">
      {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
    </button>
  );
}
