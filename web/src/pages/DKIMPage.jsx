import React, { useEffect, useState } from "react";
import { listDKIMRecords, generateDKIM } from "../api";

export default function DKIMPage() {
  const [records, setRecords] = useState([]);
  const [msg, setMsg] = useState("");
  const [form, setForm] = useState({ domain: "", local_part: "" });
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      const recs = await listDKIMRecords();
      setRecords(recs);
    } catch (err) {
      setMsg(err.message || "Failed to load DKIM records");
    }
  };

  useEffect(() => {
    load();
  }, []);

  const onChange = (e) => {
    const { name, value } = e.target;
    setForm((f) => ({ ...f, [name]: value }));
  };

  const onGenerate = async (e) => {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      await generateDKIM(form.domain, form.local_part || undefined);
      setMsg("DKIM keys generated. Refreshing records...");
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to generate DKIM");
    } finally {
      setBusy(false);
    }
  };

  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      setMsg("Copied to clipboard");
      setTimeout(() => setMsg(""), 2000);
    } catch {
      setMsg("Failed to copy");
      setTimeout(() => setMsg(""), 2000);
    }
  };

  return (
    <div className="space-y-4 text-sm">
      <h2 className="text-lg font-semibold">DKIM</h2>
      <form onSubmit={onGenerate} className="flex flex-wrap gap-2 items-end">
        <div>
          <label className="block text-slate-300 mb-1">Domain</label>
          <input
            name="domain"
            value={form.domain}
            onChange={onChange}
            className="px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500 text-sm"
            placeholder="example.com"
            required
          />
        </div>
        <div>
          <label className="block text-slate-300 mb-1">Local Part (optional)</label>
          <input
            name="local_part"
            value={form.local_part}
            onChange={onChange}
            className="px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500 text-sm"
            placeholder="editor (or leave blank = all)"
          />
        </div>
        <button
          type="submit"
          disabled={busy}
          className="px-3 py-2 rounded-md bg-sky-500 hover:bg-sky-600 text-xs"
        >
          {busy ? "Working..." : "Generate DKIM"}
        </button>
      </form>
      {msg && <div className="text-xs text-slate-300">{msg}</div>}
      <div className="space-y-2">
        <h3 className="text-sm font-semibold">DNS Records</h3>
        {records.length === 0 ? (
          <div className="text-xs text-slate-500">No DKIM records found.</div>
        ) : (
          <div className="space-y-2">
            {records.map((r, idx) => {
              const fullLine = `${r.dns_name} 3600 IN TXT "${r.dns_value}"`;
              return (
                <div
                  key={idx}
                  className="bg-slate-900 border border-slate-800 rounded-md p-2 text-[11px]"
                >
                  <div className="flex justify-between items-center mb-1">
                    <div className="text-slate-300">
                      {r.domain} ({r.selector})
                    </div>
                    <div className="flex gap-1">
                      <button
                        onClick={() => copy(r.dns_name)}
                        className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-[10px]"
                      >
                        Copy name
                      </button>
                      <button
                        onClick={() => copy(r.dns_value)}
                        className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-[10px]"
                      >
                        Copy value
                      </button>
                      <button
                        onClick={() => copy(fullLine)}
                        className="px-2 py-1 rounded bg-sky-500 hover:bg-sky-600 text-[10px]"
                      >
                        Copy full TXT
                      </button>
                    </div>
                  </div>
                  <div className="text-slate-400">
                    Name: <code>{r.dns_name}</code>
                  </div>
                  <div className="text-slate-400">
                    Value:{" "}
                    <code className="break-all">{r.dns_value}</code>
                  </div>
                  <div className="mt-1 text-slate-500">
                    Full TXT:{" "}
                    <code className="break-all">{fullLine}</code>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
