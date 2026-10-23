import React, { useState } from 'react';
import { Terminal, Copy, Check, ChevronDown, ChevronUp, Code, Clock } from 'lucide-react';

interface DevInspectorProps {
  curlCommand: string;
  responseJson: any;
  durationMs?: number;
  meiliDurationMs?: number;
}

export const DevInspector: React.FC<DevInspectorProps> = ({
  curlCommand,
  responseJson,
  durationMs,
  meiliDurationMs,
}) => {
  const [copiedCurl, setCopiedCurl] = useState(false);
  const [copiedJson, setCopiedJson] = useState(false);
  const [isExpanded, setIsExpanded] = useState(true);

  const copyCurl = () => {
    navigator.clipboard.writeText(curlCommand);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 1500);
  };

  const copyJson = () => {
    navigator.clipboard.writeText(JSON.stringify(responseJson, null, 2));
    setCopiedJson(true);
    setTimeout(() => setCopiedJson(false), 1500);
  };

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
      {/* Header */}
      <div className="px-4 py-2.5 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-brand-400" />
          <span className="text-xs font-semibold text-slate-200">Developer Inspector</span>

          {durationMs !== undefined && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              <Clock className="w-2.5 h-2.5" />
              HTTP: {durationMs}ms
            </span>
          )}

          {meiliDurationMs !== undefined && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-mono bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              Meili: {meiliDurationMs}ms
            </span>
          )}
        </div>

        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="text-slate-400 hover:text-slate-200 p-1"
        >
          {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </button>
      </div>

      {isExpanded && (
        <div className="p-3 flex flex-col gap-3">
          {/* Live cURL Command */}
          <div>
            <div className="flex items-center justify-between text-[11px] text-slate-400 mb-1">
              <span className="font-mono text-slate-300">Live cURL</span>
              <button
                onClick={copyCurl}
                className="flex items-center gap-1 text-[10px] text-slate-400 hover:text-slate-200 px-1.5 py-0.5 rounded bg-slate-800/80 hover:bg-slate-800 border border-slate-700/60"
              >
                {copiedCurl ? (
                  <>
                    <Check className="w-3 h-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" />
                    <span>Copy cURL</span>
                  </>
                )}
              </button>
            </div>
            <pre className="p-2.5 bg-slate-950 rounded-lg text-[11px] font-mono text-emerald-300/90 overflow-x-auto border border-slate-800/80 select-all">
              {curlCommand}
            </pre>
          </div>

          {/* Raw JSON Response */}
          <div>
            <div className="flex items-center justify-between text-[11px] text-slate-400 mb-1">
              <div className="flex items-center gap-1.5">
                <Code className="w-3.5 h-3.5 text-slate-400" />
                <span className="text-slate-300">Response Payload</span>
              </div>
              <button
                onClick={copyJson}
                disabled={!responseJson}
                className="flex items-center gap-1 text-[10px] text-slate-400 hover:text-slate-200 px-1.5 py-0.5 rounded bg-slate-800/80 hover:bg-slate-800 border border-slate-700/60 disabled:opacity-50"
              >
                {copiedJson ? (
                  <>
                    <Check className="w-3 h-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" />
                    <span>Copy JSON</span>
                  </>
                )}
              </button>
            </div>
            <pre className="p-2.5 bg-slate-950 rounded-lg text-[11px] font-mono text-slate-300 max-h-52 overflow-y-auto border border-slate-800/80">
              {responseJson
                ? JSON.stringify(responseJson, null, 2)
                : '// No query response yet'}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
};
