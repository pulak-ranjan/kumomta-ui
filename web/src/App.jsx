import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ThemeProvider } from './components/ThemeProvider';
import { AuthProvider, useAuth } from './AuthContext';
import Layout from './components/Layout';

// Pages
import LoginRegister from './pages/LoginRegister';
import Dashboard from './pages/Dashboard';
import StatsPage from './pages/StatsPage';
import Domains from './pages/Domains';
import DMARCPage from './pages/DMARCPage';
import DKIMPage from './pages/DKIMPage';
import BouncePage from './pages/BouncePage';
import IPsPage from './pages/IPsPage';
import QueuePage from './pages/QueuePage';
import WebhooksPage from './pages/WebhooksPage';
import ConfigPage from './pages/ConfigPage';
import LogsPage from './pages/LogsPage';
import SecurityPage from './pages/SecurityPage';
import Settings from './pages/Settings';
import ToolsPage from './pages/ToolsPage'; // NEW

function ProtectedRoute({ children }) {
  const { user, loading } = useAuth();
  
  if (loading) return <div className="flex items-center justify-center min-h-screen bg-background text-muted-foreground">Loading...</div>;
  if (!user) return <Navigate to="/login" replace />;
  
  return <Layout>{children}</Layout>;
}

export default function App() {
  return (
    <AuthProvider>
      <ThemeProvider defaultTheme="dark" storageKey="kumoui-theme">
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginRegister />} />
            
            <Route path="/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
            <Route path="/tools" element={<ProtectedRoute><ToolsPage /></ProtectedRoute>} />
            <Route path="/stats" element={<ProtectedRoute><StatsPage /></ProtectedRoute>} />
            <Route path="/domains" element={<ProtectedRoute><Domains /></ProtectedRoute>} />
            <Route path="/dmarc" element={<ProtectedRoute><DMARCPage /></ProtectedRoute>} />
            <Route path="/dkim" element={<ProtectedRoute><DKIMPage /></ProtectedRoute>} />
            <Route path="/bounce" element={<ProtectedRoute><BouncePage /></ProtectedRoute>} />
            <Route path="/ips" element={<ProtectedRoute><IPsPage /></ProtectedRoute>} />
            <Route path="/queue" element={<ProtectedRoute><QueuePage /></ProtectedRoute>} />
            <Route path="/webhooks" element={<ProtectedRoute><WebhooksPage /></ProtectedRoute>} />
            <Route path="/config" element={<ProtectedRoute><ConfigPage /></ProtectedRoute>} />
            <Route path="/logs" element={<ProtectedRoute><LogsPage /></ProtectedRoute>} />
            <Route path="/security" element={<ProtectedRoute><SecurityPage /></ProtectedRoute>} />
            <Route path="/settings" element={<ProtectedRoute><Settings /></ProtectedRoute>} />
            
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </ThemeProvider>
    </AuthProvider>
  );
}
