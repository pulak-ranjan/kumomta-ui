import React, { useState, useRef, useEffect } from "react";
import { Bot, Send, X, MessageSquare, Loader2, Terminal } from "lucide-react";
import { sendAIChat } from "../api";
import { cn } from "../lib/utils";

export default function AIAssistant() {
  const [isOpen, setIsOpen] = useState(false);
  const [input, setInput] = useState("");
  const [messages, setMessages] = useState([
    { role: "assistant", content: "Hello! I'm your KumoMTA Agent. I can check logs, queues, and documentation. How can I help?" }
  ]);
  const [loading, setLoading] = useState(false);
  const scrollRef = useRef(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isOpen]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const newMsg = { role: "user", content: input };
    const newHistory = [...messages, newMsg];
    
    setMessages(newHistory);
    setInput("");
    setLoading(true);

    try {
      // Send entire history for context
      const res = await sendAIChat(newHistory);
      setMessages([...newHistory, { role: "assistant", content: res.reply }]);
    } catch (err) {
      setMessages([...newHistory, { role: "assistant", content: "Error: " + err.message }]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col items-end gap-4">
      {isOpen && (
        <div className="w-[380px] h-[500px] bg-card border border-border rounded-xl shadow-2xl flex flex-col animate-in slide-in-from-bottom-5 fade-in duration-200 overflow-hidden">
          {/* Header */}
          <div className="p-3 border-b bg-primary/5 flex justify-between items-center">
            <div className="flex items-center gap-2">
              <div className="p-1.5 bg-primary/10 rounded-md text-primary">
                <Bot className="w-4 h-4" />
              </div>
              <span className="font-semibold text-sm">Kumo Agent</span>
            </div>
            <button onClick={() => setIsOpen(false)} className="hover:bg-muted p-1 rounded transition-colors">
              <X className="w-4 h-4" />
            </button>
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
                  "p-3 rounded-lg max-w-[85%] whitespace-pre-wrap leading-relaxed shadow-sm",
                  m.role === "user" 
                    ? "bg-primary text-primary-foreground rounded-tr-none" 
                    : "bg-card border text-card-foreground rounded-tl-none"
                )}>
                  {m.content}
                </div>
              </div>
            ))}
            {loading && (
              <div className="flex gap-3 text-sm">
                <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                  <Loader2 className="w-3 h-3 text-primary animate-spin" />
                </div>
                <div className="text-muted-foreground text-xs py-2 italic">Analyzing logs and docs...</div>
              </div>
            )}
          </div>

          {/* Input */}
          <form onSubmit={handleSubmit} className="p-3 border-t bg-card">
            <div className="relative flex items-center">
              <input
                value={input}
                onChange={e => setInput(e.target.value)}
                placeholder="Ask about errors or config..."
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
