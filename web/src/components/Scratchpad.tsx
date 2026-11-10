import React, { useState } from 'react';
import { Send, Save, Copy, Check, Terminal } from 'lucide-react';

interface ScratchpadProps {
  activeCassette: string;
}

export const Scratchpad: React.FC<ScratchpadProps> = ({ activeCassette }) => {
  const [method, setMethod] = useState('GET');
  const [url, setUrl] = useState('https://api.github.com/zen');
  const [headersText, setHeadersText] = useState('{\n  "Accept": "application/vnd.github.v3+json",\n  "User-Agent": "CassetteCraft-Scratchpad"\n}');
  const [bodyText, setBodyText] = useState('');
  
  const [loading, setLoading] = useState(false);
  const [resStatus, setResStatus] = useState<number | null>(null);
  const [resLatency, setResLatency] = useState<number | null>(null);
  const [resHeaders, setResHeaders] = useState<Record<string, string>>({});
  const [resBody, setResBody] = useState<string>('');
  const [copied, setCopied] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const handleSend = async () => {
    if (!url) return;
    setLoading(true);
    const start = performance.now();

    try {
      let parsedHeaders: Record<string, string> = {};
      if (headersText.trim()) {
        try {
          parsedHeaders = JSON.parse(headersText);
        } catch {
          // ignore parsing error
        }
      }

      const options: RequestInit = {
        method,
        headers: parsedHeaders,
      };

      if (['POST', 'PUT', 'PATCH'].includes(method) && bodyText) {
        options.body = bodyText;
      }

      const res = await fetch(url, options);
      const elapsed = Math.round(performance.now() - start);
      setResStatus(res.status);
      setResLatency(elapsed);

      const headerObj: Record<string, string> = {};
      res.headers.forEach((v, k) => {
        headerObj[k] = v;
      });
      setResHeaders(headerObj);

      const text = await res.text();
      try {
        const json = JSON.parse(text);
        setResBody(JSON.stringify(json, null, 2));
      } catch {
        setResBody(text);
      }
    } catch (err: unknown) {
      const elapsed = Math.round(performance.now() - start);
      setResStatus(0);
      setResLatency(elapsed);
      setResBody(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleSaveToCassette = async () => {
    if (!activeCassette || resStatus === null) return;

    let parsedHeaders: Record<string, string> = {};
    try {
      parsedHeaders = JSON.parse(headersText);
    } catch {
      // ignore
    }

    const newInteraction = {
      id: `int_${Date.now()}`,
      recorded_at: new Date().toISOString(),
      request: {
        method,
        url,
        headers: parsedHeaders,
        body: bodyText,
      },
      response: {
        status: resStatus,
        headers: resHeaders,
        body: resBody,
        latency_ms: resLatency || 0,
      },
    };

    try {
      // Load current cassette
      const getRes = await fetch(`/api/v1/cassettes/${encodeURIComponent(activeCassette)}`);
      if (getRes.ok) {
        const data = await getRes.json();
        const updated = {
          ...data.cassette,
          interactions: [...(data.cassette.interactions || []), newInteraction],
        };

        const saveRes = await fetch(`/api/v1/cassettes/${encodeURIComponent(activeCassette)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updated),
        });

        if (saveRes.ok) {
          setSaveSuccess(true);
          setTimeout(() => setSaveSuccess(false), 2000);
        }
      }
    } catch (err) {
      console.error('Failed to save to cassette:', err);
    }
  };

  const getCurlSnippet = () => {
    let curl = `curl -X ${method} "${url}"`;
    try {
      const parsed = JSON.parse(headersText);
      for (const [k, v] of Object.entries(parsed)) {
        curl += ` -H "${k}: ${v}"`;
      }
    } catch {
      // ignore
    }
    if (bodyText) {
      curl += ` -d '${bodyText.replace(/'/g, "'\\''")}'`;
    }
    return curl;
  };

  const copyCurl = () => {
    navigator.clipboard.writeText(getCurlSnippet());
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16] p-6 overflow-y-auto">
      <div className="max-w-6xl mx-auto w-full space-y-6">
        {/* Title */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-slate-100 flex items-center gap-2">
              <Terminal className="w-5 h-5 text-indigo-400" />
              <span>HTTP Request Scratchpad</span>
            </h2>
            <p className="text-xs text-slate-400 mt-1">
              Send test requests to real upstream APIs or local mocks, preview responses, and save them directly to active cassette <code className="text-indigo-400 font-mono">{activeCassette}</code>.
            </p>
          </div>
          <button
            onClick={copyCurl}
            className="text-xs bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800 px-3 py-1.5 rounded-lg flex items-center gap-1.5 transition-colors font-mono"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5 text-indigo-400" />}
            <span>{copied ? 'Copied cURL' : 'Copy cURL'}</span>
          </button>
        </div>

        {/* Request Builder Bar */}
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4 space-y-3">
          <div className="flex items-center gap-2">
            <select
              value={method}
              onChange={(e) => setMethod(e.target.value)}
              className="bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono font-bold text-indigo-400 focus:outline-hidden focus:border-indigo-500"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
              <option value="DELETE">DELETE</option>
            </select>

            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://api.example.com/v1/resource"
              className="flex-1 bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs font-mono text-slate-100 focus:outline-hidden focus:border-indigo-500"
            />

            <button
              onClick={handleSend}
              disabled={loading}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors shadow-sm"
            >
              <Send className="w-3.5 h-3.5" />
              <span>{loading ? 'Sending...' : 'Send Request'}</span>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs font-mono">
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-1">
                Headers (JSON format)
              </label>
              <textarea
                value={headersText}
                onChange={(e) => setHeadersText(e.target.value)}
                rows={5}
                className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-300 focus:outline-hidden resize-none font-mono"
              />
            </div>

            <div>
              <label className="block text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-1">
                Request Body
              </label>
              <textarea
                value={bodyText}
                onChange={(e) => setBodyText(e.target.value)}
                rows={5}
                placeholder="Request payload (JSON, text)..."
                className="w-full bg-slate-950 border border-slate-800 rounded-lg p-2.5 text-xs text-slate-300 focus:outline-hidden resize-none font-mono"
              />
            </div>
          </div>
        </div>

        {/* Response Viewer */}
        {resStatus !== null && (
          <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden font-mono">
            <div className="p-3 border-b border-slate-800 bg-slate-900 flex items-center justify-between text-xs">
              <div className="flex items-center gap-3">
                <span className={`px-2 py-0.5 rounded font-bold border ${
                  resStatus >= 200 && resStatus < 300 
                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                    : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                }`}>
                  Status: {resStatus}
                </span>
                <span className="text-slate-400">{resLatency} ms</span>
              </div>

              <button
                onClick={handleSaveToCassette}
                className="px-3 py-1 bg-emerald-600 hover:bg-emerald-500 text-white rounded text-xs font-medium flex items-center gap-1.5 transition-colors shadow-xs"
              >
                {saveSuccess ? <Check className="w-3.5 h-3.5" /> : <Save className="w-3.5 h-3.5" />}
                <span>{saveSuccess ? 'Saved to Cassette' : 'Save to Cassette'}</span>
              </button>
            </div>

            <div className="p-4">
              <pre className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-xs text-slate-200 overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-96 overflow-y-auto">
                {resBody}
              </pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
