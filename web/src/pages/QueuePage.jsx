import React, { useState, useEffect } from 'react';
import { 
  RefreshCw, 
  Trash2, 
  Zap, 
  Inbox, 
  Clock, 
  AlertCircle,
  CheckCircle2,
  Mail
} from 'lucide-react';
import { cn } from '../lib/utils';

export default function QueuePage() {
  const [messages, setMessages] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [limit, setLimit] = useState(100);

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

  const formatDate = (d) => d ? new Date(d).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '-';
  const formatSize = (b) => b > 1024 ? `${(b/1024).toFixed(1)} KB` : `${b} B`;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Mail Queue</h1>
          <p className="text-muted-foreground">Monitor and manage outbound messages.</p>
        </div>
        <div className="flex gap-2">
          <select value={limit} onChange={e => setLimit(+e.target.value)} className="h-10 rounded-md border bg-background px-3 py-2 text-sm focus:ring-2 focus:ring-ring">
            <option value={50}>50 Items</option>
            <option value={100}>100 Items</option>
            <option value={500}>500 Items</option>
          </select>
          <button onClick={fetchQueue} className="flex items-center gap-2 h-10 px-4 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm font-medium transition-colors">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
          <button onClick={flushQueue} className="flex items-center gap-2 h-10 px-4 rounded-md bg-amber-600 text-white hover:bg-amber-700 text-sm font-medium transition-colors shadow-sm">
            <Zap className="w-4 h-4" /> Flush Queue
          </button>
        </div>
      </div>

      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <QueueStat label="Total Messages" value={stats.total} icon={Inbox} />
          <QueueStat label="Queued (Active)" value={stats.queued} icon={Mail} color="text-blue-500" />
          <QueueStat label="Deferred (Retry)" value={stats.deferred} icon={Clock} color="text-amber-500" />
          <QueueStat label="Total Size" value={formatSize(stats.total_size)} icon={AlertCircle} />
        </div>
      )}

      <div className="bg-card border rounded-xl overflow-hidden shadow-sm">
        {loading ? (
          <div className="p-12 text-center text-muted-foreground">Loading queue data...</div>
        ) : messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-16 text-center">
            <div className="p-4 bg-green-100 dark:bg-green-900/20 rounded-full mb-4">
              <CheckCircle2 className="w-12 h-12 text-green-600 dark:text-green-400" />
            </div>
            <h3 className="text-xl font-semibold mb-1">Queue is Empty</h3>
            <p className="text-muted-foreground">All messages have been delivered or processed.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  <th className="px-4 py-3 font-medium">ID</th>
                  <th className="px-4 py-3 font-medium">Sender</th>
                  <th className="px-4 py-3 font-medium">Recipient</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Created</th>
                  <th className="px-4 py-3 font-medium text-center">Tries</th>
                  <th className="px-4 py-3 font-medium text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {messages.map(msg => (
                  <tr key={msg.id} className="hover:bg-muted/50 transition-colors group">
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground" title={msg.id}>
                      {msg.id.substring(0, 8)}...
                    </td>
                    <td className="px-4 py-3 truncate max-w-[150px]" title={msg.sender}>{msg.sender || '-'}</td>
                    <td className="px-4 py-3 truncate max-w-[150px]" title={msg.recipient}>{msg.recipient || '-'}</td>
                    <td className="px-4 py-3">
                      <span className={cn(
                        "inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize",
                        msg.status === 'deferred' 
                          ? "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400" 
                          : "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"
                      )}>
                        {msg.status || 'queued'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground whitespace-nowrap">{formatDate(msg.created_at)}</td>
                    <td className="px-4 py-3 text-center">{msg.attempts || 0}</td>
                    <td className="px-4 py-3 text-right">
                      <button 
                        onClick={() => deleteMessage(msg.id)} 
                        className="p-1.5 hover:bg-destructive/10 text-muted-foreground hover:text-destructive rounded-md transition-colors opacity-0 group-hover:opacity-100"
                        title="Delete Message"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Error Summary */}
      {messages.length > 0 && messages.some(m => m.error_msg) && (
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2 text-destructive">
            <AlertCircle className="w-5 h-5" /> Recent Deferral Reasons
          </h3>
          <div className="space-y-2 max-h-60 overflow-y-auto pr-2">
            {messages.filter(m => m.error_msg).slice(0, 10).map((m, i) => (
              <div key={i} className="text-sm p-3 bg-destructive/5 border border-destructive/10 rounded-md">
                <div className="font-medium text-foreground mb-1">{m.recipient}</div>
                <div className="text-destructive font-mono text-xs break-all">{m.error_msg}</div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function QueueStat({ label, value, icon: Icon, color }) {
  return (
    <div className="bg-card border rounded-xl p-4 shadow-sm flex items-center justify-between">
      <div>
        <p className="text-sm font-medium text-muted-foreground">{label}</p>
        <p className="text-2xl font-bold mt-1">{value || 0}</p>
      </div>
      <div className={cn("p-2 rounded-lg bg-secondary", color)}>
        <Icon className="w-5 h-5" />
      </div>
    </div>
  );
}
