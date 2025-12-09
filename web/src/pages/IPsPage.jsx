import React, { useEffect, useState } from "react";
import { getSystemIPs, addSystemIPs } from "../api";

export default function IPsPage() {
  const [ips, setIPs] = useState([]);
  const [form, setForm] = useState({ cidr: "", list: "" });
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      const list = await getSystemIPs();
      setIPs(list || []);
    } catch (err) {
      setMsg("Failed to load IPs");
    }
  };

  useEffect(() => {
    load();
  }, []);

  const onAdd = async (e) => {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      const res = await addSystemIPs(form.cidr, form.list);
      setMsg(`Added ${res.added || 0} IPs.`);
      setForm({ cidr: "", list: "" });
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to add IPs");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 text-sm">
      <h2 className="text-lg font-semibold">IP Management</h2>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Add IPs Form */}
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-xs uppercase text-slate-400 mb-3">
            Add IPs to Inventory
          </h3>
          <form onSubmit={onAdd} className="space-y-3">
            <div>
              <label className="block text-slate-300 mb-1">
                Add by Range (CIDR)
              </label>
              <input
                value={form.cidr}
                onChange={(e) => setForm({ ...form, cidr: e.target.value })}
                placeholder="192.168.1.0/24"
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">
                Add by List (One per line)
              </label>
              <textarea
                value={form.list}
                onChange={(e) => setForm({ ...form, list: e.target.value })}
                rows={5}
                placeholder={"10.0.0.1\n10.0.0.2"}
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              />
            </div>
            {msg && <div className="text-xs text-green-400">{msg}</div>}
            <button
              type="submit"
              disabled={busy}
              className="px-3 py-2 rounded-md bg-sky-500 hover:bg-sky-600 disabled:opacity-60"
            >
              {busy ? "Adding..." : "Add IPs"}
            </button>
          </form>
        </div>

        {/* List IPs */}
        <div className="bg-slate-900 border border-slate-800 rounded-lg p-4">
          <h3 className="text-xs uppercase text-slate-400 mb-3">
            Available IPs ({ips.length})
          </h3>
          <div className="overflow-auto max-h-[300px] space-y-1">
            {ips.map((ip, i) => (
              <div
                key={i}
                className="bg-slate-950 px-2 py-1 rounded text-xs font-mono text-slate-300 mb-1 flex justify-between items-center"
              >
                <span>{ip.value}</span>
                <span className="text-slate-500 text-[10px]">{ip.interface}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
