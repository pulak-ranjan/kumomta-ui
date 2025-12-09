import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import { ThemeProvider, ThemeToggleCompact } from './components/ThemeProvider';

// Pages
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import DomainsPage from './pages/DomainsPage';
import SendersPage from './pages/SendersPage';
import BouncePage from './pages/BouncePage';
import IPsPage from './pages/IPsPage';
import DKIMPage from './pages/DKIMPage';
import SettingsPage from './pages/SettingsPage';
import SystemPage from './pages/SystemPage';
import ImportPage from './pages/ImportPage';

// New pages
import StatsPage from './pages/StatsPage';
import QueuePage from './pages/QueuePage';
import WebhooksPage from './pages/WebhooksPage';
import DMARCPage from './pages/DMARCPage';
import SecurityPage from './pages/SecurityPage';

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('token');
  if (!token) return <Navigate to="/login" replace />;
  return children;
}

function Sidebar() {
  const location = useLocation();
  const isActive = (path) => location.pathname === path;

  const links = [
    { path: '/', icon: '📊', label: 'Dashboard' },
    { path: '/stats', icon: '📈', label: 'Statistics' },
    { path: '/domains', icon: '🌐', label: 'Domains' },
    { path: '/dmarc', icon: '🛡️', label: 'DMARC' },
    { path: '/dkim', icon: '🔑', label: 'DKIM' },
    { path: '/bounce', icon: '📬', label: 'Bounce' },
    { path: '/ips', icon: '🖥️', label: 'IPs' },
    { path: '/queue', icon: '📤', label: 'Queue' },
    { path: '/webhooks', icon: '🔔', label: 'Webhooks' },
    { path: '/import', icon: '📥', label: 'Import' },
    { path: '/system', icon: '⚙️', label: 'System' },
    { path: '/security', icon: '🔐', label: 'Security' },
    { path: '/settings', icon: '⚙️', label: 'Settings' },
  ];

  const logout = () => {
    localStorage.removeItem('token');
    window.location.href = '/login';
  };

  return (
    <div className="w-64 bg-gray-800 dark:bg-gray-900 min-h-screen p-4 flex flex-col">
      <div className="text-xl font-bold text-white mb-8 flex items-center gap-2">
        <span className="text-2xl">📧</span> KumoMTA UI
      </div>
      
      <nav className="flex-1 space-y-1">
        {links.map(link => (
          <Link
            key={link.path}
            to={link.path}
            className={`flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
              isActive(link.path)
                ? 'bg-blue-600 text-white'
                : 'text-gray-300 hover:bg-gray-700 hover:text-white'
            }`}
          >
            <span>{link.icon}</span>
            <span>{link.label}</span>
          </Link>
        ))}
      </nav>

      <div className="border-t border-gray-700 pt-4 space-y-2">
        <div className="flex justify-between items-center px-3">
          <span className="text-gray-400 text-sm">Theme</span>
          <ThemeToggleCompact />
        </div>
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-red-400 hover:bg-gray-700 transition-colors"
        >
          <span>🚪</span>
          <span>Logout</span>
        </button>
      </div>
    </div>
  );
}

function Layout({ children }) {
  return (
    <div className="flex min-h-screen bg-gray-900 dark:bg-gray-950">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          
          <Route path="/" element={
            <ProtectedRoute>
              <Layout><DashboardPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/stats" element={
            <ProtectedRoute>
              <Layout><StatsPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/domains" element={
            <ProtectedRoute>
              <Layout><DomainsPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/domains/:id/senders" element={
            <ProtectedRoute>
              <Layout><SendersPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/dmarc" element={
            <ProtectedRoute>
              <Layout><DMARCPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/dkim" element={
            <ProtectedRoute>
              <Layout><DKIMPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/bounce" element={
            <ProtectedRoute>
              <Layout><BouncePage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/ips" element={
            <ProtectedRoute>
              <Layout><IPsPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/queue" element={
            <ProtectedRoute>
              <Layout><QueuePage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/webhooks" element={
            <ProtectedRoute>
              <Layout><WebhooksPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/import" element={
            <ProtectedRoute>
              <Layout><ImportPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/system" element={
            <ProtectedRoute>
              <Layout><SystemPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/security" element={
            <ProtectedRoute>
              <Layout><SecurityPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="/settings" element={
            <ProtectedRoute>
              <Layout><SettingsPage /></Layout>
            </ProtectedRoute>
          } />
          
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}
