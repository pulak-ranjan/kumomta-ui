import React, { useState, useEffect } from 'react';

export default function QueuePage() {
  const [messages, setMessages] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [limit, setLimit] = useState(100);

  // FIX: Use correct token key
  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  useEffect(() => { fetchQueue(); fetchStats(); }, [limit]);

  const fetchQueue = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/queue?limit=${limit}`, { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setMessages(Array.isArray(data) ? data : []);
    } catch (e) { console.error(e); setMessages([]); }
    setLoading(false);
  };

  const fetchStats = async () => {
    try {
      const res = await fetch('/api/queue/stats', { headers });
      if (res.ok) setStats(await res.json());
    } catch (e) { console.error(e); }
  };

  const deleteMessage = async (id) => {
    if (!confirm('Delete this message from queue?')) return;
    try {
      await fetch(`/api/queue/${id}`, { method: 'DELETE', headers });
      fetchQueue(); fetchStats();
    } catch (e) { console.error(e); }
  };

  const flushQueue = async () => {
    if (!confirm('Retry all deferred messages?')) return;
    try {
      await fetch('/api/queue/flush', { method: 'POST', headers });
      fetchQueue(); fetchStats();
    } catch (e) { console.error(e); }
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString() : '-';
  const formatSize = (b) => b > 1024 ? `${(b/1024).toFixed(1)} KB` : `${b} B`;

  return (
    <div className="p-6 bg-gray-900 min-h-screen text-white">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">📬 Email Queue</h1>
        <div className="flex gap-4">
          <select value={limit} onChange={e => setLimit(+e.target.value)} className="bg-gray-800 border border-gray-700 rounded px-3 py-2">
            <option value={50}>50 messages</option><option value={100}>100 messages</option><option value={500}>500 messages</option><option value={1000}>1000 messages</option>
          </select>
          <button onClick={fetchQueue} className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded">🔄 Refresh</button>
          <button onClick={flushQueue} className="bg-yellow-600 hover:bg-yellow-700 px-4 py-2 rounded">⚡ Flush Queue</button>
        </div>
      </div>

      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Total Messages</div><div className="text-2xl font-bold">{stats.total || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Queued</div><div className="text-2xl font-bold text-blue-400">{stats.queued || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Deferred</div><div className="text-2xl font-bold text-yellow-400">{stats.deferred || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Total Size</div><div className="text-2xl font-bold">{formatSize(stats.total_size || 0)}</div></div>
        </div>
      )}

      <div className="bg-gray-800 rounded-lg overflow-hidden">
        {loading ? <p className="p-4">Loading...</p> : messages.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <div className="text-4xl mb-2">✅</div>
            <p>Queue is empty - all messages delivered!</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead><tr className="bg-gray-700 text-gray-300">
              <th className="text-left p-3">ID</th><th className="text-left p-3">Sender</th><th className="text-left p-3">Recipient</th><th className="text-left p-3">Status</th><th className="text-left p-3">Created</th><th className="text-left p-3">Attempts</th><th className="text-left p-3">Actions</th>
            </tr></thead>
            <tbody>
              {messages.map(msg => (
                <tr key={msg.id} className="border-b border-gray-700 hover:bg-gray-700">
                  <td className="p-3 font-mono text-xs">{msg.id?.substring(0, 8)}...</td>
                  <td className="p-3">{msg.sender || '-'}</td>
                  <td className="p-3">{msg.recipient || '-'}</td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${msg.status === 'deferred' ? 'bg-yellow-600' : 'bg-blue-600'}`}>
                      {msg.status || 'queued'}
                    </span>
                  </td>
                  <td className="p-3 text-gray-400">{formatDate(msg.created_at)}</td>
                  <td className="p-3">{msg.attempts || 0}</td>
                  <td className="p-3">
                    <button onClick={() => deleteMessage(msg.id)} className="text-red-400 hover:text-red-300">🗑️</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {messages.length > 0 && messages.some(m => m.error_msg) && (
        <div className="mt-6 bg-gray-800 p-4 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">⚠️ Recent Errors</h2>
          <div className="space-y-2">
            {messages.filter(m => m.error_msg).slice(0, 5).map(m => (
              <div key={m.id} className="bg-gray-700 p-3 rounded text-sm">
                <span className="text-gray-400">{m.recipient}:</span> <span className="text-red-400">{m.error_msg}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
