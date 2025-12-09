import React, { useState, useEffect } from 'react';
import QRCode from 'qrcode';
import { 
  Shield, 
  Smartphone, 
  Monitor, 
  LogOut, 
  Key, 
  CheckCircle2, 
  AlertOctagon,
  ScanLine
} from 'lucide-react';
import { useAuth } from "../AuthContext"; // Need to fetch user from context or api
import { cn } from "../lib/utils";

export default function SecurityPage() {
  const [user, setUser] = useState(null);
  const [sessions, setSessions] = useState([]);
  const [setup2FA, setSetup2FA] = useState(null);
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disablePassword, setDisablePassword] = useState('');
  const [qrDataUrl, setQrDataUrl] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);

  const token = localStorage.getItem('kumoui_token');
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

  useEffect(() => { fetchUser(); fetchSessions(); }, []);

  const fetchUser = async () => {
    try {
      const res = await fetch('/api/auth/me', { headers });
      if (res.status === 401) { window.location.href = '/login'; return; }
      setUser(await res.json());
    } catch (e) { console.error(e); }
  };

  const fetchSessions = async () => {
    try {
      const res = await fetch('/api/auth/sessions', { headers });
      if (res.ok) setSessions(await res.json() || []);
    } catch (e) { console.error(e); }
  };

  const startSetup2FA = async (e) => {
    e.preventDefault();
    if (!password) { setMessage('Enter your password'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/setup-2fa', { method: 'POST', headers, body: JSON.stringify({ password }) });
      if (res.ok) {
        const data = await res.json();
        setSetup2FA(data);
        const qr = await QRCode.toDataURL(data.uri);
        setQrDataUrl(qr);
        setPassword('');
        setMessage('');
      } else {
        const err = await res.json();
        setMessage(err.error || 'Failed');
      }
    } catch (e) { setMessage('Error: ' + e.message); }
    setLoading(false);
  };

  const enable2FA = async (e) => {
    e.preventDefault();
    if (!code || code.length !== 6) { setMessage('Enter 6-digit code'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/enable-2fa', { method: 'POST', headers, body: JSON.stringify({ code }) });
      if (res.ok) {
        setMessage('2FA enabled successfully!');
        setSetup2FA(null);
        setCode('');
        fetchUser();
      } else {
        const err = await res.json();
        setMessage(err.error || 'Invalid code');
      }
    } catch (e) { setMessage('Error: ' + e.message); }
    setLoading(false);
  };

  const disable2FA = async (e) => {
    e.preventDefault();
    if (!disablePassword || !disableCode) { setMessage('Enter password and code'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/disable-2fa', { method: 'POST', headers, body: JSON.stringify({ password: disablePassword, code: disableCode }) });
      if (res.ok) {
        setMessage('2FA disabled');
        setDisablePassword('');
        setDisableCode('');
        fetchUser();
      } else {
        const err = await res.json();
        setMessage(err.error || 'Failed');
      }
    } catch (e) { setMessage('Error: ' + e.message); }
    setLoading(false);
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString() : '-';
  const getDeviceIcon = (ua) => {
    if (!ua) return Monitor;
    if (ua.includes('Mobile')) return Smartphone;
    return Monitor;
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Security</h1>
        <p className="text-muted-foreground">Manage your account security and active sessions.</p>
      </div>

      {message && (
        <div className={cn("p-4 rounded-md text-sm font-medium", message.includes("enabled") ? "bg-green-500/10 text-green-600" : "bg-destructive/10 text-destructive")}>
          {message}
        </div>
      )}

      <div className="grid lg:grid-cols-2 gap-6">
        
        {/* 2FA Card */}
        <div className="bg-card border rounded-xl p-6 shadow-sm h-fit">
          <h3 className="text-lg font-semibold mb-6 flex items-center gap-2">
            <Shield className="w-5 h-5 text-primary" /> Two-Factor Authentication
          </h3>
          
          {user?.has_2fa ? (
            <div className="space-y-6">
              <div className="flex items-center gap-3 p-4 bg-green-500/10 border border-green-500/20 rounded-lg text-green-700 dark:text-green-400">
                <CheckCircle2 className="w-6 h-6" />
                <div>
                  <div className="font-semibold">2FA is active</div>
                  <div className="text-xs opacity-90">Your account is secured with TOTP.</div>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h4 className="text-sm font-medium mb-4">Disable 2FA</h4>
                <form onSubmit={disable2FA} className="space-y-3">
                  <input type="password" value={disablePassword} onChange={e => setDisablePassword(e.target.value)}
                    placeholder="Current Password" className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
                  <input type="text" value={disableCode} onChange={e => setDisableCode(e.target.value)}
                    placeholder="6-Digit Authenticator Code" maxLength={6} className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
                  <button type="submit" disabled={loading} className="w-full h-10 bg-destructive text-destructive-foreground rounded-md text-sm font-medium hover:bg-destructive/90 transition-colors">
                    Disable 2FA
                  </button>
                </form>
              </div>
            </div>
          ) : setup2FA ? (
            <div className="space-y-6 animate-in fade-in">
              <div className="text-center space-y-4">
                <div className="bg-white p-4 rounded-xl inline-block shadow-sm">
                  {qrDataUrl && <img src={qrDataUrl} alt="QR Code" className="w-48 h-48" />}
                </div>
                <div className="text-sm text-muted-foreground">
                  <p>Scan this QR code with your authenticator app.</p>
                  <p className="mt-2 text-xs font-mono bg-muted p-2 rounded select-all">{setup2FA.secret}</p>
                </div>
              </div>

              <form onSubmit={enable2FA} className="space-y-3">
                <input type="text" value={code} onChange={e => setCode(e.target.value)}
                  placeholder="Enter 6-digit code" maxLength={6}
                  className="w-full h-12 text-center text-xl tracking-[0.5em] font-mono rounded-md border bg-background px-3 focus:ring-2 focus:ring-ring" />
                <button type="submit" disabled={loading} className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors">
                  {loading ? 'Verifying...' : 'Verify & Enable'}
                </button>
              </form>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="p-4 bg-muted/50 rounded-lg text-sm text-muted-foreground">
                Protect your admin account by requiring a code from your phone when logging in.
              </div>
              <form onSubmit={startSetup2FA} className="space-y-3">
                <div className="relative">
                  <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <input type="password" value={password} onChange={e => setPassword(e.target.value)}
                    placeholder="Enter password to start setup" className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring" />
                </div>
                <button type="submit" disabled={loading} className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2">
                  <ScanLine className="w-4 h-4" /> Setup 2FA
                </button>
              </form>
            </div>
          )}
        </div>

        {/* Sessions Card */}
        <div className="bg-card border rounded-xl p-6 shadow-sm h-fit">
          <h3 className="text-lg font-semibold mb-6 flex items-center gap-2">
            <Monitor className="w-5 h-5 text-blue-500" /> Active Sessions
          </h3>
          <p className="text-sm text-muted-foreground mb-4">
            You are logged in on these devices. The system automatically rotates old sessions.
          </p>
          
          <div className="space-y-1">
            {sessions.map((sess, i) => {
              const Icon = getDeviceIcon(sess.user_agent);
              return (
                <div key={i} className="flex items-start gap-3 p-3 rounded-lg hover:bg-muted/50 transition-colors border border-transparent hover:border-border">
                  <div className="p-2 bg-secondary rounded-full">
                    <Icon className="w-4 h-4 text-secondary-foreground" />
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm">
                        {sess.user_agent ? sess.user_agent.split('/')[0] : 'Unknown Device'}
                      </span>
                      {i === 0 && <span className="bg-green-500/10 text-green-600 text-[10px] font-bold px-1.5 py-0.5 rounded uppercase">Current</span>}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5 font-mono">
                      IP: {sess.device_ip}
                    </div>
                    <div className="text-[10px] text-muted-foreground/70 mt-1">
                      Started: {formatDate(sess.created_at)}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
