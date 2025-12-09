import React, { useState, useEffect } from 'react';
import QRCode from 'qrcode';

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
      setUser(await res.json());
    } catch (e) { console.error(e); }
  };

  const fetchSessions = async () => {
    try {
      const res = await fetch('/api/auth/sessions', { headers });
      setSessions(await res.json() || []);
    } catch (e) { console.error(e); }
  };

  const startSetup2FA = async (e) => {
    e.preventDefault();
    if (!password) { setMessage('❌ Enter your password'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/setup-2fa', { method: 'POST', headers, body: JSON.stringify({ password }) });
      if (res.ok) {
        const data = await res.json();
        setSetup2FA(data);
        const qr = await QRCode.toDataURL(data.uri);
        setQrDataUrl(qr);
        setPassword('');
      } else {
        const err = await res.json();
        setMessage('❌ ' + (err.error || 'Failed'));
      }
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setLoading(false);
  };

  const enable2FA = async (e) => {
    e.preventDefault();
    if (!code || code.length !== 6) { setMessage('❌ Enter 6-digit code'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/enable-2fa', { method: 'POST', headers, body: JSON.stringify({ code }) });
      if (res.ok) {
        setMessage('✅ 2FA enabled!');
        setSetup2FA(null);
        setCode('');
        fetchUser();
      } else {
        const err = await res.json();
        setMessage('❌ ' + (err.error || 'Invalid code'));
      }
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setLoading(false);
  };

  const disable2FA = async (e) => {
    e.preventDefault();
    if (!disablePassword || !disableCode) { setMessage('❌ Enter password and code'); return; }
    setLoading(true);
    try {
      const res = await fetch('/api/auth/disable-2fa', { method: 'POST', headers, body: JSON.stringify({ password: disablePassword, code: disableCode }) });
      if (res.ok) {
        setMessage('✅ 2FA disabled');
        setDisablePassword('');
        setDisableCode('');
        fetchUser();
      } else {
        const err = await res.json();
        setMessage('❌ ' + (err.error || 'Failed'));
      }
    } catch (e) { setMessage('❌ Error: ' + e.message); }
    setLoading(false);
  };

  const formatDate = (d) => d ? new Date(d).toLocaleString() : '-';
  const parseUA = (ua) => {
    if (!ua) return 'Unknown';
    if (ua.includes('Mobile')) return '📱 Mobile';
    if (ua.includes('Chrome')) return '🖥️ Chrome';
    if (ua.includes('Firefox')) return '🖥️ Firefox';
    if (ua.includes('Safari')) return '🖥️ Safari';
    return '🖥️ Browser';
  };

  return (
    <div className="p-6 bg-gray-900 min-h-screen text-white">
      <h1 className="text-2xl font-bold mb-6">🔐 Security Settings</h1>

      {message && <div className="mb-4 p-3 bg-gray-800 rounded">{message}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-800 p-6 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">Two-Factor Authentication</h2>
          
          {user?.has_2fa ? (
            <div>
              <div className="flex items-center gap-2 mb-4">
                <span className="text-green-400 text-2xl">✅</span>
                <span>2FA is enabled</span>
              </div>
              <form onSubmit={disable2FA} className="space-y-4">
                <p className="text-gray-400 text-sm">To disable 2FA, enter your password and current code:</p>
                <input type="password" value={disablePassword} onChange={e => setDisablePassword(e.target.value)}
                  placeholder="Password" className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
                <input type="text" value={disableCode} onChange={e => setDisableCode(e.target.value)}
                  placeholder="6-digit code" maxLength={6} className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
                <button type="submit" disabled={loading} className="bg-red-600 hover:bg-red-700 px-4 py-2 rounded disabled:opacity-50">
                  Disable 2FA
                </button>
              </form>
            </div>
          ) : setup2FA ? (
            <div className="space-y-4">
              <p className="text-gray-400">Scan this QR code with your authenticator app:</p>
              {qrDataUrl && <img src={qrDataUrl} alt="QR Code" className="mx-auto bg-white p-2 rounded" />}
              <div className="bg-gray-700 p-3 rounded">
                <p className="text-xs text-gray-400 mb-1">Or enter manually:</p>
                <code className="text-sm break-all">{setup2FA.secret}</code>
              </div>
              <form onSubmit={enable2FA} className="space-y-4">
                <input type="text" value={code} onChange={e => setCode(e.target.value)}
                  placeholder="Enter 6-digit code" maxLength={6}
                  className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2 text-center text-2xl tracking-widest" />
                <button type="submit" disabled={loading} className="w-full bg-green-600 hover:bg-green-700 px-4 py-2 rounded disabled:opacity-50">
                  {loading ? 'Verifying...' : 'Verify & Enable'}
                </button>
              </form>
            </div>
          ) : (
            <form onSubmit={startSetup2FA} className="space-y-4">
              <p className="text-gray-400">Add an extra layer of security with TOTP-based 2FA.</p>
              <input type="password" value={password} onChange={e => setPassword(e.target.value)}
                placeholder="Enter your password to begin" className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2" />
              <button type="submit" disabled={loading} className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded disabled:opacity-50">
                {loading ? 'Loading...' : '🔑 Setup 2FA'}
              </button>
            </form>
          )}
        </div>

        <div className="bg-gray-800 p-6 rounded-lg">
          <h2 className="text-lg font-semibold mb-4">Active Sessions</h2>
          <p className="text-gray-400 text-sm mb-4">You can be logged in on up to 3 devices. Oldest sessions are removed automatically.</p>
          <div className="space-y-3">
            {sessions.map((sess, i) => (
              <div key={i} className="bg-gray-700 p-3 rounded flex justify-between items-center">
                <div>
                  <div className="font-semibold">{parseUA(sess.user_agent)}</div>
                  <div className="text-sm text-gray-400">{sess.device_ip}</div>
                  <div className="text-xs text-gray-500">Active since {formatDate(sess.created_at)}</div>
                </div>
                {i === 0 && <span className="text-green-400 text-xs">Current</span>}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-6 bg-gray-800 p-6 rounded-lg">
        <h2 className="text-lg font-semibold mb-4">Account Info</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><span className="text-gray-400">Email:</span> {user?.email}</div>
          <div><span className="text-gray-400">2FA:</span> {user?.has_2fa ? '✅ Enabled' : '❌ Disabled'}</div>
        </div>
      </div>
    </div>
  );
}
