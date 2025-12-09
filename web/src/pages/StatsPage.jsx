import React, { useState, useEffect } from 'react';
import { Line, Bar } from 'react-chartjs-2';
import {
  Chart as ChartJS, CategoryScale, LinearScale, PointElement,
  LineElement, BarElement, Title, Tooltip, Legend, Filler
} from 'chart.js';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend, Filler);

export default function StatsPage() {
  const [stats, setStats] = useState({});
  const [summary, setSummary] = useState(null);
  const [days, setDays] = useState(7);
  const [selectedDomain, setSelectedDomain] = useState('');
  const [loading, setLoading] = useState(true);

  const token = localStorage.getItem('token');
  const headers = { Authorization: `Bearer ${token}` };

  useEffect(() => { fetchStats(); fetchSummary(); }, [days]);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/stats/domains?days=${days}`, { headers });
      setStats(await res.json() || {});
    } catch (e) { console.error(e); }
    setLoading(false);
  };

  const fetchSummary = async () => {
    try {
      const res = await fetch('/api/stats/summary', { headers });
      setSummary(await res.json());
    } catch (e) { console.error(e); }
  };

  const refreshStats = async () => {
    await fetch('/api/stats/refresh', { method: 'POST', headers });
    fetchStats(); fetchSummary();
  };

  const domains = Object.keys(stats);

  const getChartData = () => {
    if (selectedDomain && stats[selectedDomain]) return stats[selectedDomain];
    const agg = {};
    domains.forEach(d => (stats[d] || []).forEach(day => {
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

  const lineData = {
    labels,
    datasets: [
      { label: 'Sent', data: chartData.map(d => d.sent), borderColor: '#3b82f6', backgroundColor: 'rgba(59,130,246,0.1)', fill: true, tension: 0.3 },
      { label: 'Delivered', data: chartData.map(d => d.delivered), borderColor: '#22c55e', backgroundColor: 'rgba(34,197,94,0.1)', fill: true, tension: 0.3 },
      { label: 'Bounced', data: chartData.map(d => d.bounced), borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.1)', fill: true, tension: 0.3 },
    ],
  };

  const barData = {
    labels: domains.slice(0, 10),
    datasets: [
      { label: 'Sent', data: domains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.sent || 0), 0)), backgroundColor: '#3b82f6' },
      { label: 'Bounced', data: domains.slice(0, 10).map(d => (stats[d] || []).reduce((s, x) => s + (x.bounced || 0), 0)), backgroundColor: '#ef4444' },
    ],
  };

  const opts = {
    responsive: true, maintainAspectRatio: false,
    plugins: { legend: { position: 'top', labels: { color: '#9ca3af' } } },
    scales: { x: { ticks: { color: '#9ca3af' }, grid: { color: '#374151' } }, y: { ticks: { color: '#9ca3af' }, grid: { color: '#374151' } } }
  };

  return (
    <div className="p-6 bg-gray-900 min-h-screen text-white">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">📊 Email Statistics</h1>
        <div className="flex gap-4">
          <select value={days} onChange={e => setDays(+e.target.value)} className="bg-gray-800 border border-gray-700 rounded px-3 py-2">
            <option value={7}>7 days</option><option value={14}>14 days</option><option value={30}>30 days</option><option value={90}>90 days</option>
          </select>
          <select value={selectedDomain} onChange={e => setSelectedDomain(e.target.value)} className="bg-gray-800 border border-gray-700 rounded px-3 py-2">
            <option value="">All Domains</option>
            {domains.map(d => <option key={d} value={d}>{d}</option>)}
          </select>
          <button onClick={refreshStats} className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded">🔄 Refresh</button>
        </div>
      </div>

      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Total Sent</div><div className="text-2xl font-bold text-blue-400">{summary.total_sent?.toLocaleString() || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Delivered</div><div className="text-2xl font-bold text-green-400">{summary.total_delivered?.toLocaleString() || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Bounced</div><div className="text-2xl font-bold text-red-400">{summary.total_bounced?.toLocaleString() || 0}</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Delivery Rate</div><div className="text-2xl font-bold text-green-400">{summary.delivery_rate?.toFixed(1) || 0}%</div></div>
          <div className="bg-gray-800 p-4 rounded-lg"><div className="text-gray-400 text-sm">Bounce Rate</div><div className="text-2xl font-bold text-red-400">{summary.bounce_rate?.toFixed(1) || 0}%</div></div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-800 p-4 rounded-lg"><h2 className="text-lg font-semibold mb-4">Traffic Over Time</h2><div style={{height:'300px'}}><Line data={lineData} options={opts} /></div></div>
        <div className="bg-gray-800 p-4 rounded-lg"><h2 className="text-lg font-semibold mb-4">Top Domains</h2><div style={{height:'300px'}}><Bar data={barData} options={opts} /></div></div>
      </div>

      <div className="bg-gray-800 p-4 rounded-lg">
        <h2 className="text-lg font-semibold mb-4">Domain Breakdown</h2>
        {loading ? <p>Loading...</p> : (
          <table className="w-full text-sm">
            <thead><tr className="text-gray-400 border-b border-gray-700"><th className="text-left p-2">Domain</th><th className="text-right p-2">Sent</th><th className="text-right p-2">Delivered</th><th className="text-right p-2">Bounced</th><th className="text-right p-2">Rate</th></tr></thead>
            <tbody>
              {domains.map(domain => {
                const s = (stats[domain] || []).reduce((a, d) => ({ sent: a.sent + (d.sent || 0), delivered: a.delivered + (d.delivered || 0), bounced: a.bounced + (d.bounced || 0) }), { sent: 0, delivered: 0, bounced: 0 });
                const rate = s.sent > 0 ? (s.delivered / s.sent * 100).toFixed(1) : 0;
                return (
                  <tr key={domain} className="border-b border-gray-700 hover:bg-gray-700">
                    <td className="p-2">{domain}</td><td className="text-right p-2">{s.sent.toLocaleString()}</td><td className="text-right p-2 text-green-400">{s.delivered.toLocaleString()}</td><td className="text-right p-2 text-red-400">{s.bounced.toLocaleString()}</td><td className="text-right p-2">{rate}%</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
