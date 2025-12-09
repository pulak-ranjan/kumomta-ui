import React, { useState, useEffect } from 'react';
import { 
  ShieldCheck, 
  Globe, 
  Settings, 
  Copy, 
  Check, 
  Server, 
  Mail, 
  FileKey,
  AlertTriangle 
} from 'lucide-react';
import { listDomains } from "../api"; // Assuming listDomains is exported from api.js
import { cn } from '../lib/utils';

export default function DMARCPage() {
  const [domains, setDomains] = useState([]);
  const [selected, setSelected] = useState(null);
  const [dmarc, setDmarc] = useState({ policy: 'none', rua: '', ruf: '', percentage: 100 });
  const [record, setRecord] = useState(null);
  const [allDns, setAllDns] = useState(null);
  const [saving, setSaving] = useState(false);
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
    setRecord(null); // Clear previous
    setAllDns(null);
    
    try {
      const res = await fetch(`/api/dmarc/${domain.id}`, { headers });
      if (res.ok) setRecord(await res.json());
      
      const dnsRes = await fetch(`/api/dns/${domain.id}`, { headers });
      if (dnsRes.ok) setAllDns(await dnsRes.json());
    } catch (e) { console.error(e); }
  };

  const saveDMARC = async (e) => {
    e.preventDefault();
    if (!selected) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/dmarc/${selected.id}`, { method: 'POST', headers, body: JSON.stringify(dmarc) });
      if (res.ok) { 
        setRecord(await res.json()); 
        setMessage('DMARC record updated successfully'); 
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
        <p className="text-muted-foreground">Configure email authentication policies and view DNS records.</p>
      </div>

      {message && (
        <div className={cn("p-4 rounded-md text-sm font-medium", message.includes("Failed") || message.includes("Error") ? "bg-destructive/10 text-destructive" : "bg-green-500/10 text-green-600")}>
          {message}
        </div>
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        
        {/* Column 1: Domain Selection */}
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
                  <div className="text-xs text-muted-foreground mt-0.5 capitalize">Policy: {d.dmarc_policy || 'none'}</div>
                </div>
                {selected?.id === d.id && <Check className="w-4 h-4" />}
              </button>
            ))}
          </div>
        </div>

        {/* Column 2: Settings Form */}
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="font-semibold mb-6 flex items-center gap-2">
            <Settings className="w-4 h-4 text-muted-foreground" /> Configuration
          </h3>
          {!selected ? (
            <div className="h-full flex flex-col items-center justify-center text-muted-foreground text-sm opacity-50">
              <Globe className="w-12 h-12 mb-2 stroke-1" />
              Select a domain to configure
            </div>
          ) : (
            <form onSubmit={saveDMARC} className="space-y-5">
              <div className="space-y-2">
                <label className="text-sm font-medium">Policy (p)</label>
                <select value={dmarc.policy} onChange={e => setDmarc({...dmarc, policy: e.target.value})}
                  className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                >
                  <option value="none">None (Monitor Only)</option>
                  <option value="quarantine">Quarantine (Spam Folder)</option>
                  <option value="reject">Reject (Block)</option>
                </select>
                <p className="text-[10px] text-muted-foreground">Action to take if checks fail.</p>
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
                {saving ? 'Generating...' : 'Save & Generate Record'}
              </button>
            </form>
          )}
        </div>

        {/* Column 3: DNS Records */}
        <div className="bg-card border rounded-xl p-6 shadow-sm overflow-y-auto h-[calc(100vh-200px)]">
          <h3 className="font-semibold mb-6 flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-muted-foreground" /> DNS Preview
          </h3>
          
          <div className="space-y-6">
            {/* DMARC Result */}
            {record && (
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  <span>_dmarc TXT Record</span>
                  <button onClick={() => copyToClipboard(record.dns_value, 'dmarc-val')} className="hover:text-foreground transition-colors">
                    {copied === 'dmarc-val' ? <Check className="w-3 h-3 text-green-500" /> : <Copy className="w-3 h-3" />}
                  </button>
                </div>
                <div className="p-3 bg-muted/50 border rounded-md font-mono text-xs break-all text-foreground">
                  {record.dns_value}
                </div>
              </div>
            )}

            {/* Other Records */}
            {allDns && (
              <div className="space-y-4 pt-4 border-t">
                {allDns.a?.map((r, i) => (
                  <DNSRow key={`a-${i}`} type="A" name={r.name} value={r.value} icon={Server} color="bg-blue-500/10 text-blue-600" />
                ))}
                {allDns.mx?.map((r, i) => (
                  <DNSRow key={`mx-${i}`} type="MX" name={r.name} value={r.value} icon={Mail} color="bg-purple-500/10 text-purple-600" />
                ))}
                {allDns.spf && (
                  <DNSRow type="SPF" name="TXT" value={allDns.spf.value} icon={ShieldCheck} color="bg-green-500/10 text-green-600" isCopyable onCopy={copyToClipboard} />
                )}
                {allDns.dkim?.map((r, i) => (
                  <DNSRow key={`dkim-${i}`} type="DKIM" name={r.dns_name} value={r.dns_value} icon={FileKey} color="bg-orange-500/10 text-orange-600" isCopyable onCopy={copyToClipboard} />
                ))}
              </div>
            )}

            {!record && !allDns && (
              <div className="text-center py-8 text-muted-foreground text-sm italic">
                Select a domain to view records
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function DNSRow({ type, name, value, icon: Icon, color, isCopyable, onCopy }) {
  const [isHovered, setIsHovered] = useState(false);
  
  return (
    <div 
      className="group relative"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div className="flex items-start gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors">
        <div className={cn("p-1.5 rounded-md shrink-0", color)}>
          <Icon className="w-3.5 h-3.5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-0.5">
            <span className="text-[10px] font-bold uppercase text-muted-foreground">{type}</span>
            <span className="text-xs font-medium truncate">{name}</span>
          </div>
          <div className="text-[11px] font-mono text-muted-foreground break-all leading-tight">
            {value}
          </div>
        </div>
        {isCopyable && (
          <button 
            onClick={() => onCopy(value, value)} // simple id strategy
            className={cn("absolute top-2 right-2 p-1.5 rounded-md hover:bg-background border shadow-sm transition-opacity", isHovered ? "opacity-100" : "opacity-0")}
          >
            <Copy className="w-3 h-3" />
          </button>
        )}
      </div>
    </div>
  );
}
