import React, { useEffect, useState } from "react";
import {
  listDomains,
  saveDomain,
  deleteDomain,
  saveSender,
  deleteSender,
  getSettings,
  getSystemIPs,
  importSenders
} from "../api";

export default function Domains() {
  const [domains, setDomains] = useState([]);
  const [settings, setSettings] = useState(null);
  const [systemIPs, setSystemIPs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState("");
  const [editingDomain, setEditingDomain] = useState(null);
  const [showImport, setShowImport] = useState(false);
  const [senderForm, setSenderForm] = useState({
    domainID: null,
    id: 0,
    local_part: "",
    email: "",
    ip: "",
    smtp_password: ""
  });

  const load = async () => {
    setLoading(true);
    setMsg("");
    try {
      const [d, s, ips] = await Promise.all([
        listDomains(),
        getSettings(),
        getSystemIPs()
      ]);
      setDomains(Array.isArray(d) ? d : []);
      setSettings(s || null);
      setSystemIPs(Array.isArray(ips) ? ips : []);
    } catch (err) {
      setMsg(err.message || "Failed to load data");
      setDomains([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const emptyDomain = { id: 0, name: "", mail_host: "", bounce_host: "" };

  const handleEditDomain = (d) => {
    setEditingDomain(d ? { ...d } : { ...emptyDomain });
  };

  const handleSaveDomain = async (e) => {
    e.preventDefault();
    try {
      await saveDomain(editingDomain);
      setEditingDomain(null);
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to save domain");
    }
  };

  const handleDeleteDomain = async (id) => {
    if (!window.confirm("Delete this domain?")) return;
    try {
      await deleteDomain(id);
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to delete domain");
    }
  };

  const handleImport = async (e) => {
    e.preventDefault();
    const file = e.target.file.files[0];
    if (!file) return;
    
    setLoading(true);
    try {
      const res = await importSenders(file);
      setMsg(res.message);
      setShowImport(false);
      await load();
    } catch (err) {
      setMsg("Import Error: " + err.message);
    } finally {
      setLoading(false);
    }
  };

  const startNewSender = (domain) => {
    setSenderForm({
      domainID: domain.id,
      id: 0,
      local_part: "",
      email: "",
      ip: "",
      smtp_password: ""
    });
  };

  const startEditSender = (domain, s) => {
    setSenderForm({
      domainID: domain.id,
      id: s.id,
      local_part: s.local_part || "",
      email: s.email || "",
      ip: s.ip || "",
      smtp_password: s.smtp_password || ""
    });
  };

  const handleSaveSender = async (e) => {
    e.preventDefault();
    if (!senderForm.domainID) return;
    try {
      await saveSender(senderForm.domainID, {
        id: senderForm.id,
        local_part: senderForm.local_part,
        email: senderForm.email,
        ip: senderForm.ip,
        smtp_password: senderForm.smtp_password
      });
      setSenderForm({
        domainID: null,
        id: 0,
        local_part: "",
        email: "",
        ip: "",
        smtp_password: ""
      });
      await load();
      setMsg("Sender saved successfully.");
      setTimeout(() => setMsg(""), 3000);
    } catch (err) {
      setMsg(err.message || "Failed to save sender");
    }
  };

  const handleDeleteSender = async (id) => {
    if (!window.confirm("Delete this sender?")) return;
    try {
      await deleteSender(id);
      await load();
    } catch (err) {
      setMsg(err.message || "Failed to delete sender");
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

  const dnsHelpers = (d) => {
    const mainIp = settings?.main_server_ip || "<SERVER_IP>";

    const ips = new Set();
    if (d.senders && d.senders.length > 0) {
      d.senders.forEach((s) => {
        if (s.ip) ips.add(s.ip);
      });
    }
    ips.add(mainIp);

    const ipParts = Array.from(ips)
      .map((ip) => `ip4:${ip}`)
      .join(" ");
    const spfValue = `v=spf1 ${ipParts} ~all`;

    const root = d.name || "<domain>";
    const mailHost = d.mail_host || `mail.${root}`;
    const bounceHost = d.bounce_host || `bounce.${root}`;

    return [
      {
        label: "A for mail host",
        value: `${mailHost} 3600 IN A ${mainIp}`
      },
      {
        label: "A for bounce host",
        value: `${bounceHost} 3600 IN A ${mainIp}`
      },
      {
        label: "MX to mail host",
        value: `${root} 3600 IN MX 10 ${mailHost}.`
      },
      {
        label: "SPF (Includes Sender IPs)",
        value: `${root} 3600 IN TXT "${spfValue}"`
      }
    ];
  };

  const getSenders = (domain) => {
    return Array.isArray(domain.senders) ? domain.senders : [];
  };

  return (
    <div className="space-y-4 text-sm">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Domains & Senders</h2>
          {settings?.main_server_ip && (
            <div className="text-[11px] text-slate-500">
              Main server IP:{" "}
              <span className="font-mono">{settings.main_server_ip}</span>
            </div>
          )}
        </div>
        <div className="flex gap-2">
            <button
            onClick={() => setShowImport(true)}
            className="px-3 py-1 rounded-md bg-slate-800 hover:bg-slate-700 text-xs"
            >
            ↑ Bulk Import
            </button>
            <button
            onClick={() => handleEditDomain(null)}
            className="px-3 py-1 rounded-md bg-sky-500 hover:bg-sky-600 text-xs"
            >
            + New Domain
            </button>
        </div>
      </div>
      {msg && <div className="text-xs text-slate-300">{msg}</div>}
      
      {loading ? (
        <div className="text-slate-400">Loading...</div>
      ) : domains.length === 0 ? (
        <div className="text-slate-400">No domains yet.</div>
      ) : (
        <div className="space-y-3">
          {domains.map((d) => (
            <div
              key={d.id}
              className="bg-slate-900 border border-slate-800 rounded-lg p-3"
            >
              <div className="flex justify-between items-start mb-2 gap-3">
                <div>
                  <div className="font-medium">{d.name}</div>
                  <div className="text-xs text-slate-400">
                    mail: {d.mail_host || "-"} | bounce: {d.bounce_host || "-"}
                  </div>
                </div>
                <div className="flex gap-2 text-xs">
                  <button
                    onClick={() => handleEditDomain(d)}
                    className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDeleteDomain(d.id)}
                    className="px-2 py-1 rounded bg-red-600/80 hover:bg-red-600"
                  >
                    Delete
                  </button>
                </div>
              </div>

              {/* DNS Helper */}
              <div className="mt-2 mb-2">
                <div className="text-[11px] text-slate-400 mb-1">
                  Quick DNS helpers
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-1">
                  {dnsHelpers(d).map((rec, idx) => (
                    <div
                      key={idx}
                      className="bg-slate-950/60 border border-slate-800 rounded-md px-2 py-1 flex justify-between items-center gap-2"
                    >
                      <div className="text-[11px] text-slate-300 pr-1">
                        <div className="text-slate-400">{rec.label}</div>
                        <div className="font-mono break-all">{rec.value}</div>
                      </div>
                      <button
                        onClick={() => copy(rec.value)}
                        className="flex-shrink-0 px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-[10px]"
                      >
                        Copy
                      </button>
                    </div>
                  ))}
                </div>
              </div>

              {/* Senders */}
              <div className="mt-2">
                <div className="flex items-center justify-between mb-1">
                  <div className="text-xs text-slate-400">Senders</div>
                  <button
                    onClick={() => startNewSender(d)}
                    className="text-[11px] px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
                  >
                    + Add Sender
                  </button>
                </div>
                {getSenders(d).length === 0 ? (
                  <div className="text-xs text-slate-500">
                    No senders for this domain.
                  </div>
                ) : (
                  <div className="space-y-1">
                    {getSenders(d).map((s) => (
                      <div
                        key={s.id}
                        className="flex justify-between items-center text-xs bg-slate-950/50 border border-slate-800 rounded-md px-2 py-1"
                      >
                        <div>
                          <div className="flex items-center gap-2">
                            <span>{s.email || "-"}</span>
                            
                            {/* DKIM Badge */}
                            {s.has_dkim ? (
                                <span className="bg-green-900/50 text-green-300 px-1.5 py-0.5 rounded text-[10px] border border-green-800">
                                    DKIM
                                </span>
                            ) : (
                                <span className="bg-slate-800 text-slate-500 px-1.5 py-0.5 rounded text-[10px]">
                                    No DKIM
                                </span>
                            )}

                            {/* Bounce Badge */}
                            {s.bounce_username ? (
                                <span className="bg-purple-900/50 text-purple-300 px-1.5 py-0.5 rounded text-[10px] border border-purple-800">
                                    Bounce: {s.bounce_username}
                                </span>
                            ) : (
                                <span className="bg-slate-800 text-slate-500 px-1.5 py-0.5 rounded text-[10px]">
                                    No Bounce
                                </span>
                            )}
                          </div>
                          <div className="text-slate-500 mt-0.5">
                            local: {s.local_part || "-"} | IP: {s.ip || "-"}
                          </div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => startEditSender(d, s)}
                            className="px-2 py-1 rounded bg-slate-800 hover:bg-slate-700"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => handleDeleteSender(s.id)}
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
            </div>
          ))}
        </div>
      )}

      {/* Import Modal */}
      {showImport && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-4 w-full max-w-md">
            <h3 className="text-sm font-semibold mb-3">Bulk Import Senders</h3>
            <p className="text-xs text-slate-400 mb-3">
                Upload a <b>CSV file</b> with columns:<br/>
                <code>Domain, LocalPart, IP, Password</code>
            </p>
            <form onSubmit={handleImport} className="space-y-3 text-sm">
                <input type="file" name="file" accept=".csv" className="block w-full text-xs text-slate-300
                  file:mr-4 file:py-2 file:px-4
                  file:rounded-md file:border-0
                  file:text-xs file:font-semibold
                  file:bg-slate-800 file:text-slate-300
                  hover:file:bg-slate-700
                " required />
                
                <div className="flex justify-end gap-2 pt-2 text-xs">
                    <button
                    type="button"
                    onClick={() => setShowImport(false)}
                    className="px-3 py-1 rounded bg-slate-800"
                    >
                    Cancel
                    </button>
                    <button
                    type="submit"
                    className="px-3 py-1 rounded bg-sky-500 hover:bg-sky-600"
                    >
                    Upload & Process
                    </button>
                </div>
            </form>
          </div>
        </div>
      )}

      {/* Domain modal */}
      {editingDomain && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-4 w-full max-w-md">
            <h3 className="text-sm font-semibold mb-3">
              {editingDomain.id ? "Edit Domain" : "New Domain"}
            </h3>
            <form onSubmit={handleSaveDomain} className="space-y-2 text-sm">
              <div>
                <label className="block text-slate-300 mb-1">Domain</label>
                <input
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  value={editingDomain.name || ""}
                  onChange={(e) =>
                    setEditingDomain((d) => ({ ...d, name: e.target.value }))
                  }
                  required
                />
              </div>
              <div>
                <label className="block text-slate-300 mb-1">Mail Host</label>
                <input
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  value={editingDomain.mail_host || ""}
                  onChange={(e) =>
                    setEditingDomain((d) => ({
                      ...d,
                      mail_host: e.target.value
                    }))
                  }
                />
              </div>
              <div>
                <label className="block text-slate-300 mb-1">Bounce Host</label>
                <input
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  value={editingDomain.bounce_host || ""}
                  onChange={(e) =>
                    setEditingDomain((d) => ({
                      ...d,
                      bounce_host: e.target.value
                    }))
                  }
                />
              </div>
              <div className="flex justify-end gap-2 pt-2 text-xs">
                <button
                  type="button"
                  onClick={() => setEditingDomain(null)}
                  className="px-3 py-1 rounded bg-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-3 py-1 rounded bg-sky-500 hover:bg-sky-600"
                >
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Sender modal */}
      {senderForm.domainID && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-4 w-full max-w-md">
            <h3 className="text-sm font-semibold mb-3">
              {senderForm.id ? "Edit Sender" : "New Sender"}
            </h3>
            <form onSubmit={handleSaveSender} className="space-y-2 text-sm">
              <div>
                <label className="block text-slate-300 mb-1">Local Part</label>
                <input
                  value={senderForm.local_part}
                  onChange={(e) =>
                    setSenderForm((s) => ({ ...s, local_part: e.target.value }))
                  }
                  placeholder="editor / info / marketing"
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                />
              </div>
              <div>
                <label className="block text-slate-300 mb-1">Email</label>
                <input
                  value={senderForm.email}
                  onChange={(e) =>
                    setSenderForm((s) => ({ ...s, email: e.target.value }))
                  }
                  placeholder="editor@example.com"
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                />
              </div>
              <div>
                <label className="block text-slate-300 mb-1">IP Address</label>
                {systemIPs.length > 0 ? (
                  <select
                    value={senderForm.ip}
                    onChange={(e) =>
                      setSenderForm((s) => ({ ...s, ip: e.target.value }))
                    }
                    className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  >
                    <option value="">-- Select Server IP --</option>
                    {systemIPs.map((ip) => (
                      <option key={ip} value={ip}>
                        {ip}
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    value={senderForm.ip}
                    onChange={(e) =>
                      setSenderForm((s) => ({ ...s, ip: e.target.value }))
                    }
                    placeholder="No IPs detected, type manually..."
                    className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  />
                )}
                <p className="text-[10px] text-slate-500 mt-1">
                  Select the interface IP this sender will use.
                </p>
              </div>
              <div>
                <label className="block text-slate-300 mb-1">
                  SMTP Password
                </label>
                <input
                  value={senderForm.smtp_password}
                  onChange={(e) =>
                    setSenderForm((s) => ({
                      ...s,
                      smtp_password: e.target.value
                    }))
                  }
                  placeholder="Secret123"
                  className="w-full px-3 py-2 rounded-md bg-slate-950 border border-slate-700 outline-none focus:border-sky-500"
                  type="password"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2 text-xs">
                <button
                  type="button"
                  onClick={() =>
                    setSenderForm({
                      domainID: null,
                      id: 0,
                      local_part: "",
                      email: "",
                      ip: "",
                      smtp_password: ""
                    })
                  }
                  className="px-3 py-1 rounded bg-slate-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-3 py-1 rounded bg-sky-500 hover:bg-sky-600"
                >
                  Save & Auto-Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
