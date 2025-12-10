import React, { useState } from "react";
import { 
  Send, ShieldAlert, Search, Loader2, CheckCircle2, AlertTriangle, Terminal, Ban 
} from "lucide-react";
import { sendTestEmail, blockIP, checkBlacklist, checkSecurity } from "../api";
import { cn } from "../lib/utils";

export default function ToolsPage() {
  const [activeTab, setActiveTab] = useState("test-email");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">System Tools</h1>
        <p className="text-muted-foreground">Utilities for testing and securing your MTA.</p>
      </div>

      <div className="flex gap-2 border-b">
        <TabButton id="test-email" label="Test Lab" icon={Send} active={activeTab} onClick={setActiveTab} />
        <TabButton id="security" label="Guardian" icon={ShieldAlert} active={activeTab} onClick={setActiveTab} />
      </div>

      <div className="pt-4">
        {activeTab === "test-email" && <TestEmailPanel />}
        {activeTab === "security" && <SecurityPanel />}
      </div>
    </div>
  );
}

function TabButton({ id, label, icon: Icon, active, onClick }) {
  return (
    <button
      onClick={() => onClick(id)}
      className={cn(
        "flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors",
        active === id 
          ? "border-primary text-primary" 
          : "border-transparent text-muted-foreground hover:text-foreground hover:border-muted"
      )}
    >
      <Icon className="w-4 h-4" />
      {label}
    </button>
  );
}

function TestEmailPanel() {
  const [form, setForm] = useState({
    sender: "",
    recipient: "",
    subject: "Test from KumoMTA UI",
    body: "This is a test email sent via the KumoMTA Control Panel.\n\nTime: " + new Date().toLocaleString()
  });
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setResult(null);
    setError("");
    try {
      const res = await sendTestEmail(form);
      setResult(res);
    } catch (err) { setError(err.message); } 
    finally { setLoading(false); }
  };

  return (
    <div className="grid lg:grid-cols-2 gap-6">
      <div className="bg-card border rounded-xl p-6 shadow-sm">
        <h3 className="font-semibold mb-4 flex items-center gap-2">
          <Send className="w-4 h-4 text-blue-500" /> Compose Test
        </h3>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Sender Identity</label>
            <input 
              value={form.sender}
              onChange={e => setForm({...form, sender: e.target.value})}
              placeholder="editor@yourdomain.com"
              className="w-full h-10 px-3 rounded-md border bg-background text-sm"
              required
            />
            <p className="text-[10px] text-muted-foreground">Must match a configured sender in Domains.</p>
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Recipient</label>
            <input 
              value={form.recipient}
              onChange={e => setForm({...form, recipient: e.target.value})}
              placeholder="mail-tester@..."
              className="w-full h-10 px-3 rounded-md border bg-background text-sm"
              type="email"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Subject</label>
            <input 
              value={form.subject}
              onChange={e => setForm({...form, subject: e.target.value})}
              className="w-full h-10 px-3 rounded-md border bg-background text-sm"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Body</label>
            <textarea 
              value={form.body}
              onChange={e => setForm({...form, body: e.target.value})}
              className="w-full h-32 p-3 rounded-md border bg-background text-sm font-mono"
              required
            />
          </div>
          <button type="submit" disabled={loading} className="w-full h-10 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 flex items-center justify-center gap-2">
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />} Send Message
          </button>
        </form>
      </div>

      <div className="space-y-6">
        {result && (
          <div className="bg-green-500/10 border border-green-500/20 rounded-xl p-4">
            <div className="flex items-center gap-2 text-green-600 font-semibold mb-2">
              <CheckCircle2 className="w-5 h-5" /> Message Sent
            </div>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between border-b border-green-500/10 pb-2">
                <span className="text-muted-foreground">Source IP:</span>
                <span className="font-mono font-bold">{result.sender_ip}</span>
              </div>
              <div className="flex justify-between border-b border-green-500/10 pb-2">
                <span className="text-muted-foreground">HELO:</span>
                <span className="font-mono">{result.used_helo}</span>
              </div>
              <div className="mt-2">
                <span className="text-muted-foreground block mb-1">SMTP Output:</span>
                <pre className="bg-black/5 dark:bg-black/30 p-2 rounded overflow-x-auto font-mono text-[10px]">{result.smtp_output}</pre>
              </div>
            </div>
          </div>
        )}
        {error && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-xl p-4 flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-destructive shrink-0" />
            <div className="text-sm text-destructive font-medium break-all">{error}</div>
          </div>
        )}
        {!result && !error && (
          <div className="bg-card border rounded-xl p-8 flex flex-col items-center justify-center text-muted-foreground text-sm border-dashed">
            <Terminal className="w-12 h-12 mb-2 opacity-20" />
            <p>Ready to test delivery.</p>
          </div>
        )}
      </div>
    </div>
  );
}

