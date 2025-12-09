import React, { useEffect, useState } from "react";
import { 
  Save, 
  Server, 
  Globe, 
  Network, 
  Bot, 
  Key, 
  Loader2 
} from "lucide-react";
import { getSettings, saveSettings } from "../api";
import { cn } from "../lib/utils";

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
        setMsg("Failed to load settings");
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
      setMsg("Settings saved successfully.");
      setForm((f) => ({ ...f, ai_api_key: "" })); // clear sensitive field
    } catch (err) {
      setMsg(err.message || "Failed to save settings");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-8 py-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">System Settings</h1>
        <p className="text-muted-foreground">Configure global server parameters and integrations.</p>
      </div>

      <form onSubmit={onSubmit} className="space-y-8">
        
        {/* Server Config Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Server className="w-5 h-5" /> Server Configuration
          </h3>
          
          <div className="grid gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Main Hostname</label>
              <div className="relative">
                <Globe className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="main_hostname"
                  value={form.main_hostname}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="mta.example.com"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Main Server IP</label>
              <div className="relative">
                <Network className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="main_server_ip"
                  value={form.main_server_ip}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="1.2.3.4"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Relay IPs (CSV)</label>
              <input
                name="relay_ips"
                value={form.relay_ips}
                onChange={onChange}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                placeholder="127.0.0.1, 10.0.0.5"
              />
              <p className="text-[10px] text-muted-foreground">IPs allowed to relay through this MTA.</p>
            </div>
          </div>
        </div>

        {/* AI Integration Section */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold flex items-center gap-2 border-b pb-2">
            <Bot className="w-5 h-5" /> AI Integration
          </h3>
          
          <div className="grid sm:grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">AI Provider</label>
              <select
                name="ai_provider"
                value={form.ai_provider}
                onChange={onChange}
                className="w-full h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
              >
                <option value="">Select Provider</option>
                <option value="openai">OpenAI (GPT-3.5)</option>
                <option value="deepseek">DeepSeek</option>
              </select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">API Key</label>
              <div className="relative">
                <Key className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  name="ai_api_key"
                  type="password"
                  value={form.ai_api_key}
                  onChange={onChange}
                  className="w-full pl-9 h-10 rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-ring"
                  placeholder="sk-..."
                />
              </div>
              <p className="text-[10px] text-muted-foreground">Key is write-only, never shown.</p>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="flex items-center justify-between pt-4">
          <div className={cn("text-sm font-medium", msg.includes("Failed") ? "text-destructive" : "text-green-600")}>
            {msg}
          </div>
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-2 bg-primary text-primary-foreground hover:bg-primary/90 px-6 py-2 rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
            {saving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
