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

// Status & settings
export function getStatus() {
  return apiRequest("/status", { auth: false });
}

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
  return apiRequest("/domains", { method: "POST", body: domain });
}

export function deleteDomain(id) {
  return apiRequest(`/domains/${id}`, { method: "DELETE" });
}

export function listSenders(domainID) {
  return apiRequest(`/domains/${domainID}/senders`);
}

export function saveSender(domainID, sender) {
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
