import React, { useState, useEffect } from 'react';
import { 
  ShieldCheck, Globe, Settings, Copy, Check, Server, Mail, FileKey, RefreshCw, AlertTriangle
} from 'lucide-react';
import { cn } from '../lib/utils';

export default function DMARCPage() {
  const [domains, setDomains] = useState([]);
  const [selected, setSelected] = useState(null);
  const [dmarc, setDmarc] = useState({ policy: 'none', rua: '', ruf: '', percentage: 100 });
  const [dnsData, setDnsData] = useState(null); // { generated, live }
  const [saving, setSaving] = useState(false);
  const [loadingDNS, setLoadingDNS] = useState(false);
  const [message, setMessage] = useState('');

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  useEffect(() => { fetchDomains(); }, []);

  const fetchDomains = async () => {
    try {
      const res = await fetch('/api/domains', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setDomains(Array.isArray(data) ? data : []);
    } catch (e) { console.error(e); setDomains([]); }
  };

  const selectDomain = async (domain) => {
    setSelected(domain);
    setDmarc({ 
      policy: domain.dmarc_policy || 'none', 
      rua: domain.dmarc_rua || '', 
      ruf: domain.dmarc_ruf || '', 
      percentage: domain.dmarc_percentage || 100 
    });
    setDnsData(null);
    loadDNS(domain.id);
  };

  const loadDNS = async (id) => {
    setLoadingDNS(true);
    try {
      const res = await fetch(`/api/dns/${id}`, { headers });
      if (res.ok) setDnsData(await res.json());
    } catch (e) { console.error(e); }
    setLoadingDNS(false);
  };

  const saveDMARC = async (e) => {
    e.preventDefault();
    if (!selected) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/dmarc/${selected.id}`, { method: 'POST', headers, body: JSON.stringify(dmarc) });
      if (res.ok) { 
        setMessage('DMARC record updated'); 
        loadDNS(selected.id);
        fetchDomains(); 
      }
      else setMessage('Failed to save settings');
    } catch (e) { setMessage('Error: ' + e.message); }
    setSaving(false);
    setTimeout(() => setMessage(''), 3000);
  };

  const [copied, setCopied] = useState("");
  const copyToClipboard = (text, id) => {
    navigator.clipboard.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied(""), 2000);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">DMARC & DNS</h1>
        <p className="text-muted-foreground">Configure policies and verify live DNS records.</p>
      </div>

      {message && (
        <div className="p-4 rounded-md text-sm font-medium bg-green-500/10 text-green-600">
          {message}
        </div>
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        
        {/* Column 1: Domain List */}
        <div className="bg-card border rounded-xl p-4 shadow-sm flex flex-col h-[calc(100vh-200px)]">
          <h3 className="font-semibold mb-4 flex items-center gap-2">
            <Globe className="w-4 h-4 text-muted-foreground" /> Select Domain
          </h3>
          <div className="space-y-2 overflow-y-auto flex-1 pr-2">
            {domains.map(d => (
              <button key={d.id} onClick={() => selectDomain(d)}
                className={cn(
                  "w-full text-left p-3 rounded-lg border transition-all flex items-center justify-between group",
                  selected?.id === d.id 
                    ? "bg-primary/5 border-primary text-primary" 
                    : "bg-background border-transparent hover:bg-muted"
                )}
              >
                <div>
                  <div className="font-medium">{d.name}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">Policy: {d.dmarc_policy || 'none'}</div>
                </div>
                {selected?.id === d.id && <Check className="w-4 h-4" />}
              </button>
            ))}
          </div>
        </div>

        {/* Column 2: Configuration */}
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="font-semibold mb-6 flex items-center gap-2">
            <Settings className="w-4 h-4 text-muted-foreground" /> Policy Config
          </h3>
          {!selected ? (
            <div className="h-full flex flex-col items-center justify-center text-muted-foreground text-sm opacity-50">
              <Globe className="w-12 h-12 mb-2 stroke-1" />
              Select a domain
            </div>
          ) : (
            <form onSubmit={saveDMARC} className="space-y-5">
              <div className="space-y-2">
                <label className="text-sm font-medium">Policy (p)</label>
                <select value={dmarc.policy} onChange={e => setDmarc({...dmarc, policy: e.target.value})}
                  className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                >
                  <option value="none">None (Monitor Only)</option>
                  <option value="quarantine">Quarantine</option>
                  <option value="reject">Reject</option>
                </select>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Aggregate Email (rua)</label>
                <input type="email" value={dmarc.rua} onChange={e => setDmarc({...dmarc, rua: e.target.value})}
                  placeholder="mailto:dmarc@..." className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Forensic Email (ruf)</label>
                <input type="email" value={dmarc.ruf} onChange={e => setDmarc({...dmarc, ruf: e.target.value})}
                  placeholder="mailto:forensic@..." className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Percentage (pct)</label>
                <div className="flex items-center gap-3">
                  <input 
                    type="range" min="0" max="100" 
                    value={dmarc.percentage} onChange={e => setDmarc({...dmarc, percentage: +e.target.value})}
                    className="flex-1"
                  />
                  <span className="w-12 text-right text-sm font-mono">{dmarc.percentage}%</span>
                </div>
              </div>
              <button type="submit" disabled={saving} className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors shadow-sm">
                {saving ? 'Saving...' : 'Save Configuration'}
              </button>
            </form>
          )}
        </div>

        {/* Column 3: DNS Status */}
        <div className="bg-card border rounded-xl p-6 shadow-sm overflow-y-auto h-[calc(100vh-200px)]">
          <div className="flex items-center justify-between mb-6">
            <h3 className="font-semibold flex items-center gap-2">
              <ShieldCheck className="w-4 h-4 text-muted-foreground" /> DNS Status
            </h3>
            {selected && (
              <button onClick={() => loadDNS(selected.id)} className="p-1.5 hover:bg-muted rounded-md transition-colors" title="Refresh Live Data">
                <RefreshCw className={cn("w-4 h-4", loadingDNS && "animate-spin")} />
              </button>
            )}
          </div>
          
          {loadingDNS ? (
            <div className="text-center py-12 text-muted-foreground">Scanning DNS records...</div>
          ) : dnsData ? (
            <div className="space-y-6">
              <DNSSect title="A Records" 
                recs={dnsData.generated.a} 
                live={dnsData.live.a} 
                icon={Server} color="text-blue-500" 
                onCopy={copyToClipboard} copied={copied}
              />
              <DNSSect title="MX Records" 
                recs={dnsData.generated.mx} 
                live={dnsData.live.mx} 
                icon={Mail} color="text-purple-500" 
                onCopy={copyToClipboard} copied={copied}
              />
              <DNSSect title="TXT / SPF" 
                recs={dnsData.generated.spf?.value ? [dnsData.generated.spf] : []} 
                live={dnsData.live.spf?.value ? [dnsData.live.spf] : []} 
                icon={ShieldCheck} color="text-green-500" 
                onCopy={copyToClipboard} copied={copied}
              />
              <DNSSect title="DMARC" 
                recs={dnsData.generated.dmarc?.value ? [dnsData.generated.dmarc] : []} 
                live={dnsData.live.dmarc?.value ? [dnsData.live.dmarc] : []} 
                icon={ShieldCheck} color="text-orange-500" 
                onCopy={copyToClipboard} copied={copied}
              />
              <DNSSect title="DKIM" 
                recs={dnsData.generated.dkim} 
                live={dnsData.live.dkim} 
                icon={FileKey} color="text-pink-500" 
                onCopy={copyToClipboard} copied={copied}
                isDKIM
              />
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground text-sm italic">
              Select a domain to view records
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function DNSSect({ title, recs = [], live = [], icon: Icon, color, onCopy, copied, isDKIM }) {
  if (recs.length === 0 && live.length === 0) return null;

  return (
    <div className="space-y-3 pt-4 first:pt-0 border-t first:border-0">
      <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider flex items-center gap-2">
        <Icon className={cn("w-3.5 h-3.5", color)} /> {title}
      </div>
      
      {recs.map((r, i) => {
        // Find matching live record
        const found = live.find(l => 
          isDKIM 
            ? l.selector === r.selector // Match selector for DKIM
            : l.name === r.name // Match name for others
        );
        
        // Simple status check
        const isMatch = found && (
          isDKIM 
            ? found.dns_value.includes("p=") // Rough check for DKIM key presence
            : found.value === r.value
        );

        return (
          <div key={i} className="text-sm border rounded-lg overflow-hidden">
            <div className="bg-muted/30 p-2 border-b flex justify-between items-center">
              <span className="font-mono text-xs font-semibold">{r.name || r.dns_name}</span>
              <div className="flex gap-2">
                {found ? (
                  isMatch 
                    ? <span className="text-[10px] bg-green-500/10 text-green-600 px-1.5 rounded flex items-center gap-1"><Check className="w-3 h-3" /> Live</span>
                    : <span className="text-[10px] bg-amber-500/10 text-amber-600 px-1.5 rounded flex items-center gap-1">Mismatch</span>
                ) : (
                  <span className="text-[10px] bg-red-500/10 text-red-600 px-1.5 rounded">Not Found</span>
                )}
              </div>
            </div>
            
            {/* Recommended */}
            <div className="p-2 flex gap-2 group">
              <div className="min-w-[60px] text-[10px] uppercase text-muted-foreground pt-0.5">Config</div>
              <code className="flex-1 font-mono text-[10px] break-all text-muted-foreground">
                {r.value || r.dns_value}
              </code>
              <button onClick={() => onCopy(r.value || r.dns_value, r.name)} className="opacity-0 group-hover:opacity-100 transition-opacity">
                {copied === r.name ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3 text-muted-foreground" />}
              </button>
            </div>

            {/* Live (only show if found and different, or just show to confirm) */}
            {found && !isMatch && (
              <div className="p-2 flex gap-2 bg-red-500/5 border-t border-red-100 dark:border-red-900/20">
                <div className="min-w-[60px] text-[10px] uppercase text-red-500 pt-0.5">Found</div>
                <code className="flex-1 font-mono text-[10px] break-all text-red-600 dark:text-red-400">
                  {found.value || found.dns_value}
                </code>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
