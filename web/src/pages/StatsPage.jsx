import React, { useState, useEffect } from 'react';
import { Line, Bar } from 'react-chartjs-2';
import {
  Chart as ChartJS, CategoryScale, LinearScale, PointElement,
  LineElement, BarElement, Title, Tooltip, Legend, Filler
} from 'chart.js';
import { 
  BarChart3, 
  Send, 
  CheckCircle2, 
  XCircle, 
  Clock, 
  RefreshCw, 
  Filter 
} from 'lucide-react';
import { cn } from '../lib/utils';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend, Filler);

export default function StatsPage() {
  const [stats, setStats] = useState({});
  const [summary, setSummary] = useState(null);
  const [domainList, setDomainList] = useState([]);
  const [days, setDays] = useState(7);
  const [selectedDomain, setSelectedDomain] = useState('');
  const [loading, setLoading] = useState(true);

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}` };

  useEffect(() => { 
    fetchDomains();
    fetchStats(); 
    fetchSummary(); 
  }, [days]);

  const fetchDomains = async () => {
    try {
      const res = await fetch('/api/domains', { headers });
      const data = await res.json();
      setDomainList(Array.isArray(data) ? data : []);
    } catch (e) { console.error(e); }
  };

  const fetchStats = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/stats/domains?days=${days}`, { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      const data = await res.json();
      setStats(data || {});
    } catch (e) { console.error(e); }
    setLoading(false);
  };

  const fetchSummary = async () => {
    try {
      const res = await fetch('/api/stats/summary', { headers });
      if (res.ok) setSummary(await res.json());
    } catch (e) { console.error(e); }
  };

  const refreshStats = async () => {
    await fetch('/api/stats/refresh', { method: 'POST', headers });
    fetchStats(); fetchSummary();
  };

  const availableDomains = domainList.length > 0 
    ? domainList.map(d => d.name) 
    : Object.keys(stats);

  const getChartData = () => {
    if (selectedDomain && stats[selectedDomain]) return stats[selectedDomain];
    
    if (selectedDomain && !stats[selectedDomain]) {
       return Array.from({length: days}).map((_, i) => ({
         date: new Date(Date.now() - (days - 1 - i) * 86400000).toISOString().split('T')[0],
         sent: 0, delivered: 0, bounced: 0
       }));
    }

    const agg = {};
    Object.keys(stats).forEach(d => (stats[d] || []).forEach(day => {
      if (!agg[day.date]) agg[day.date] = { date: day.date, sent: 0, delivered: 0, bounced: 0, deferred: 0 };
      agg[day.date].sent += day.sent || 0;
      agg[day.date].delivered += day.delivered || 0;
      agg[day.date].bounced += day.bounced || 0;
      agg[day.date].deferred += day.deferred || 0;
    }));
    return Object.values(agg).sort((a, b) => a.date.localeCompare(b.date));
  };

  const chartData = getChartData();
  const labels = chartData.map(d => d.date);

  // Chart Styles (Dark Mode Compatible)
  const isDark = document.documentElement.classList.contains('dark');
  const gridColor = isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.05)';
  const textColor = isDark ? '#9ca3af' : '#6b7280';

  const lineData = {
    labels,
    datasets: [
      { label: 'Sent', data: chartData.map(d => d.sent), borderColor: '#3b82f6', backgroundColor: 'rgba(59,130,246,0.1)', fill: true, tension: 0.3 },
      { label: 'Delivered', data: chartData.map(d => d.delivered), borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.1)', fill: true, tension: 0.3 },
      { label: 'Bounced', data: chartData.map(d => d.bounced), borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.1)', fill: true, tension: 0.3 },
    ],
  };

  const barData = {
    labels: availableDomains.slice(0, 10),
    datasets: [
      { 
        label: 'Sent', 
        data: availableDomains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.sent || 0), 0)), 
        backgroundColor: '#3b82f6' 
      },
      { 
        label: 'Bounced', 
        data: availableDomains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.bounced || 0), 0)), 
        backgroundColor: '#ef4444' 
      },
    ],
  };

  const opts = {
    responsive: true, 
    maintainAspectRatio: false,
    plugins: { legend: { position: 'top', labels: { color: textColor } } },
    scales: { x: { ticks: { color: textColor }, grid: { color: gridColor } }, y: { ticks: { color: textColor }, grid: { color: gridColor } } }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Statistics</h1>
          <p className="text-muted-foreground">Email traffic analysis and delivery reports.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <select value={days} onChange={e => setDays(+e.target.value)} className="h-10 rounded-md border bg-background px-3 py-2 text-sm focus:ring-2 focus:ring-ring">
            <option value={7}>Last 7 Days</option>
            <option value={14}>Last 14 Days</option>
            <option value={30}>Last 30 Days</option>
            <option value={90}>Last 90 Days</option>
          </select>
          <div className="relative">
            <Filter className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <select value={selectedDomain} onChange={e => setSelectedDomain(e.target.value)} className="h-10 rounded-md border bg-background pl-9 pr-3 py-2 text-sm focus:ring-2 focus:ring-ring min-w-[150px]">
              <option value="">All Domains</option>
              {availableDomains.map(d => <option key={d} value={d}>{d}</option>)}
            </select>
          </div>
          <button onClick={refreshStats} className="flex items-center gap-2 h-10 px-4 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
        </div>
      </div>

      {summary && (
        <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
          <StatBox label="Total Sent" value={summary.total_sent} icon={Send} color="text-blue-500" />
          <StatBox label="Delivered" value={summary.total_delivered} icon={CheckCircle2} color="text-green-500" />
          <StatBox label="Bounced" value={summary.total_bounced} icon={XCircle} color="text-red-500" />
          <StatBox label="Delivery Rate" value={`${summary.delivery_rate?.toFixed(1)}%`} icon={BarChart3} color="text-emerald-500" />
          <StatBox label="Bounce Rate" value={`${summary.bounce_rate?.toFixed(1)}%`} icon={Clock} color="text-orange-500" />
        </div>
      )}

      <div className="grid lg:grid-cols-2 gap-6">
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold mb-6">Traffic Trend</h3>
          <div className="h-[300px]">
            <Line data={lineData} options={opts} />
          </div>
        </div>
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold mb-6">Volume by Domain</h3>
          <div className="h-[300px]">
            <Bar data={barData} options={opts} />
          </div>
        </div>
      </div>

      <div className="bg-card border rounded-xl shadow-sm overflow-hidden">
        <div className="p-6 border-b">
          <h3 className="text-lg font-semibold">Domain Breakdown</h3>
        </div>
        <div className="overflow-x-auto">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">Loading breakdown...</div>
          ) : (
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground uppercase text-xs">
                <tr>
                  <th className="px-6 py-3 font-medium">Domain</th>
                  <th className="px-6 py-3 font-medium text-right">Sent</th>
                  <th className="px-6 py-3 font-medium text-right">Delivered</th>
                  <th className="px-6 py-3 font-medium text-right">Bounced</th>
                  <th className="px-6 py-3 font-medium text-right">Success Rate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {availableDomains.length === 0 ? (
                  <tr><td colSpan="5" className="p-6 text-center text-muted-foreground">No data available</td></tr>
                ) : (
                  availableDomains.map(domain => {
                    const s = (stats[domain] || []).reduce((a, d) => ({ 
                      sent: a.sent + (d.sent || 0), 
                      delivered: a.delivered + (d.delivered || 0), 
                      bounced: a.bounced + (d.bounced || 0) 
                    }), { sent: 0, delivered: 0, bounced: 0 });
                    
                    const rate = s.sent > 0 ? (s.delivered / s.sent * 100).toFixed(1) : 0;
                    
                    return (
                      <tr key={domain} className="hover:bg-muted/50 transition-colors">
                        <td className="px-6 py-4 font-medium">{domain}</td>
                        <td className="px-6 py-4 text-right tabular-nums">{s.sent.toLocaleString()}</td>
                        <td className="px-6 py-4 text-right tabular-nums text-green-600 dark:text-green-400">{s.delivered.toLocaleString()}</td>
                        <td className="px-6 py-4 text-right tabular-nums text-red-600 dark:text-red-400">{s.bounced.toLocaleString()}</td>
                        <td className="px-6 py-4 text-right">
                          <span className={cn("px-2 py-1 rounded-full text-xs font-medium", 
                            rate > 95 ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" :
                            rate > 80 ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400" :
                            "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                          )}>
                            {rate}%
                          </span>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
}

function StatBox({ label, value, icon: Icon, color }) {
  return (
    <div className="bg-card border rounded-xl p-4 shadow-sm flex flex-col justify-between h-full">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-muted-foreground">{label}</span>
        <Icon className={cn("w-4 h-4", color)} />
      </div>
      <div className="text-2xl font-bold">{value?.toLocaleString() || 0}</div>
    </div>
  );
}
