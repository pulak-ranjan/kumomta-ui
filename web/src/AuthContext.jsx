import React, { createContext, useContext, useEffect, useState } from "react";
import { login as apiLogin, me as apiMe, registerAdmin } from "./api";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(localStorage.getItem("kumoui_token") || "");
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(!!token);

  useEffect(() => {
    if (!token) {
      setUser(null);
      setLoading(false);
      return;
    }
    (async () => {
      try {
        const u = await apiMe();
        setUser(u);
      } catch {
        localStorage.removeItem("kumoui_token");
        setToken("");
        setUser(null);
      } finally {
        setLoading(false);
      }
    })();
  }, [token]);

  const handleLogin = async (email, password) => {
    const res = await apiLogin(email, password);
    
    // FIX: Check if 2FA is required before setting token
    if (res.requires_2fa) {
        return res; // Return early, let component handle 2FA step
    }

    // Normal login success
    localStorage.setItem("kumoui_token", res.token);
    setToken(res.token);
    setUser({ email: res.email });
    return res;
  };

  // FIX: New helper for 2FA verification step
  const verify2FA = async (tempToken, code) => {
    const res = await fetch('/api/auth/verify-2fa', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-Temp-Token': tempToken
        },
        body: JSON.stringify({ code })
    });
    
    const data = await res.json();
    if (!res.ok) {
        throw new Error(data.error || 'Verification failed');
    }

    localStorage.setItem("kumoui_token", data.token);
    setToken(data.token);
    setUser({ email: data.email });
    return data;
  };

  const handleRegister = async (email, password) => {
    const res = await registerAdmin(email, password);
    localStorage.setItem("kumoui_token", res.token);
    setToken(res.token);
    setUser({ email: res.email });
  };

  const logout = () => {
    localStorage.removeItem("kumoui_token");
    setToken("");
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{ 
        token, 
        user, 
        loading, 
        login: handleLogin, 
        verify2FA, // Export new function
        register: handleRegister, 
        logout 
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
