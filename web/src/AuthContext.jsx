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
    localStorage.setItem("kumoui_token", res.token);
    setToken(res.token);
    setUser({ email: res.email });
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
      value={{ token, user, loading, login: handleLogin, register: handleRegister, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
