import React, { useState, useEffect } from "react";
import { useAuth } from "../AuthContext";
import { useNavigate } from "react-router-dom"; // 1. Import useNavigate

export default function LoginRegister() {
  const { login, register, user } = useAuth(); // 2. Get user to check status
  const [mode, setMode] = useState("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate(); // 3. Initialize hook

  // 4. Redirect if user is already logged in
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
    try {
      if (mode === "register") {
        await register(email, password);
      } else {
        await login(email, password);
      }
      // 5. Navigate to dashboard on successful submit
      navigate("/");
    } catch (err) {
      setError(err.message || "Failed");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-xl">
        <h1 className="text-xl font-semibold mb-2 text-slate-50">
          Kumo Control Panel
        </h1>
        <p className="text-[11px] text-slate-400 mb-3">
          Panel URL:{" "}
          <span className="font-mono text-slate-200">{panelUrl}</span>
          <br />
          API URL:{" "}
          <span className="font-mono text-slate-200">{apiUrl}</span>
        </p>
        <p className="text-xs text-slate-400 mb-4">
          {mode === "register"
            ? "First-time setup: create the first admin user."
            : "Login with your admin account."}
        </p>

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

        <form onSubmit={onSubmit} className="space-y-3 text-sm">
          <div>
            <label className="block text-slate-300 mb-1">Email</label>
            <input
              type="email"
              className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-sm outline-none focus:border-sky-500 transition-colors"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
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
          {error && (
            <div className="text-xs text-red-400">
              {error}
            </div>
          )}
          <button
            type="submit"
            className="w-full mt-2 py-2 rounded-md bg-sky-500 hover:bg-sky-600 text-sm font-medium text-slate-50 transition-colors"
          >
            {mode === "register" ? "Create Admin" : "Login"}
          </button>
        </form>

        <p className="mt-4 text-[11px] text-slate-500">
          After installation, open{" "}
          <span className="font-mono">{panelUrl}</span>{" "}
          in your browser and use <b>First-time Setup</b> to create the admin.
        </p>
      </div>
    </div>
  );
}
