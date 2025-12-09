import React, { useState, useEffect } from "react";
import { useAuth } from "../AuthContext";
import { useNavigate } from "react-router-dom";
import { Mail, Lock, Key, ArrowRight, Loader2, ShieldCheck, UserPlus, LogIn } from "lucide-react";
import { cn } from "../lib/utils";

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

  const onSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setBusy(true);

    try {
      if (mode === "register") {
        await register(email, password);
        navigate("/");
      } else {
        if (step === 1) {
          const res = await login(email, password);
          if (res && res.requires_2fa) {
            setTempToken(res.temp_token);
            setStep(2);
            setError("");
          } else {
            navigate("/");
          }
        } else {
          await verify2FA(tempToken, totp);
          navigate("/");
        }
      }
    } catch (err) {
      setError(err.message || "Authentication failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-md bg-card border border-border rounded-xl shadow-lg p-8 space-y-6">
        <div className="text-center space-y-2">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-primary/10 text-primary mb-2">
            {step === 2 ? <ShieldCheck className="w-6 h-6" /> : <Lock className="w-6 h-6" />}
          </div>
          <h1 className="text-2xl font-bold tracking-tight">
            {step === 2 ? "Two-Factor Auth" : "KumoMTA UI"}
          </h1>
          <p className="text-sm text-muted-foreground">
            {step === 2 
              ? "Enter the code from your authenticator app" 
              : mode === "register" 
                ? "Create your admin account" 
                : "Enter your credentials to continue"}
          </p>
        </div>

        <form onSubmit={onSubmit} className="space-y-4">
          {step === 1 ? (
            <>
              <div className="space-y-2">
                <div className="relative">
                  <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <input
                    type="email"
                    placeholder="name@example.com"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pl-9 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    autoFocus
                  />
                </div>
                <div className="relative">
                  <Key className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                  <input
                    type="password"
                    placeholder="Password"
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 pl-9 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                  />
                </div>
              </div>
            </>
          ) : (
            <div className="space-y-2">
              <input
                type="text"
                className="flex h-12 w-full text-center text-2xl tracking-[0.5em] font-mono rounded-md border border-input bg-background px-3 py-2 text-foreground ring-offset-background placeholder:text-muted-foreground/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                value={totp}
                onChange={(e) => setTotp(e.target.value)}
                placeholder="000000"
                maxLength={6}
                required
                autoFocus
              />
            </div>
          )}

          {error && (
            <div className="p-3 rounded-md bg-destructive/15 text-destructive text-sm font-medium text-center">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={busy}
            className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2 w-full"
          >
            {busy ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : step === 2 ? (
              "Verify Code"
            ) : mode === "register" ? (
              <span className="flex items-center">Create Account <ArrowRight className="ml-2 h-4 w-4" /></span>
            ) : (
              <span className="flex items-center">Sign In <ArrowRight className="ml-2 h-4 w-4" /></span>
            )}
          </button>
        </form>

        {step === 1 && (
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-card px-2 text-muted-foreground">Or</span>
            </div>
          </div>
        )}

        {step === 1 && (
          <div className="grid grid-cols-2 gap-4">
            <button
              type="button"
              onClick={() => setMode("login")}
              className={cn(
                "inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2 transition-colors",
                mode === "login" 
                  ? "bg-secondary text-secondary-foreground shadow-sm" 
                  : "ghost hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <LogIn className="mr-2 h-4 w-4" /> Login
            </button>
            <button
              type="button"
              onClick={() => setMode("register")}
              className={cn(
                "inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2 transition-colors",
                mode === "register" 
                  ? "bg-secondary text-secondary-foreground shadow-sm" 
                  : "ghost hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <UserPlus className="mr-2 h-4 w-4" /> Register
            </button>
          </div>
        )}

        {step === 2 && (
          <button
            onClick={() => { setStep(1); setPassword(""); setTempToken(""); }}
            className="w-full text-sm text-muted-foreground hover:text-foreground text-center"
          >
            Back to Login
          </button>
        )}
      </div>
    </div>
  );
}
