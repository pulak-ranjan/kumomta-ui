import React, { createContext, useContext, useState, useEffect } from 'react';

const ThemeContext = createContext();

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem('theme');
    if (saved) return saved;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  });

  useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
      root.classList.remove('light');
    } else if (theme === 'light') {
      root.classList.remove('dark');
      root.classList.add('light');
    } else {
      // System
      if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
        root.classList.add('dark');
        root.classList.remove('light');
      } else {
        root.classList.remove('dark');
        root.classList.add('light');
      }
    }
    localStorage.setItem('theme', theme);
  }, [theme]);

  const saveThemeToServer = async (newTheme) => {
    const token = localStorage.getItem('token');
    if (!token) return;
    try {
      await fetch('/api/auth/theme', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ theme: newTheme })
      });
    } catch (e) { console.error(e); }
  };

  const changeTheme = (newTheme) => {
    setTheme(newTheme);
    saveThemeToServer(newTheme);
  };

  return (
    <ThemeContext.Provider value={{ theme, setTheme: changeTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}

export function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  const options = [
    { value: 'light', icon: '☀️', label: 'Light' },
    { value: 'dark', icon: '🌙', label: 'Dark' },
    { value: 'system', icon: '💻', label: 'System' },
  ];

  return (
    <div className="flex items-center gap-2 bg-gray-700 dark:bg-gray-800 rounded-lg p-1">
      {options.map(opt => (
        <button
          key={opt.value}
          onClick={() => setTheme(opt.value)}
          className={`px-3 py-1 rounded text-sm transition-colors ${
            theme === opt.value
              ? 'bg-blue-600 text-white'
              : 'text-gray-400 hover:text-white'
          }`}
          title={opt.label}
        >
          {opt.icon}
        </button>
      ))}
    </div>
  );
}

// Compact toggle for header/navbar
export function ThemeToggleCompact() {
  const { theme, setTheme } = useTheme();

  const cycleTheme = () => {
    const order = ['light', 'dark', 'system'];
    const idx = order.indexOf(theme);
    setTheme(order[(idx + 1) % 3]);
  };

  const icon = theme === 'light' ? '☀️' : theme === 'dark' ? '🌙' : '💻';

  return (
    <button
      onClick={cycleTheme}
      className="p-2 rounded-lg bg-gray-700 hover:bg-gray-600 transition-colors"
      title={`Theme: ${theme}`}
    >
      {icon}
    </button>
  );
}
