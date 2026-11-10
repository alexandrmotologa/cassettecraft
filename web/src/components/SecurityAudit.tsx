import React, { useState, useEffect } from 'react';
import { ShieldAlert, CheckCircle2, AlertOctagon, RefreshCw } from 'lucide-react';

interface Violation {
  file: string;
  interaction_id: string;
  location: string;
  secret_type: string;
  snippet: string;
}

interface LintReport {
  scanned_files: number;
  clean: boolean;
  violations: Violation[];
}

export const SecurityAudit: React.FC = () => {
  const [report, setReport] = useState<LintReport | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchLint = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/lint');
      if (res.ok) {
        const data: LintReport = await res.json();
        setReport(data);
      }
    } catch (err) {
      console.error('Failed to run security lint:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLint();
  }, []);

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16] p-6 overflow-y-auto">
      <div className="max-w-5xl mx-auto w-full space-y-6">
        {/* Title */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-100 flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-indigo-400" />
              <span>Repository Security Audit & Linter</span>
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              Recursively scan all cassette files in the repository for leaked API keys, Bearer tokens, private keys, or plain passwords before git commits.
            </p>
          </div>
          <button
            onClick={fetchLint}
            disabled={loading}
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-sm"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>{loading ? 'Scanning...' : 'Re-Scan Repository'}</span>
          </button>
        </div>

        {/* Scan Status Banner */}
        {report && (
          <div className={`p-5 rounded-xl border flex items-center gap-4 ${
            report.clean
              ? 'bg-emerald-950/20 border-emerald-800/40 text-emerald-300'
              : 'bg-rose-950/20 border-rose-800/40 text-rose-300'
          }`}>
            {report.clean ? (
              <CheckCircle2 className="w-8 h-8 text-emerald-400 shrink-0" />
            ) : (
              <AlertOctagon className="w-8 h-8 text-rose-400 shrink-0" />
            )}
            <div>
              <div className="text-sm font-bold">
                {report.clean
                  ? `Clean: 0 Secrets Detected across ${report.scanned_files} Cassette Files`
                  : `Security Alert: ${report.violations.length} Potential Secret Leak(s) Detected`}
              </div>
              <div className="text-xs text-slate-400 mt-1">
                {report.clean
                  ? 'All cassette interactions are properly sanitized and safe for version control.'
                  : 'Remove or redact these secrets before committing cassettes to GitHub.'}
              </div>
            </div>
          </div>
        )}

        {/* Violations List */}
        {report && report.violations.length > 0 && (
          <div className="space-y-3 font-mono text-xs">
            {report.violations.map((v, idx) => (
              <div key={idx} className="bg-slate-900/60 border border-rose-900/50 rounded-xl p-4 space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-rose-400 text-xs">{v.secret_type}</span>
                  <span className="text-[11px] text-slate-500">{v.interaction_id}</span>
                </div>
                <div className="text-slate-300 text-xs">{v.file}</div>
                <div className="text-[11px] text-slate-400">Location: <span className="text-slate-200">{v.location}</span></div>
                <div className="bg-slate-950 p-2 rounded border border-slate-800 text-[11px] text-rose-300">
                  Snippet: {v.snippet}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
