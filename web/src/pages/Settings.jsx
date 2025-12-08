import React, { useEffect, useState } from "react";
import { getSettings, saveSettings } from "../api";

export default function Settings() {
  const [form, setForm] = useState({
    main_hostname: "",
    main_server_ip: "",
    relay_ips: "",
    ai_provider: "",
    ai_api_key: ""
  });
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const s = await getSettings();
        setForm((f) => ({ ...f, ...s }));
      } catch (err) {
        setMsg(err.message || "Failed to load settings");
      }
    })();
  }, []);

  const onChange = (e) => {
    const { name, value } = e.target;
    setForm((f) => ({ ...f, [name]: value }));
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setMsg("");
    try {
      await saveSettings(form);
      setMsg("Settings saved.");
      setForm((f) => ({ ...f, ai_api_key: "" })); // clear input
    } catch (err) {
      setMsg(err.message || "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4 text-sm">
      <h2 className="text-lg font-semibold">Settings</h2>
      <form onSubmit={onSubmit} className="space-y-3 max-w-xl">
        <div>
          <label className="block text-slate-300 mb-1">Main Hostname</label>
          <input
            name="main_hostname"
            value={form.main_hostname}
            onChange={onChange}
            className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
            placeholder="mta.example.com"
          />
        </div>
        <div>
          <label className="block text-slate-300 mb-1">Main Server IP</label>
          <input
            name="main_server_ip"
            value={form.main_server_ip}
            onChange={onChange}
            className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
            placeholder="1.2.3.4"
          />
        </div>
        <div>
          <label className="block text-slate-300 mb-1">Relay IPs (CSV)</label>
          <input
            name="relay_ips"
            value={form.relay_ips}
            onChange={onChange}
            className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
            placeholder="127.0.0.1,10.0.0.5"
          />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <label className="block text-slate-300 mb-1">AI Provider</label>
            <input
              name="ai_provider"
              value={form.ai_provider}
              onChange={onChange}
              className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              placeholder="openai / deepseek"
            />
          </div>
          <div>
            <label className="block text-slate-300 mb-1">AI API Key</label>
            <input
              name="ai_api_key"
              type="password"
              value={form.ai_api_key}
              onChange={onChange}
              className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
            />
            <p className="text-[11px] text-slate-500 mt-1">
              Key is write-only, never returned by API.
            </p>
          </div>
        </div>
        {msg && <div className="text-xs text-slate-300">{msg}</div>}
        <button
          type="submit"
          disabled={saving}
          className="px-4 py-2 rounded-md bg-sky-500 hover:bg-sky-600 text-sm font-medium text-slate-50 disabled:opacity-60"
        >
          {saving ? "Saving..." : "Save Settings"}
        </button>
      </form>
    </div>
  );
}
