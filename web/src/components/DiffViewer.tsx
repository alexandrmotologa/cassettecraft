import React, { useState } from 'react';
import { Split, RefreshCw } from 'lucide-react';

interface DiffViewerProps {
  initialLeft?: string;
  initialRight?: string;
}

export const DiffViewer: React.FC<DiffViewerProps> = ({ initialLeft, initialRight }) => {
  const [leftText, setLeftText] = useState(initialLeft || '{\n  "status": "success",\n  "amount": 2000,\n  "currency": "usd",\n  "customer": "cus_123456789"\n}');
  const [rightText, setRightText] = useState(initialRight || '{\n  "status": "success",\n  "amount": 2500,\n  "currency": "usd",\n  "customer": "cus_123456789",\n  "tier": "enterprise"\n}');

  // Simple line-by-line diff computation
  const leftLines = leftText.split('\n');
  const rightLines = rightText.split('\n');
  const maxLines = Math.max(leftLines.length, rightLines.length);

  const formatJSON = (text: string, setter: (val: string) => void) => {
    try {
      const parsed = JSON.parse(text);
      setter(JSON.stringify(parsed, null, 2));
    } catch {
      // not JSON or syntax error
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16]">
      {/* Header */}
      <div className="border-b border-slate-800 bg-[#0c1220] px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Split className="w-4 h-4 text-indigo-400" />
          <span className="text-xs font-semibold text-slate-200 uppercase tracking-wider">Payload Diff Inspector</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              formatJSON(leftText, setLeftText);
              formatJSON(rightText, setRightText);
            }}
            className="text-xs bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800 px-2.5 py-1.5 rounded-lg flex items-center gap-1.5 transition-colors"
          >
            <RefreshCw className="w-3 h-3 text-indigo-400" />
            <span>Format JSON</span>
          </button>
        </div>
      </div>

      {/* Editor & Diff Columns */}
      <div className="flex-1 grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-slate-800 overflow-hidden">
        {/* Left Column: Recorded Baseline */}
        <div className="flex flex-col h-full bg-[#0a0f1d] overflow-hidden">
          <div className="p-2.5 border-b border-slate-800 bg-slate-900/60 text-[11px] font-semibold text-slate-400 uppercase tracking-wider flex items-center justify-between">
            <span>Recorded Baseline (Cassette)</span>
            <span className="text-slate-500 font-mono">{leftLines.length} lines</span>
          </div>
          <div className="flex-1 p-3 overflow-y-auto">
            <textarea
              value={leftText}
              onChange={(e) => setLeftText(e.target.value)}
              className="w-full h-full bg-slate-950 border border-slate-800/80 rounded-lg p-3 text-xs text-slate-200 font-mono leading-relaxed focus:outline-hidden focus:border-indigo-500"
              placeholder="Paste recorded baseline JSON here..."
            />
          </div>
        </div>

        {/* Right Column: Live Incoming Payload */}
        <div className="flex flex-col h-full bg-[#0a0f1d] overflow-hidden">
          <div className="p-2.5 border-b border-slate-800 bg-slate-900/60 text-[11px] font-semibold text-indigo-400 uppercase tracking-wider flex items-center justify-between">
            <span>Live Response (Test Execution)</span>
            <span className="text-slate-500 font-mono">{rightLines.length} lines</span>
          </div>
          <div className="flex-1 p-3 overflow-y-auto">
            <textarea
              value={rightText}
              onChange={(e) => setRightText(e.target.value)}
              className="w-full h-full bg-slate-950 border border-slate-800/80 rounded-lg p-3 text-xs text-slate-200 font-mono leading-relaxed focus:outline-hidden focus:border-indigo-500"
              placeholder="Paste live response JSON here..."
            />
          </div>
        </div>
      </div>

      {/* Line-by-Line Diff Preview Bar */}
      <div className="border-t border-slate-800 bg-slate-950 p-4 max-h-56 overflow-y-auto font-mono text-xs">
        <div className="text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-2">Visual Line Diff</div>
        <div className="space-y-0.5">
          {Array.from({ length: maxLines }).map((_, idx) => {
            const left = leftLines[idx] ?? '';
            const right = rightLines[idx] ?? '';
            const isMatch = left === right;

            return (
              <div key={idx} className={`flex text-xs px-2 py-0.5 rounded ${isMatch ? 'text-slate-400' : 'bg-amber-950/20 text-amber-300'}`}>
                <div className="w-12 text-slate-600 select-none text-right pr-3">{idx + 1}</div>
                <div className={`flex-1 ${!isMatch ? 'bg-rose-950/40 text-rose-300 px-1 rounded' : ''}`}>{left || ' '}</div>
                <div className="w-6 text-center text-slate-600 select-none">{isMatch ? '=' : '≠'}</div>
                <div className={`flex-1 ${!isMatch ? 'bg-emerald-950/40 text-emerald-300 px-1 rounded' : ''}`}>{right || ' '}</div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};
