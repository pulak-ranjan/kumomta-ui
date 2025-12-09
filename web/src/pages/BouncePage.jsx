import React, { useEffect, useState } from "react";
import { listBounces, listDomains } from "../api";

export default function BouncePage() {
  const [list, setList] = useState([]);
  const [domains, setDomains] = useState([]);
  const [filter, setFilter] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    load();
  }, []);

  const load = async () => {
    try {
      const [b, d] = await Promise.all([listBounces(), listDomains()]);
      setList(b);
      setDomains(d);
    } catch (err) {
      setMsg("Failed to load");
    }
  };

  const filteredList = filter 
    ? list.filter(b => b.domain === filter)
    : list;

  return (
    <div className="space-y-4 text-sm">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold">Bounce Accounts</h2>
        
        <select 
          className="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-xs outline-none"
          value={filter}
          onChange={e => setFilter(e.target.value)}
        >
          <option value="">All Domains</option>
          {domains.map(d => <option key={d.id} value={d.name}>{d.name}</option>)}
        </select>
      </div>

      <div className="grid gap-2">
        {filteredList.map(b => (
          <div key={b.id} className="bg-slate-900 border border-slate-800 p-2 rounded flex justify-between items-center">
            <div>
              <div className="font-mono text-sky-400">{b.username}</div>
              <div className="text-xs text-slate-500">{b.domain}</div>
            </div>
            <div className="text-xs text-slate-600 italic">
              {b.notes || "No notes"}
            </div>
          </div>
        ))}
        {filteredList.length === 0 && <div className="text-slate-500">No accounts found.</div>}
      </div>
    </div>
  );
}
