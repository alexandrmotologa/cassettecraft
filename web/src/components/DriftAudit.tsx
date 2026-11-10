import React, { useState, useEffect } from 'react';
import { Compass, Play, AlertTriangle, CheckCircle2, RefreshCw } from 'lucide-react';
import { CassetteSummary } from '../types/api';

interface DriftItem {
  interaction_id: string;
  method: string;
  url: string;
  recorded_status: number;
  live_status: number;
  status_changed: boolean;
  differences: string[];
}

interface DriftReport {
  cassette: string;
  total_interactions: number;
  drifted_count: number;
  has_drift: boolean;
  items: DriftItem[];
}

export const DriftAudit: React.FC = () => {
  const [cassettes, setCassettes] = useState<CassetteSummary[]>([]);
  const [selectedCassette, setSelectedCassette] = useState('');
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<DriftReport | null>(null);

  useEffect(() => {
    fetch('/api/v1/cassettes')
      .then((res) => res.json())
      .then((data: CassetteSummary[]) => {
        setCassettes(data);
        if (data.length > 0 && !selectedCassette) {
          const active = data.find((c) => c.active) || data[0];
          setSelectedCassette(active.name);
        }
      })
      .catch((err) => console.error('Failed to load cassettes:', err));
  }, []);

  const handleRunDrift = async () => {
    if (!selectedCassette) return;
    setLoading(true);
    setReport(null);

    try {
      const res = await fetch('/api/v1/drift', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cassette: selectedCassette }),
      });
      if (res.ok) {
        const data: DriftReport = await res.json();
        setReport(data);
      }
    } catch (err) {
      console.error('Failed to run drift check:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16] p-6 overflow-y-auto">
      <div className="max-w-5xl mx-auto w-full space-y-6">
        {/* Title */}
        <div>
          <h2 className="text-base font-semibold text-slate-100 flex items-center gap-2">
            <Compass className="w-5 h-5 text-indigo-400" />
            <span>API Drift Detector</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Detect when third-party upstream APIs (Stripe, GitHub, OpenAI) change their responses, add new fields, or deprecate existing endpoints compared to your recorded cassettes.
          </p>
        </div>

        {/* Audit Control Bar */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <label className="text-xs font-semibold text-slate-300">Target Cassette:</label>
            <select
              value={selectedCassette}
              onChange={(e) => setSelectedCassette(e.target.value)}
              className="bg-slate-950 border border-slate-800 rounded-lg px-3 py-1.5 text-xs font-mono text-slate-200 focus:outline-hidden focus:border-indigo-500"
            >
              {cassettes.map((c) => (
                <option key={c.name} value={c.name}>
                  {c.name} ({c.interactions} calls)
                </option>
              ))}
            </select>
          </div>

          <button
            onClick={handleRunDrift}
            disabled={loading || !selectedCassette}
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors shadow-sm"
          >
            {loading ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Play className="w-3.5 h-3.5" />}
            <span>{loading ? 'Checking Live APIs...' : 'Run Drift Audit'}</span>
          </button>
        </div>

        {/* Audit Report Results */}
        {report && (
          <div className="space-y-4">
            {/* Status Summary Banner */}
            <div className={`p-4 rounded-xl border flex items-center gap-3 ${
              report.has_drift
                ? 'bg-amber-950/20 border-amber-800/40 text-amber-300'
                : 'bg-emerald-950/20 border-emerald-800/40 text-emerald-300'
            }`}>
              {report.has_drift ? (
                <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
              ) : (
                <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
              )}
              <div>
                <div className="text-xs font-bold">
                  {report.has_drift 
                    ? `Drift Detected in ${report.drifted_count} of ${report.total_interactions} interactions`
                    : `0 API Drift Detected across ${report.total_interactions} interactions`}
                </div>
                <div className="text-[11px] text-slate-400 mt-0.5">
                  {report.has_drift 
                    ? 'Upstream responses have differed in status code or JSON schema compared to recorded cassette.'
                    : 'The live upstream API matches your recorded cassette schema perfectly.'}
                </div>
              </div>
            </div>

            {/* Detailed Item Diffs */}
            <div className="space-y-3 font-mono text-xs">
              {report.items.map((item) => {
                const hasItemDrift = item.differences.length > 0 || item.status_changed;

                return (
                  <div
                    key={item.interaction_id}
                    className={`bg-slate-900/40 border rounded-xl p-4 transition-all ${
                      hasItemDrift ? 'border-amber-800/50' : 'border-slate-800/70'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <div className="flex items-center gap-2 truncate">
                        <span className="px-1.5 py-0.5 rounded bg-slate-800 text-indigo-300 font-bold text-[10px]">
                          {item.method}
                        </span>
                        <span className="truncate text-slate-200">{item.url}</span>
                      </div>
                      <span className={`text-[10px] px-2 py-0.5 rounded font-semibold ${
                        hasItemDrift ? 'bg-amber-950 text-amber-300 border border-amber-800' : 'bg-emerald-950 text-emerald-300'
                      }`}>
                        {hasItemDrift ? 'DRIFT' : 'MATCH'}
                      </span>
                    </div>

                    <div className="text-[11px] text-slate-400 mb-2">
                      <span>Status: recorded <span className="font-bold text-slate-200">{item.recorded_status}</span>, live <span className="font-bold text-slate-200">{item.live_status}</span></span>
                    </div>

                    {item.differences.length > 0 && (
                      <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800/80 space-y-1">
                        {item.differences.map((d, idx) => (
                          <div key={idx} className="text-amber-300/90 text-[11px] flex gap-2">
                            <span>•</span>
                            <span>{d}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
