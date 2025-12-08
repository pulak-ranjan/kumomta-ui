import React, { useEffect, useState } from "react";
import { listBounces, saveBounce, deleteBounce, applyBounces } from "../api";

export default function BouncePage() {
  const [list, setList] = useState([]);
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({ id: 0, username: "", password: "", domain: "", notes: "" });

  const load = async () => {
    try {
      const b = await listBounces();
      setList(b);
    } catch (err) {
      setMsg(err.message || "Failed to load bounce accounts");
    }
  };

  useEffect(() => {
    load();
  }, []);

  const onChange = (e) => {
    const { name, value } = e.target;
    setForm((f) => ({ ...f, [name]: value }));
  };

  const startNew = () => {
    setForm({ id: 0, username: "", password: "", domain: "", notes: "" });
  };

  const startEdit = (b) => {
    setForm({ id: b.id, username: b.username, password: "", domain: b.domain, notes: b.notes });
  };

  const onSubmit = async (e) => {
    e.preventDefault();
    setBusy(true);
    setMsg("");
    try {
      await saveBounce(form);
      await load();
      setForm({ id: 0, username: "", password: "", domain: "", notes: "" });
      setMsg("Bounce account saved & applied to system.");
    } catch (err) {
      setMsg(err.message || "Failed to save bounce account");
    } finally {
      setBusy(false);
    }
  };

  const onDelete = async (id) => {
    if (!window.confirm("Delete bounce account?")) return;
    try {
      await deleteBounce(id);
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to delete bounce account");
    }
  };

  const onApplyAll = async () => {
    setBusy(true);
    setMsg("");
    try {
      await applyBounces();
      setMsg("Bounce accounts applied to system (users + Maildir).");
    } catch (err) {
      setMsg(err.message || "Failed to apply bounces");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4 text-sm">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Bounce Accounts</h2>
        <div className="flex gap-2 text-xs">
          <button
            onClick={startNew}
            className="px-3 py-1 rounded-md bg-sky-500 hover:bg-sky-600"
          >
            + New
          </button>
          <button
            onClick={onApplyAll}
            disabled={busy}
            className="px-3 py-1 rounded-md bg-slate-800 hover:bg-slate-700 disabled:opacity-60"
          >
            Apply All
          </button>
        </div>
      </div>
      {msg && <div className="text-xs text-slate-300">{msg}</div>}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* List */}
        <div className="space-y-2">
          <h3 className="text-xs text-slate-400">Existing</h3>
          {list.length === 0 ? (
            <div className="text-xs text-slate-500">No bounce accounts.</div>
          ) : (
            <div className="space-y-2">
              {list.map((b) => (
                <div
                  key={b.id}
                  className="bg-slate-900 border border-slate-800 rounded-md p-2 text-xs flex justify-between"
                >
                  <div>
                    <div className="font-mono">{b.username}</div>
                    {b.domain && (
                      <div className="text-slate-400">
                        Domain: {b.domain}
                      </div>
                    )}
                    {b.notes && (
                      <div className="text-slate-500">
                        {b.notes}
                      </div>
                    )}
                  </div>
                  <div className="flex flex-col gap-1">
                    <button
                      onClick={() => startEdit(b)}
                      className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => onDelete(b.id)}
                      className="px-2 py-1 rounded bg-red-600/80 hover:bg-red-600"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
        {/* Form */}
        <div>
          <h3 className="text-xs text-slate-400 mb-2">
            {form.id ? "Edit Bounce" : "New Bounce"}
          </h3>
          <form onSubmit={onSubmit} className="space-y-2 text-xs">
            <div>
              <label className="block text-slate-300 mb-1">Username</label>
              <input
                name="username"
                value={form.username}
                onChange={onChange}
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                placeholder="bou1-1"
                required
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">
                Password {form.id ? "(leave blank to keep)" : ""}
              </label>
              <input
                type="password"
                name="password"
                value={form.password}
                onChange={onChange}
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Domain (optional)</label>
              <input
                name="domain"
                value={form.domain}
                onChange={onChange}
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              />
            </div>
            <div>
              <label className="block text-slate-300 mb-1">Notes</label>
              <textarea
                name="notes"
                value={form.notes}
                onChange={onChange}
                rows={2}
                className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
              />
            </div>
            <button
              type="submit"
              disabled={busy}
              className="px-3 py-2 rounded-md bg-sky-500 hover:bg-sky-600"
            >
              {busy ? "Saving..." : "Save & Apply"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
