import React, { useState } from "react";
import { AuthProvider, useAuth } from "./AuthContext";
import Dashboard from "./pages/Dashboard";
import Settings from "./pages/Settings";
import Domains from "./pages/Domains";
import ConfigPage from "./pages/ConfigPage";
import DKIMPage from "./pages/DKIMPage";
import BouncePage from "./pages/BouncePage";
import LogsPage from "./pages/LogsPage";
import LoginRegister from "./pages/LoginRegister";

function AppShell() {
  const { user, loading, logout } = useAuth();
  const [tab, setTab] = useState("dashboard");

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-50">
        <div className="text-lg">Loading...</div>
      </div>
    );
  }

  if (!user) {
    return <LoginRegister />;
  }

  const renderTab = () => {
    switch (tab) {
      case "dashboard":
        return <Dashboard />;
      case "settings":
        return <Settings />;
      case "domains":
        return <Domains />;
      case "config":
        return <ConfigPage />;
      case "dkim":
        return <DKIMPage />;
      case "bounce":
        return <BouncePage />;
      case "logs":
        return <LogsPage />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col">
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur">
        <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
          <div className="font-semibold tracking-wide">
            Kumo Control Panel
          </div>
          <div className="flex items-center gap-4 text-sm">
            <span className="text-slate-300">{user.email}</span>
            <button
              onClick={logout}
              className="px-3 py-1 rounded-md bg-slate-800 hover:bg-slate-700 text-xs"
            >
              Logout
            </button>
          </div>
        </div>
        <nav className="max-w-6xl mx-auto px-4 pb-2 flex gap-2 text-sm">
          {[
            ["dashboard", "Dashboard"],
            ["settings", "Settings"],
            ["domains", "Domains & Senders"],
            ["config", "Config"],
            ["dkim", "DKIM"],
            ["bounce", "Bounce"],
            ["logs", "Logs"]
          ].map(([key, label]) => (
            <button
              key={key}
              onClick={() => setTab(key)}
              className={`px-3 py-1 rounded-md ${
                tab === key ? "bg-sky-500 text-slate-50" : "bg-slate-800 text-slate-200 hover:bg-slate-700"
              }`}
            >
              {label}
            </button>
          ))}
        </nav>
      </header>

      <main className="flex-1 max-w-6xl mx-auto w-full px-4 py-4">
        {renderTab()}
      </main>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppShell />
    </AuthProvider>
  );
}
