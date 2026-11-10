import React, { useState, useEffect } from 'react';
import { Sliders, Zap, AlertTriangle, Check, Clock } from 'lucide-react';
import { ChaosConfig } from '../types/api';

export const ChaosControls: React.FC = () => {
  const [latencyMs, setLatencyMs] = useState(0);
  const [jitterMs, setJitterMs] = useState(0);
  const [errorRate, setErrorRate] = useState(0);
  const [errorStatus, setErrorStatus] = useState(500);
  const [rateLimitFreq, setRateLimitFreq] = useState(0);
  const [saveSuccess, setSaveSuccess] = useState(false);

  useEffect(() => {
    fetch('/api/v1/chaos')
      .then((res) => res.json())
      .then((data: ChaosConfig) => {
        // Latency is in nanoseconds in Go time.Duration
        setLatencyMs(Math.round((data.latency || 0) / 1000000));
        setJitterMs(Math.round((data.jitter || 0) / 1000000));
        setErrorRate(Math.round((data.error_rate || 0) * 100));
        setErrorStatus(data.error_status || 500);
        setRateLimitFreq(data.rate_limit_freq || 0);
      })
      .catch((err) => console.error('Failed to load chaos config:', err));
  }, []);

  const handleSave = async () => {
    const payload = {
      latency: latencyMs * 1000000, // convert ms to ns
      jitter: jitterMs * 1000000,
      error_rate: errorRate / 100,
      error_status: errorStatus,
      rate_limit_freq: rateLimitFreq,
    };

    try {
      const res = await fetch('/api/v1/chaos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        setSaveSuccess(true);
        setTimeout(() => setSaveSuccess(false), 2000);
      }
    } catch (err) {
      console.error('Failed to update chaos config:', err);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16] p-6 overflow-y-auto">
      <div className="max-w-3xl mx-auto w-full space-y-6">
        {/* Title */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-100 flex items-center gap-2">
              <Sliders className="w-5 h-5 text-indigo-400" />
              <span>Chaos and Latency Simulator</span>
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              Simulate high-latency networks, random API downtime, and rate limits to stress-test your application's retry and circuit breaker logic.
            </p>
          </div>
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-sm"
          >
            {saveSuccess ? <Check className="w-4 h-4 text-emerald-300" /> : <Zap className="w-4 h-4" />}
            <span>{saveSuccess ? 'Chaos Config Applied' : 'Apply Settings'}</span>
          </button>
        </div>

        {/* Latency Section */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200">
            <Clock className="w-4 h-4 text-indigo-400" />
            <span>Network Latency Simulation</span>
          </div>

          <div>
            <div className="flex justify-between text-xs mb-1.5 font-mono">
              <span className="text-slate-400">Base Synthetic Latency</span>
              <span className="text-indigo-400 font-semibold">{latencyMs} ms</span>
            </div>
            <input
              type="range"
              min="0"
              max="2000"
              step="25"
              value={latencyMs}
              onChange={(e) => setLatencyMs(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-indigo-500"
            />
            <div className="flex justify-between text-[10px] text-slate-600 font-mono mt-1">
              <span>0 ms (Instant)</span>
              <span>500 ms</span>
              <span>1000 ms</span>
              <span>2000 ms (2 sec)</span>
            </div>
          </div>

          <div>
            <div className="flex justify-between text-xs mb-1.5 font-mono">
              <span className="text-slate-400">Random Jitter Window</span>
              <span className="text-indigo-400 font-semibold">+{jitterMs} ms</span>
            </div>
            <input
              type="range"
              min="0"
              max="500"
              step="10"
              value={jitterMs}
              onChange={(e) => setJitterMs(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-indigo-500"
            />
          </div>
        </div>

        {/* Fault Injection Section */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200">
            <AlertTriangle className="w-4 h-4 text-rose-400" />
            <span>Fault and Error Injection</span>
          </div>

          <div>
            <div className="flex justify-between text-xs mb-1.5 font-mono">
              <span className="text-slate-400">Random Error Rate</span>
              <span className="text-rose-400 font-semibold">{errorRate}%</span>
            </div>
            <input
              type="range"
              min="0"
              max="100"
              step="5"
              value={errorRate}
              onChange={(e) => setErrorRate(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-rose-500"
            />
            <div className="flex justify-between text-[10px] text-slate-600 font-mono mt-1">
              <span>0% (Disabled)</span>
              <span>25%</span>
              <span>50%</span>
              <span>100% (All Requests Fail)</span>
            </div>
          </div>

          <div>
            <label className="block text-xs text-slate-400 font-mono mb-2">Injected HTTP Error Code</label>
            <div className="grid grid-cols-4 gap-2">
              {[500, 502, 503, 429].map((code) => (
                <button
                  key={code}
                  type="button"
                  onClick={() => setErrorStatus(code)}
                  className={`py-2 rounded-lg border text-xs font-mono font-semibold transition-all ${
                    errorStatus === code
                      ? 'bg-rose-600 text-white border-rose-500 shadow-sm'
                      : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
                  }`}
                >
                  HTTP {code}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs text-slate-400 font-mono mb-1.5">
              Rate Limit Frequency (Fail with 429 every Nth request)
            </label>
            <input
              type="number"
              min="0"
              placeholder="0 (disabled)"
              value={rateLimitFreq || ''}
              onChange={(e) => setRateLimitFreq(parseInt(e.target.value) || 0)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-100 font-mono focus:outline-hidden focus:border-indigo-500"
            />
            <p className="text-[11px] text-slate-500 mt-1 font-mono">
              e.g. Set to 5 to trigger HTTP 429 Too Many Requests on every 5th call.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
