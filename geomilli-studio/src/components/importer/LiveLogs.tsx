import React, { useRef, useEffect, useState } from 'react';
import { Terminal, Trash2, ArrowDown, Copy, Check } from 'lucide-react';

interface LiveLogsProps {
  logs: string[];
  onClear: () => void;
}

export const LiveLogs: React.FC<LiveLogsProps> = ({ logs, onClear }) => {
  const [autoScroll, setAutoScroll] = useState(true);
  const [copied, setCopied] = useState(false);
  const terminalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const copyLogs = () => {
    navigator.clipboard.writeText(logs.join('\n'));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm flex flex-col">
      {/* Terminal Header */}
      <div className="px-4 py-2 bg-slate-950 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-emerald-400" />
          <span className="text-xs font-semibold text-slate-200">
            Harvester Live Terminal Logs
          </span>
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
        </div>

        <div className="flex items-center gap-2 text-xs">
          <label className="flex items-center gap-1.5 text-slate-400 cursor-pointer text-[11px]">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded text-brand-500 bg-slate-900 border-slate-700"
            />
            <ArrowDown className="w-3 h-3" />
            <span>Autoscroll</span>
          </label>

          <button
            onClick={copyLogs}
            className="p-1 text-slate-400 hover:text-slate-200 rounded hover:bg-slate-800"
            title="Copy logs"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          </button>

          <button
            onClick={onClear}
            className="p-1 text-slate-400 hover:text-rose-400 rounded hover:bg-slate-800"
            title="Clear logs"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Log Output Area */}
      <div
        ref={terminalRef}
        className="p-3 bg-slate-950 font-mono text-[11px] text-slate-300 h-64 overflow-y-auto leading-relaxed flex flex-col gap-1 select-text"
      >
        {logs.length === 0 ? (
          <div className="text-slate-600 italic select-none">
            [system] Awaiting harvester pipeline output stream...
          </div>
        ) : (
          logs.map((line, idx) => {
            const isWarn = line.includes('WARN');
            const isErr = line.includes('ERR') || line.includes('FATAL');
            const isSuccess = line.includes('completed') || line.includes('indexed') || line.includes('swapped');

            return (
              <div
                key={idx}
                className={`${
                  isErr
                    ? 'text-rose-400'
                    : isWarn
                    ? 'text-amber-300'
                    : isSuccess
                    ? 'text-emerald-400'
                    : 'text-slate-300'
                }`}
              >
                {line}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};
