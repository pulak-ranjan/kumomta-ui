import React, { useState, useEffect } from 'react';

export default function DMARCPage() {
  const [domains, setDomains] = useState([]);
  const [selected, setSelected] = useState(null);
  const [dmarc, setDmarc] = useState({ policy: 'none', rua: '', ruf: '', percentage: 100 });
  const [record, setRecord] = useState(null);
  const [allDns, setAllDns] = useState(null);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  const token = localStorage.getItem('token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  useEffect(() => { fetchDomains(); }, []);

  const fetchDomains = async () => {
    try {
      const res = await fetch('/api/domains', { headers });
      setDomains(await res.json() || []);
    } catch (e) { console.error(e); }
  };

  const selectDomain = async (domain) => {
    setSelected(domain);
    setDmarc({ policy: domain.dmarc_policy || 'none', rua: domain.dmarc_rua || '', ruf: domain.dmarc_ruf || '', percentage: domain.dmarc_percentage || 100 });
    try {
      const res = await fetch(`/api/dmarc/${domain.ID}`, { headers });
      setRecord(await res.json());
      const dnsRes = await fetch(`/api/dns/${domain.ID}`, { headers });
      setAllDns(await dnsRes.json());
    } catch (e) { console.error(e); }
  };

  const saveDMARC = async (e) => {
    e.preventDefault();
    if (!selected) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/dmarc/${selected.ID}`, { method: 'POST', headers, body: JSON.stringify(dmarc) });
      if (res.ok) { setRecord(await res.json()); setMessage('✅ DMARC saved!'); fetchDomains(); }
      else setMessage('❌ Failed to save');
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setSaving(false);
    setTimeout(() => setMessage(''), 3000);
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    setMessage('📋 Copied to clipboard!');
    setTimeout(() => setMessage(''), 2000);
  };

  return (
    <div className="p-6 bg-gray-900 min-h-screen text-white">
      <h1 className="text-2xl font-bold mb-6">🛡️ DMARC Generator</h1>

      {message && <div className="mb-4 p-3 bg-gray-800 rounded">{message}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="bg-gray-800 p-4 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">Select Domain</h2>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {domains.map(d => (
              <button key={d.ID} onClick={() => selectDomain(d)}
                className={`w-full text-left p-3 rounded ${selected?.ID === d.ID ? 'bg-blue-600' : 'bg-gray-700 hover:bg-gray-600'}`}>
                <div className="font-semibold">{d.Name}</div>
                <div className="text-sm text-gray-400">Policy: {d.dmarc_policy || 'none'}</div>
              </button>
            ))}
          </div>
        </div>

        <div className="bg-gray-800 p-4 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">DMARC Settings</h2>
          {!selected ? <p className="text-gray-400">Select a domain to configure DMARC</p> : (
            <form onSubmit={saveDMARC} className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Policy</label>
                <select value={dmarc.policy} onChange={e => setDmarc({...dmarc, policy: e.target.value})}
                  className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2">
                  <option value="none">none (monitor only)</option>
                  <option value="quarantine">quarantine (spam folder)</option>
                  <option value="reject">reject (block delivery)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Aggregate Reports (rua)</label>
                <input type="email" value={dmarc.rua} onChange={e => setDmarc({...dmarc, rua: e.target.value})}
                  placeholder="dmarc@yourdomain.com" className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Forensic Reports (ruf)</label>
                <input type="email" value={dmarc.ruf} onChange={e => setDmarc({...dmarc, ruf: e.target.value})}
                  placeholder="forensic@yourdomain.com" className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-1">Percentage (%)</label>
                <input type="number" min="1" max="100" value={dmarc.percentage} onChange={e => setDmarc({...dmarc, percentage: +e.target.value})}
                  className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
              </div>
              <button type="submit" disabled={saving} className="w-full bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded disabled:opacity-50">
                {saving ? 'Saving...' : '💾 Save & Generate'}
              </button>
            </form>
          )}
        </div>

        <div className="space-y-4">
          {record && (
            <div className="bg-gray-800 p-4 rounded-lg">
              <h2 className="text-lg font-semibold mb-4">DMARC Record</h2>
              <div className="space-y-2">
                <div><span className="text-gray-400">Name:</span> <code className="bg-gray-700 px-2 py-1 rounded">{record.dns_name}</code></div>
                <div><span className="text-gray-400">Type:</span> <code className="bg-gray-700 px-2 py-1 rounded">TXT</code></div>
                <div>
                  <span className="text-gray-400">Value:</span>
                  <div className="mt-1 bg-gray-700 p-2 rounded text-sm break-all">{record.dns_value}</div>
                </div>
                <button onClick={() => copyToClipboard(record.dns_value)} className="text-blue-400 hover:text-blue-300 text-sm">📋 Copy Value</button>
              </div>
            </div>
          )}

          {allDns && (
            <div className="bg-gray-800 p-4 rounded-lg">
              <h2 className="text-lg font-semibold mb-4">All DNS Records</h2>
              <div className="space-y-3 text-sm">
                {allDns.a?.map((r, i) => (
                  <div key={i} className="bg-gray-700 p-2 rounded">
                    <div className="text-blue-400">A Record</div>
                    <div>{r.name} → {r.value}</div>
                  </div>
                ))}
                {allDns.mx?.map((r, i) => (
                  <div key={i} className="bg-gray-700 p-2 rounded">
                    <div className="text-purple-400">MX Record</div>
                    <div>{r.name} → {r.value}</div>
                  </div>
                ))}
                {allDns.spf && (
                  <div className="bg-gray-700 p-2 rounded">
                    <div className="text-green-400">SPF Record (TXT)</div>
                    <div className="break-all">{allDns.spf.value}</div>
                    <button onClick={() => copyToClipboard(allDns.spf.value)} className="text-blue-400 text-xs mt-1">📋 Copy</button>
                  </div>
                )}
                {allDns.dkim?.map((r, i) => (
                  <div key={i} className="bg-gray-700 p-2 rounded">
                    <div className="text-yellow-400">DKIM Record (TXT)</div>
                    <div className="text-xs">{r.dns_name}</div>
                    <button onClick={() => copyToClipboard(r.dns_value)} className="text-blue-400 text-xs mt-1">📋 Copy</button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
