import React, { useState, useRef, useEffect } from "react";
import { Bot, Send, X, MessageSquare, Loader2, RefreshCw } from "lucide-react";
import ReactMarkdown from "react-markdown";
import { sendAIChat } from "../api"; // Assuming you update api.js with new logic or keep using it
import { cn } from "../lib/utils";

// Make sure you update your api.js to include getChatHistory if not using raw fetch
const getToken = () => localStorage.getItem("kumoui_token") || "";

export default function AIAssistant() {
  const [isOpen, setIsOpen] = useState(false);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(false);
  const scrollRef = useRef(null);

  useEffect(() => {
    if (isOpen) {
      loadHistory();
    }
  }, [isOpen]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isOpen]);

  const loadHistory = async () => {
    try {
      const res = await fetch("/api/ai/history", {
        headers: { Authorization: `Bearer ${getToken()}` }
      });
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data) && data.length > 0) {
          setMessages(data);
        } else if (messages.length === 0) {
          setMessages([{ role: "assistant", content: "Hello! I am the KumoMTA Guardian. I can secure logs, configure listeners, and manage blocks. How can I assist?" }]);
        }
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const newMsg = { role: "user", content: input };
    setMessages(prev => [...prev, newMsg]);
    setInput("");
    setLoading(true);

    try {
      const res = await sendAIChat({ messages: messages, new_msg: input });
      setMessages(prev => [...prev, { role: "assistant", content: res.reply }]);
    } catch (err) {
      setMessages(prev => [...prev, { role: "assistant", content: "Error: " + err.message }]);
    } finally {
      setLoading(false);
    }
  };

  const clearHistory = () => {
    // Ideally add an API endpoint to clear history, for now just clear UI
    setMessages([{ role: "assistant", content: "History cleared locally." }]);
  };

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col items-end gap-4">
      {isOpen && (
        <div className="w-[380px] h-[550px] bg-card border border-border rounded-xl shadow-2xl flex flex-col animate-in slide-in-from-bottom-5 fade-in duration-200 overflow-hidden">
          {/* Header */}
          <div className="p-3 border-b bg-primary/5 flex justify-between items-center">
            <div className="flex items-center gap-2">
              <div className="p-1.5 bg-primary/10 rounded-md text-primary">
                <Bot className="w-4 h-4" />
              </div>
              <span className="font-semibold text-sm">Kumo Guardian</span>
            </div>
            <div className="flex gap-1">
              <button onClick={loadHistory} className="hover:bg-muted p-1 rounded transition-colors" title="Reload History">
                <RefreshCw className="w-4 h-4" />
              </button>
              <button onClick={() => setIsOpen(false)} className="hover:bg-muted p-1 rounded transition-colors">
                <X className="w-4 h-4" />
              </button>
            </div>
          </div>

          {/* Messages */}
          <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 space-y-4 bg-muted/10">
            {messages.map((m, i) => (
              <div key={i} className={cn("flex gap-3 text-sm", m.role === "user" ? "justify-end" : "justify-start")}>
                {m.role === "assistant" && (
                  <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0 mt-0.5">
                    <Bot className="w-3 h-3 text-primary" />
                  </div>
                )}
                <div className={cn(
                  "p-3 rounded-lg max-w-[85%] shadow-sm",
                  m.role === "user" 
                    ? "bg-primary text-primary-foreground rounded-tr-none" 
                    : "bg-card border text-card-foreground rounded-tl-none"
                )}>
                  <ReactMarkdown 
                    className="prose dark:prose-invert prose-sm max-w-none break-words"
                    components={{
                      code: ({node, inline, className, children, ...props}) => (
                        <code className={cn("bg-black/20 rounded px-1", className)} {...props}>
                          {children}
                        </code>
                      ),
                      pre: ({node, children, ...props}) => (
                        <pre className="bg-black/20 p-2 rounded overflow-x-auto text-xs my-2" {...props}>
                          {children}
                        </pre>
                      )
                    }}
                  >
                    {m.content}
                  </ReactMarkdown>
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex gap-3 text-sm">
                <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                  <Loader2 className="w-3 h-3 text-primary animate-spin" />
                </div>
                <div className="text-muted-foreground text-xs py-2 italic">Analyzing system...</div>
              </div>
            )}
          </div>

          {/* Input */}
          <form onSubmit={handleSubmit} className="p-3 border-t bg-card">
            <div className="relative flex items-center">
              <input
                value={input}
                onChange={e => setInput(e.target.value)}
                placeholder="Ask to block IP, check logs, etc..."
                className="w-full bg-muted/50 border-0 rounded-md h-10 pl-3 pr-10 text-sm focus:ring-1 focus:ring-primary"
              />
              <button 
                type="submit" 
                disabled={loading || !input}
                className="absolute right-1 p-1.5 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 transition-colors"
              >
                <Send className="w-3 h-3" />
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Toggle Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          "h-12 w-12 rounded-full shadow-lg flex items-center justify-center transition-all duration-300 hover:scale-105",
          isOpen ? "bg-muted text-muted-foreground rotate-90" : "bg-primary text-primary-foreground"
        )}
      >
        {isOpen ? <X className="w-6 h-6" /> : <MessageSquare className="w-6 h-6" />}
      </button>
    </div>
  );
}