function SecurityPanel() {
  const [ip, setIp] = useState("");
  const [loading, setLoading] = useState("");
  const [msg, setMsg] = useState({ text: "", type: "" });

  const handleBlock = async (e) => {
    e.preventDefault();
    if (!ip) return;
    setLoading("block");
    try {
      await blockIP(ip);
      setMsg({ text: `Blocked IP ${ip} successfully`, type: "success" });
      setIp("");
    } catch (err) { setMsg({ text: err.message, type: "error" }); }
    setLoading("");
  };

  const handleAudit = async () => {
    setLoading("audit");
    try {
      await checkSecurity();
      setMsg({ text: "Audit started. Check Webhook.", type: "success" });
    } catch (err) { setMsg({ text: err.message, type: "error" }); }
    setLoading("");
  };

  const handleBlacklist = async () => {
    setLoading("blacklist");
    try {
      await checkBlacklist();
      setMsg({ text: "Scan started. Check Webhook.", type: "success" });
    } catch (err) { setMsg({ text: err.message, type: "error" }); }
    setLoading("");
  };

  return (
    <div className="grid lg:grid-cols-2 gap-6">
      <div className="bg-card border rounded-xl p-6 shadow-sm">
        <h3 className="font-semibold mb-4 flex items-center gap-2 text-destructive">
          <Ban className="w-4 h-4" /> Emergency IP Block
        </h3>
        <form onSubmit={handleBlock} className="flex gap-2">
          <input value={ip} onChange={e => setIp(e.target.value)} placeholder="1.2.3.4" className="flex-1 h-10 px-3 rounded-md border bg-background text-sm" />
          <button type="submit" disabled={!!loading} className="bg-destructive text-destructive-foreground px-4 h-10 rounded-md text-sm font-medium hover:bg-destructive/90">
            {loading === "block" ? <Loader2 className="w-4 h-4 animate-spin" /> : "Block"}
          </button>
        </form>
      </div>
      <div className="space-y-4">
        {msg.text && (
          <div className={cn("p-3 rounded-md text-sm font-medium", msg.type === "error" ? "bg-destructive/10 text-destructive" : "bg-green-500/10 text-green-600")}>{msg.text}</div>
        )}
        <div className="bg-card border rounded-xl p-4 flex items-center justify-between">
          <div className="font-medium text-sm">Force Security Audit</div>
          <button onClick={handleAudit} disabled={!!loading} className="bg-secondary hover:bg-secondary/80 text-secondary-foreground px-3 py-1.5 rounded text-xs font-medium">
            {loading === "audit" ? "Scanning..." : "Run Audit"}
          </button>
        </div>
        <div className="bg-card border rounded-xl p-4 flex items-center justify-between">
          <div className="font-medium text-sm">Check Blacklists</div>
          <button onClick={handleBlacklist} disabled={!!loading} className="bg-secondary hover:bg-secondary/80 text-secondary-foreground px-3 py-1.5 rounded text-xs font-medium">
            {loading === "blacklist" ? "Checking..." : "Check Now"}
          </button>
        </div>
      </div>
    </div>
  );
}
