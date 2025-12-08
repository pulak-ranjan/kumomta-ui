import React, { useState } from "react";
import { useAuth } from "../AuthContext";

export default function LoginRegister() {
  const { login, register } = useAuth();
  const [mode, setMode] = useState("login"); // or "register"
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

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
    } catch (err) {
      setError(err.message || "Failed");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-xl p-6 shadow-xl">
        <h1 className="text-xl font-semibold mb-2">
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
            className={`flex-1 py-1 rounded-md ${
              mode === "login"
                ? "bg-sky-500 text-slate-50"
                : "bg-slate-800 text-slate-200"
            }`}
          >
            Login
          </button>
          <button
            onClick={() => setMode("register")}
            className={`flex-1 py-1 rounded-md ${
              mode === "register"
                ? "bg-sky-500 text-slate-50"
                : "bg-slate-800 text-slate-200"
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
              className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-sm outline-none focus:border-sky-500"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="block text-slate-300 mb-1">Password</label>
            <input
              type="password"
              className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 text-slate-50 text-sm outline-none focus:border-sky-500"
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
            className="w-full mt-2 py-2 rounded-md bg-sky-500 hover:bg-sky-600 text-sm font-medium text-slate-50"
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
