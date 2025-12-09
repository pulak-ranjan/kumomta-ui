import React, { useState, useEffect } from 'react';

export default function WebhooksPage() {
  const [settings, setSettings] = useState({ webhook_url: '', webhook_enabled: false, bounce_alert_pct: 5 });
  const [logs, setLogs] = useState([]);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  useEffect(() => { fetchSettings(); fetchLogs(); }, []);

  const fetchSettings = async () => {
    try {
      const res = await fetch('/api/webhooks/settings', { headers });
      setSettings(await res.json());
    } catch (e) { console.error(e); }
  };

  const fetchLogs = async () => {
    try {
      const res = await fetch('/api/webhooks/logs', { headers });
      setLogs(await res.json() || []);
    } catch (e) { console.error(e); }
  };

  const saveSettings = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const res = await fetch('/api/webhooks/settings', { method: 'POST', headers, body: JSON.stringify(settings) });
      if (res.ok) setMessage('✅ Settings saved!');
      else setMessage('❌ Failed to save');
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setSaving(false);
    setTimeout(() => setMessage(''), 3000);
  };

  const testWebhook = async () => {
    if (!settings.webhook_url) { setMessage('❌ Enter webhook URL first'); return; }
    setTesting(true);
    try {
      const res = await fetch('/api/webhooks/test', { method: 'POST', headers, body: JSON.stringify({ webhook_url: settings.webhook_url }) });
      if (res.ok) { setMessage('✅ Test sent!'); fetchLogs(); }
      else setMessage('❌ Test failed');
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setTesting(false);
    setTimeout(() => setMessage(''), 3000);
  };

  const checkBounces = async () => {
    try {
      await fetch('/api/webhooks/check-bounces', { method: 'POST', headers });
      setMessage('✅ Bounce check triggered');
      fetchLogs();
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setTimeout(() => setMessage(''), 3000);
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString() : '-';

  return (
    <div className="p-6 bg-gray-900 min-h-screen text-white">
      <h1 className="text-2xl font-bold mb-6">🔔 Webhook Alerts</h1>

      {message && <div className="mb-4 p-3 bg-gray-800 rounded">{message}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 p-6 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">Settings</h2>
          <form onSubmit={saveSettings} className="space-y-4">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Webhook URL (Slack/Discord)</label>
              <input type="url" value={settings.webhook_url} onChange={e => setSettings({...settings, webhook_url: e.target.value})}
                placeholder="https://hooks.slack.com/..." className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Bounce Alert Threshold (%)</label>
              <input type="number" min="1" max="100" value={settings.bounce_alert_pct} onChange={e => setSettings({...settings, bounce_alert_pct: +e.target.value})}
                className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
              <p className="text-xs text-gray-500 mt-1">Alert when bounce rate exceeds this percentage</p>
            </div>
            <div className="flex items-center gap-2">
              <input type="checkbox" id="enabled" checked={settings.webhook_enabled} onChange={e => setSettings({...settings, webhook_enabled: e.target.checked})}
                className="w-4 h-4 rounded" />
              <label htmlFor="enabled">Enable webhook alerts</label>
            </div>
            <div className="flex gap-2">
              <button type="submit" disabled={saving} className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded disabled:opacity-50">
                {saving ? 'Saving...' : '💾 Save'}
              </button>
              <button type="button" onClick={testWebhook} disabled={testing} className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded disabled:opacity-50">
                {testing ? 'Sending...' : '🧪 Test'}
              </button>
            </div>
          </form>
        </div>

        <div className="bg-gray-800 p-6 rounded-lg">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-lg font-semibold">Actions</h2>
          </div>
          <div className="space-y-4">
            <button onClick={checkBounces} className="w-full bg-yellow-600 hover:bg-yellow-700 px-4 py-3 rounded text-left">
              <div className="font-semibold">⚠️ Check Bounce Rates</div>
              <div className="text-sm text-yellow-200">Analyze current bounce rates and send alerts if threshold exceeded</div>
            </button>
            <div className="bg-gray-700 p-4 rounded">
              <h3 className="font-semibold mb-2">📋 Supported Platforms</h3>
              <ul className="text-sm text-gray-400 space-y-1">
                <li>• <strong>Slack:</strong> Create incoming webhook in Slack App settings</li>
                <li>• <strong>Discord:</strong> Edit channel → Integrations → Webhooks</li>
                <li>• Both receive formatted alerts with stats</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      <div className="mt-6 bg-gray-800 p-6 rounded-lg">
        <h2 className="text-lg font-semibold mb-4">📜 Recent Webhook Activity</h2>
        {logs.length === 0 ? <p className="text-gray-400">No webhook activity yet</p> : (
          <table className="w-full text-sm">
            <thead><tr className="text-gray-400 border-b border-gray-700">
              <th className="text-left p-2">Time</th><th className="text-left p-2">Event</th><th className="text-left p-2">Status</th><th className="text-left p-2">Response</th>
            </tr></thead>
            <tbody>
              {logs.map((log, i) => (
                <tr key={i} className="border-b border-gray-700">
                  <td className="p-2 text-gray-400">{formatDate(log.created_at)}</td>
                  <td className="p-2">{log.event_type}</td>
                  <td className="p-2">
                    <span className={`px-2 py-1 rounded text-xs ${log.status >= 200 && log.status < 300 ? 'bg-green-600' : 'bg-red-600'}`}>
                      {log.status}
                    </span>
                  </td>
                  <td className="p-2 text-gray-400 text-xs truncate max-w-xs">{log.response}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
