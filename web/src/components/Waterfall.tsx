import React, { useState, useMemo } from 'react';
import { 
  Search, 
  AlertTriangle, 
  Clock, 
  Copy, 
  Check, 
  Database,
  Layers
} from 'lucide-react';
import { LiveInteraction } from '../types/api';

interface WaterfallProps {
  interactions: LiveInteraction[];
  onSelectForDiff?: (interaction: LiveInteraction) => void;
}

export const Waterfall: React.FC<WaterfallProps> = ({ interactions, onSelectForDiff }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedMethod, setSelectedMethod] = useState<string>('ALL');
  const [selectedStatus, setSelectedStatus] = useState<string>('ALL');
  const [selectedItem, setSelectedItem] = useState<LiveInteraction | null>(null);
  const [inspectorTab, setInspectorTab] = useState<'request' | 'response' | 'headers' | 'diagnostics'>('response');
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const filteredInteractions = useMemo(() => {
    return [...interactions].reverse().filter((item) => {
      if (selectedMethod !== 'ALL' && item.method.toUpperCase() !== selectedMethod) {
        return false;
      }
      if (selectedStatus === '2XX' && (item.status < 200 || item.status >= 300)) {
        return false;
      }
      if (selectedStatus === '4XX' && (item.status < 400 || item.status >= 500)) {
        return false;
      }
      if (selectedStatus === '5XX' && item.status < 500) {
        return false;
      }
      if (searchTerm) {
        const query = searchTerm.toLowerCase();
        return (
          item.url.toLowerCase().includes(query) ||
          item.path.toLowerCase().includes(query) ||
          item.host.toLowerCase().includes(query) ||
          (item.req_body && item.req_body.toLowerCase().includes(query)) ||
          (item.res_body && item.res_body.toLowerCase().includes(query))
        );
      }
      return true;
    });
  }, [interactions, selectedMethod, selectedStatus, searchTerm]);

  // Statistics
  const stats = useMemo(() => {
    const total = interactions.length;
    const replayed = interactions.filter((i) => i.mode === 'replay' || (i.mode === 'auto' && i.matched)).length;
    const recorded = interactions.filter((i) => i.mode === 'record' || (i.mode === 'auto' && !i.matched)).length;
    const errors = interactions.filter((i) => i.status >= 400).length;
    const avgLatency = total > 0 
      ? Math.round(interactions.reduce((acc, i) => acc + (i.latency_ms || 0), 0) / total) 
      : 0;

    return { total, replayed, recorded, errors, avgLatency };
  }, [interactions]);

  const copyToClipboard = (text: string, fieldName: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    setTimeout(() => setCopiedField(null), 1500);
  };

  const getMethodBadge = (method: string) => {
    switch (method.toUpperCase()) {
      case 'GET':
        return 'bg-blue-500/10 text-blue-400 border-blue-500/30';
      case 'POST':
        return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30';
      case 'PUT':
      case 'PATCH':
        return 'bg-amber-500/10 text-amber-400 border-amber-500/30';
      case 'DELETE':
        return 'bg-rose-500/10 text-rose-400 border-rose-500/30';
      default:
        return 'bg-slate-500/10 text-slate-400 border-slate-500/30';
    }
  };

  const getStatusBadge = (status: number) => {
    if (status >= 200 && status < 300) {
      return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30';
    }
    if (status >= 300 && status < 400) {
      return 'bg-blue-500/10 text-blue-400 border-blue-500/30';
    }
    if (status >= 400 && status < 500) {
      return 'bg-amber-500/10 text-amber-400 border-amber-500/30';
    }
    return 'bg-rose-500/10 text-rose-400 border-rose-500/30';
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)]">
      {/* Metric Cards */}
      <div className="border-b border-slate-800 bg-[#0b101b] px-4 py-3">
        <div className="max-w-7xl mx-auto grid grid-cols-2 md:grid-cols-5 gap-3">
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-3">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Total Calls</div>
            <div className="text-xl font-bold font-mono text-slate-100 mt-1">{stats.total}</div>
          </div>
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-3">
            <div className="text-[11px] font-medium text-emerald-400 uppercase tracking-wider flex items-center gap-1">
              <span>Replayed (&lt; 1ms)</span>
            </div>
            <div className="text-xl font-bold font-mono text-emerald-400 mt-1">{stats.replayed}</div>
          </div>
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-3">
            <div className="text-[11px] font-medium text-rose-400 uppercase tracking-wider">Recorded Upstream</div>
            <div className="text-xl font-bold font-mono text-rose-400 mt-1">{stats.recorded}</div>
          </div>
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-3">
            <div className="text-[11px] font-medium text-amber-400 uppercase tracking-wider">Average Latency</div>
            <div className="text-xl font-bold font-mono text-amber-300 mt-1">{stats.avgLatency} ms</div>
          </div>
          <div className="bg-slate-900/60 border border-slate-800/80 rounded-lg p-3">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Errors / Misses</div>
            <div className="text-xl font-bold font-mono text-slate-300 mt-1">{stats.errors}</div>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="border-b border-slate-800 bg-[#0e1424] px-4 py-2 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2 flex-1 max-w-md">
          <div className="relative w-full">
            <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              placeholder="Search by URL, path, host, or body..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-hidden focus:border-indigo-500 transition-colors"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Method Filter */}
          <div className="flex items-center bg-slate-900 border border-slate-800 rounded-lg p-0.5 text-xs">
            {['ALL', 'GET', 'POST', 'PUT', 'DELETE'].map((m) => (
              <button
                key={m}
                onClick={() => setSelectedMethod(m)}
                className={`px-2 py-1 rounded font-mono font-medium transition-all ${
                  selectedMethod === m
                    ? 'bg-indigo-600 text-white shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {m}
              </button>
            ))}
          </div>

          {/* Status Filter */}
          <div className="flex items-center bg-slate-900 border border-slate-800 rounded-lg p-0.5 text-xs">
            {['ALL', '2XX', '4XX', '5XX'].map((s) => (
              <button
                key={s}
                onClick={() => setSelectedStatus(s)}
                className={`px-2 py-1 rounded font-mono font-medium transition-all ${
                  selectedStatus === s
                    ? 'bg-slate-700 text-white shadow-xs'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Main Split Content: List + Drawer */}
      <div className="flex-1 flex overflow-hidden">
        {/* Interactions Waterfall Table */}
        <div className={`flex-1 overflow-y-auto ${selectedItem ? 'hidden md:block md:w-1/2 lg:w-3/5 border-r border-slate-800' : 'w-full'}`}>
          {filteredInteractions.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center text-slate-500 p-8 text-center">
              <Layers className="w-10 h-10 mb-3 opacity-30 text-indigo-400" />
              <div className="text-sm font-medium text-slate-400">No intercepted HTTP requests yet</div>
              <div className="text-xs text-slate-600 max-w-sm mt-1">
                Run tests with <code className="text-indigo-400 font-mono">cassettecraft exec</code> or send requests through proxy at <code className="text-indigo-400 font-mono">http://localhost:8585</code>.
              </div>
            </div>
          ) : (
            <table className="w-full text-left border-collapse text-xs">
              <thead className="sticky top-0 bg-[#0f172a] text-slate-400 border-b border-slate-800 font-medium z-10">
                <tr>
                  <th className="py-2.5 px-3 w-16">Status</th>
                  <th className="py-2.5 px-3 w-16">Method</th>
                  <th className="py-2.5 px-3">Path & Query</th>
                  <th className="py-2.5 px-3 w-28">Host</th>
                  <th className="py-2.5 px-3 w-20 text-right">Latency</th>
                  <th className="py-2.5 px-3 w-20">Mode</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filteredInteractions.map((item) => {
                  const isSelected = selectedItem?.id === item.id;
                  return (
                    <tr
                      key={item.id}
                      onClick={() => setSelectedItem(item)}
                      className={`cursor-pointer transition-colors ${
                        isSelected
                          ? 'bg-indigo-950/40 hover:bg-indigo-950/50'
                          : 'hover:bg-slate-900/60'
                      }`}
                    >
                      <td className="py-2 px-3 whitespace-nowrap">
                        <span className={`px-1.5 py-0.5 rounded text-[11px] font-bold border ${getStatusBadge(item.status)}`}>
                          {item.status}
                        </span>
                      </td>
                      <td className="py-2 px-3 whitespace-nowrap">
                        <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold border ${getMethodBadge(item.method)}`}>
                          {item.method}
                        </span>
                      </td>
                      <td className="py-2 px-3 truncate max-w-xs text-slate-200">
                        {item.path}
                      </td>
                      <td className="py-2 px-3 truncate text-slate-400 text-[11px]">
                        {item.host}
                      </td>
                      <td className="py-2 px-3 whitespace-nowrap text-right text-slate-400 text-[11px]">
                        {item.latency_ms} ms
                      </td>
                      <td className="py-2 px-3 whitespace-nowrap">
                        <span className={`text-[10px] px-1.5 py-0.5 rounded uppercase font-semibold ${
                          item.matched 
                            ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-800/60' 
                            : 'bg-rose-950/80 text-rose-400 border border-rose-800/60'
                        }`}>
                          {item.mode}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>

        {/* Selected Interaction Inspector */}
        {selectedItem && (
          <div className="flex-1 md:w-1/2 lg:w-2/5 flex flex-col bg-[#0b0f19] h-full overflow-hidden">
            {/* Inspector Header */}
            <div className="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-900/60">
              <div className="flex items-center gap-2 overflow-hidden">
                <span className={`px-2 py-0.5 rounded text-xs font-bold border ${getStatusBadge(selectedItem.status)}`}>
                  {selectedItem.status}
                </span>
                <span className={`px-2 py-0.5 rounded text-xs font-semibold border ${getMethodBadge(selectedItem.method)}`}>
                  {selectedItem.method}
                </span>
                <span className="text-xs font-mono text-slate-300 truncate">
                  {selectedItem.path}
                </span>
              </div>
              <div className="flex items-center gap-1">
                {onSelectForDiff && (
                  <button
                    onClick={() => onSelectForDiff(selectedItem)}
                    className="text-[11px] flex items-center gap-1 bg-slate-800 hover:bg-slate-700 text-slate-300 px-2 py-1 rounded transition-colors"
                    title="Send to Diff Inspector"
                  >
                    <span>Diff</span>
                  </button>
                )}
                <button
                  onClick={() => setSelectedItem(null)}
                  className="text-slate-400 hover:text-slate-200 px-2 py-1 text-xs"
                >
                  ✕
                </button>
              </div>
            </div>

            {/* Inspector Metadata Bar */}
            <div className="px-4 py-2 border-b border-slate-800/80 bg-slate-950/40 flex items-center justify-between text-[11px] text-slate-400 font-mono">
              <div className="flex items-center gap-3">
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3 text-slate-500" />
                  {selectedItem.latency_ms} ms
                </span>
                <span className="flex items-center gap-1">
                  <Database className="w-3 h-3 text-slate-500" />
                  {selectedItem.cassette}
                </span>
              </div>
              <div className="text-slate-500 truncate max-w-[150px]">
                {new Date(selectedItem.timestamp).toLocaleTimeString()}
              </div>
            </div>

            {/* Inspector Tabs */}
            <div className="flex border-b border-slate-800 bg-slate-900/40 text-xs">
              <button
                onClick={() => setInspectorTab('response')}
                className={`flex-1 py-2 font-medium border-b-2 transition-all ${
                  inspectorTab === 'response'
                    ? 'border-indigo-500 text-indigo-300 bg-indigo-500/10'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                Response Body
              </button>
              <button
                onClick={() => setInspectorTab('request')}
                className={`flex-1 py-2 font-medium border-b-2 transition-all ${
                  inspectorTab === 'request'
                    ? 'border-indigo-500 text-indigo-300 bg-indigo-500/10'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                Request Body
              </button>
              <button
                onClick={() => setInspectorTab('headers')}
                className={`flex-1 py-2 font-medium border-b-2 transition-all ${
                  inspectorTab === 'headers'
                    ? 'border-indigo-500 text-indigo-300 bg-indigo-500/10'
                    : 'border-transparent text-slate-400 hover:text-slate-200'
                }`}
              >
                Headers
              </button>
              {selectedItem.error && (
                <button
                  onClick={() => setInspectorTab('diagnostics')}
                  className={`flex-1 py-2 font-medium border-b-2 text-rose-400 transition-all ${
                    inspectorTab === 'diagnostics'
                      ? 'border-rose-500 bg-rose-500/10'
                      : 'border-transparent hover:text-rose-300'
                  }`}
                >
                  Diagnostic
                </button>
              )}
            </div>

            {/* Inspector Tab Content */}
            <div className="flex-1 overflow-y-auto p-4 text-xs font-mono">
              {inspectorTab === 'response' && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-slate-400 text-[11px] uppercase tracking-wider font-semibold">Response Payload</span>
                    <button
                      onClick={() => copyToClipboard(selectedItem.res_body, 'res_body')}
                      className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-indigo-400 transition-colors"
                    >
                      {copiedField === 'res_body' ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedField === 'res_body' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed">
                    {selectedItem.res_body || '(Empty Response Body)'}
                  </pre>
                </div>
              )}

              {inspectorTab === 'request' && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-slate-400 text-[11px] uppercase tracking-wider font-semibold">Request Payload</span>
                    <button
                      onClick={() => copyToClipboard(selectedItem.req_body, 'req_body')}
                      className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-indigo-400 transition-colors"
                    >
                      {copiedField === 'req_body' ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedField === 'req_body' ? 'Copied' : 'Copy'}</span>
                    </button>
                  </div>
                  <pre className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed">
                    {selectedItem.req_body || '(Empty Request Body)'}
                  </pre>
                </div>
              )}

              {inspectorTab === 'headers' && (
                <div className="space-y-4">
                  {/* Request Headers */}
                  <div>
                    <div className="text-[11px] uppercase text-indigo-400 font-semibold mb-1.5">Request Headers</div>
                    <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800 space-y-1">
                      {Object.entries(selectedItem.req_headers || {}).map(([k, v]) => (
                        <div key={k} className="flex gap-2">
                          <span className="text-slate-400 font-semibold">{k}:</span>
                          <span className="text-slate-200 break-all">{v}</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Response Headers */}
                  <div>
                    <div className="text-[11px] uppercase text-emerald-400 font-semibold mb-1.5">Response Headers</div>
                    <div className="bg-slate-950 p-2.5 rounded-lg border border-slate-800 space-y-1">
                      {Object.entries(selectedItem.res_headers || {}).map(([k, v]) => (
                        <div key={k} className="flex gap-2">
                          <span className="text-slate-400 font-semibold">{k}:</span>
                          <span className="text-slate-200 break-all">{v}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {inspectorTab === 'diagnostics' && selectedItem.error && (
                <div>
                  <div className="bg-rose-950/30 border border-rose-800/60 p-3 rounded-lg text-rose-300">
                    <div className="font-semibold text-xs flex items-center gap-1.5 mb-1">
                      <AlertTriangle className="w-4 h-4 text-rose-400" />
                      <span>Cassette Mismatch Diagnostics</span>
                    </div>
                    <p className="text-xs text-rose-200/90 leading-relaxed">{selectedItem.error}</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
