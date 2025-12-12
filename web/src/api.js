const API_BASE = "/api";

function getToken() {
  return localStorage.getItem("kumoui_token") || "";
}

export async function apiRequest(path, options = {}) {
  const { method = "GET", body, auth = true } = options;
  const headers = { "Content-Type": "application/json" };

  if (auth) {
    const token = getToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined
  });

  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { raw: text };
  }

  if (!res.ok) {
    const msg = data.error || res.statusText || "Request failed";
    throw new Error(msg);
  }

  return data;
}

// Auth
export function registerAdmin(email, password) {
  return apiRequest("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false
  });
}

export function login(email, password) {
  return apiRequest("/auth/login", {
    method: "POST",
    body: { email, password },
    auth: false
  });
}

export function me() {
  return apiRequest("/auth/me");
}

// System
export function getSystemIPs() {
  return apiRequest("/system/ips");
}

export function addSystemIPs(cidr, list) {
  return apiRequest("/system/ips", {
    method: "POST",
    body: { cidr, list }
  });
}

// NEW: Configure IP on Server (Needed for IPsPage)
export function configureSystemIP(ip, netmask, iface) {
  return apiRequest("/system/ips/configure", {
    method: "POST",
    body: { ip, netmask, interface: iface }
  });
}

// Status & Dashboard
export function getStatus() {
  return apiRequest("/status", { auth: false });
}

export function getDashboardStats() {
  return apiRequest("/dashboard/stats");
}

export function getAIInsights() {
  return apiRequest("/system/ai-analyze", { 
    method: "POST", 
    body: { type: "logs" } 
  });
}

// --- NEW: Chat Agent (Updated) ---
export function sendAIChat(payload) {
  // payload is { messages: [], new_msg: "..." }
  return apiRequest("/ai/chat", {
    method: "POST",
    body: payload
  });
}

// Settings
export function getSettings() {
  return apiRequest("/settings");
}

export function saveSettings(payload) {
  return apiRequest("/settings", { method: "POST", body: payload });
}

// Domains & senders
export function listDomains() {
  return apiRequest("/domains");
}

export function saveDomain(domain) {
  const method = domain.id ? "PUT" : "POST";
  const url = domain.id ? `/domains/${domain.id}` : "/domains";
  return apiRequest(url, { method, body: domain });
}

export function deleteDomain(id) {
  return apiRequest(`/domains/${id}`, { method: "DELETE" });
}

export function listSenders(domainID) {
  return apiRequest(`/domains/${domainID}/senders`);
}

export function saveSender(domainID, sender) {
  if (sender.id) {
    return apiRequest(`/senders/${sender.id}`, { method: "PUT", body: sender });
  }
  return apiRequest(`/domains/${domainID}/senders`, {
    method: "POST",
    body: sender
  });
}

export function deleteSender(id) {
  return apiRequest(`/senders/${id}`, { method: "DELETE" });
}

// Config
export function previewConfig() {
  return apiRequest("/config/preview");
}

export function applyConfig() {
  return apiRequest("/config/apply", { method: "POST" });
}

// DKIM
export function listDKIMRecords() {
  return apiRequest("/dkim/records");
}

export function generateDKIM(domain, localPart) {
  return apiRequest("/dkim/generate", {
    method: "POST",
    body: { domain, local_part: localPart }
  });
}

export function importSenders(file) {
  const formData = new FormData();
  formData.append("file", file);
  return fetch(`${API_BASE}/import/csv`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${getToken()}`
    },
    body: formData
  }).then(async (res) => {
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || "Import failed");
    return data;
  });
}

// Bounce
export function listBounces() {
  return apiRequest("/bounces");
}

export function saveBounce(b) {
  return apiRequest("/bounces", { method: "POST", body: b });
}

export function deleteBounce(id) {
  return apiRequest(`/bounces/${id}`, { method: "DELETE" });
}

export function applyBounces() {
  return apiRequest("/bounces/apply", { method: "POST" });
}

// Logs
export function getLogs(service, lines = 100) {
  return apiRequest(`/logs/${service}?lines=${lines}`);
}

// --- Tools & Security ---
export function sendTestEmail(payload) {
  return apiRequest("/tools/send-test", {
    method: "POST",
    body: payload
  });
}

export function blockIP(ip) {
  return apiRequest("/system/action/block-ip", {
    method: "POST",
    body: { ip }
  });
}

export function checkBlacklist() {
  return apiRequest("/system/check-blacklist", { method: "POST" });
}

export function checkSecurity() {
  return apiRequest("/system/check-security", { method: "POST" });
}

// --- Warmup ---
export function getWarmupList() {
  return apiRequest("/warmup");
}

export function updateWarmup(senderID, enabled, plan) {
  return apiRequest(`/warmup/${senderID}`, {
    method: "POST",
    body: { enabled, plan }
  });
}

// --- API Keys ---
export function listKeys() {
  return apiRequest("/keys");
}

export function createKey(name, scopes) {
  return apiRequest("/keys", {
    method: "POST",
    body: { name, scopes }
  });
}

export function deleteKey(id) {
  return apiRequest(`/keys/${id}`, { method: "DELETE" });
}
