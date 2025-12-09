import React, { useState, useEffect } from "react";
import { useAuth } from "../AuthContext";
import { useNavigate } from "react-router-dom";

export default function LoginRegister() {
  const { login, verify2FA, register, user } = useAuth();
  const [mode, setMode] = useState("login"); // login vs register
  const [step, setStep] = useState(1); // 1 = creds, 2 = 2fa
  
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [totp, setTotp] = useState("");
  const [tempToken, setTempToken] = useState("");
  
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  
  const navigate = useNavigate();

  useEffect(() => {
    if (user) {
      navigate("/");
    }
  }, [user, navigate]);

  const origin = window.location.origin;
  const panelUrl = origin;
  const apiUrl = origin + "/api";

  const onSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setBusy(true);

    try {
      if (mode === "register") {
        await register(email, password);
        navigate("/");
      } else {
        // Login Flow
        if (step === 1) {
          const res = await login(email, password);
          if (res && res.requires_2fa) {
            // FIX: Handle 2FA requirement
            setTempToken(res.temp_token);
            setStep(2);
            setError("");
          } else {
            // Success (no 2FA)
            navigate("/");
          }
        } else {
          // Step 2: Verify TOTP
          await verify2FA(tempToken, totp);
          navigate("/");
        }
      }
    } catch (err) {
      setError(err.message || "Failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-xl">
        <h1 className="text-xl font-semibold mb-2 text-slate-50">
          Kumo Control Panel
        </h1>
        <p className="text-[11px] text-slate-400 mb-3">
          Panel URL: <span className="font-mono text-slate-200">{panelUrl}</span>
          <br />
          API URL: <span className="font-mono text-slate-200">{apiUrl}</span>
        </p>
        
        <p className="text-xs text-slate-400 mb-4">
          {step === 2 
            ? "Enter your 2FA code to continue." 
            : mode === "register"
                ? "First-time setup: create the first admin user."
                : "Login with your admin account."}
        </p>

        {step === 1 && (
          <div className="flex gap-2 mb-4 text-xs">
            <button
              onClick={() => setMode("login")}
              className={`flex-1 py-1 rounded-md transition-colors ${
                mode === "login"
                  ? "bg-sky-500 text-slate-50"
                  : "bg-slate-800 text-slate-200 hover:bg-slate-700"
              }`}
            >
              Login
            </button>
            <button
              onClick={() => setMode("register")}
              className={`flex-1 py-1 rounded-md transition-colors ${
                mode === "register"
                  ? "bg-sky-500 text-slate-50"
                  : "bg-slate-800 text-slate-200 hover:bg-slate-700"
              }`}
            >
              First-time Setup
            </button>
          </div>
        )}

        <form onSubmit={onSubmit} className="space-y-3 text-sm">
          {step === 1 ? (
            <>
              <div>
                <label className="block text-slate-300 mb-1">Email</label>
                <input
                  type="email"
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-sm outline-none focus:border-sky-500 transition-colors"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-slate-300 mb-1">Password</label>
                <input
                  type="password"
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-sm outline-none focus:border-sky-500 transition-colors"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>
            </>
          ) : (
            <div>
              <label className="block text-slate-300 mb-1">Authenticator Code</label>
              <input
                type="text"
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-center text-xl tracking-widest outline-none focus:border-sky-500 transition-colors"
                value={totp}
                onChange={(e) => setTotp(e.target.value)}
                placeholder="000000"
                maxLength={6}
                required
                autoFocus
              />
            </div>
          )}

          {error && <div className="text-xs text-red-400">{error}</div>}
          
          <button
            type="submit"
            disabled={busy}
            className="w-full mt-2 py-2 rounded-md bg-sky-500 hover:bg-sky-600 text-sm font-medium text-slate-50 transition-colors disabled:opacity-60"
          >
            {busy 
              ? "Working..." 
              : step === 2 
                ? "Verify Code" 
                : mode === "register" 
                    ? "Create Admin" 
                    : "Login"}
          </button>
          
          {step === 2 && (
             <button
                type="button"
                onClick={() => { setStep(1); setPassword(""); setTempToken(""); }}
                className="w-full text-xs text-slate-500 hover:text-slate-300"
             >
                Back to Login
             </button>
          )}
        </form>

        {step === 1 && (
            <p className="mt-4 text-[11px] text-slate-500">
            After installation, open <span className="font-mono">{panelUrl}</span> in your browser and use <b>First-time Setup</b> to create the admin.
            </p>
        )}
      </div>
    </div>
  );
}
