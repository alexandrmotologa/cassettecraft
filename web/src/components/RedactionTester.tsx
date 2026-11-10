import React, { useState } from 'react';
import { ShieldCheck, Play, Lock } from 'lucide-react';

export const RedactionTester: React.FC = () => {
  const [inputText, setInputText] = useState(`{
  "user": "developer@example.com",
  "api_token": "sk-proj-abc1234567890abcdef123456",
  "stripe_secret": "sk_test_sample_key_1234567890abcdef",
  "github_pat": "ghp_1234567890abcdef1234567890abcdef1234",
  "password": "supersecretpassword123",
  "credit_card": "4111 2222 3333 4444",
  "safe_data": "This value will remain untouched."
}`);

  const [outputText, setOutputText] = useState('');
  const [loading, setLoading] = useState(false);

  const handleTest = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/redact/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: inputText }),
      });
      if (res.ok) {
        const data = await res.json();
        setOutputText(data.sanitized);
      }
    } catch (err) {
      console.error('Failed to test redaction:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col h-[calc(100vh-61px)] bg-[#090d16] p-6 overflow-y-auto">
      <div className="max-w-6xl mx-auto w-full space-y-6">
        {/* Title */}
        <div>
          <h2 className="text-base font-semibold text-slate-100 flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-indigo-400" />
            <span>Secret Redaction Sandbox</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Test and verify that credentials, Bearer tokens, API keys, and sensitive JSON fields are properly scrubbed before writing cassettes to disk.
          </p>
        </div>

        {/* Sandbox Columns */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Input Box */}
          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
            <div className="p-3 border-b border-slate-800 bg-slate-900 flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-300">Raw Input Payload</span>
              <button
                onClick={handleTest}
                disabled={loading}
                className="px-3 py-1 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white rounded text-xs font-medium flex items-center gap-1.5 transition-colors shadow-xs"
              >
                <Play className="w-3.5 h-3.5" />
                <span>{loading ? 'Redacting...' : 'Run Redactor'}</span>
              </button>
            </div>
            <textarea
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              rows={14}
              className="p-3 bg-slate-950 text-xs font-mono text-slate-200 leading-relaxed focus:outline-hidden resize-none border-none"
              placeholder="Paste text or JSON payload containing secrets..."
            />
          </div>

          {/* Output Box */}
          <div className="flex flex-col bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
            <div className="p-3 border-b border-slate-800 bg-slate-900 flex items-center justify-between">
              <span className="text-xs font-semibold text-emerald-400">Sanitized Cassette Output</span>
              <span className="text-[11px] text-slate-500 font-mono">Clean for Git Commits</span>
            </div>
            <textarea
              readOnly
              value={outputText || '(Click "Run Redactor" to preview sanitized output)'}
              rows={14}
              className="p-3 bg-slate-950 text-xs font-mono text-emerald-300/90 leading-relaxed focus:outline-hidden resize-none border-none select-all"
            />
          </div>
        </div>

        {/* Active Redaction Rules Summary */}
        <div className="bg-slate-900/40 border border-slate-800 rounded-xl p-5">
          <h3 className="text-xs font-semibold text-slate-200 uppercase tracking-wider mb-3 flex items-center gap-2">
            <Lock className="w-4 h-4 text-indigo-400" />
            <span>Built-in Scrubber Profiles</span>
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 text-xs font-mono">
            <div className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
              <div className="font-semibold text-indigo-400 mb-1.5">HTTP Headers</div>
              <ul className="text-slate-400 space-y-1 text-[11px]">
                <li>• Authorization</li>
                <li>• Proxy-Authorization</li>
                <li>• Cookie & Set-Cookie</li>
                <li>• X-Api-Key</li>
                <li>• Stripe-Signature</li>
                <li>• X-Hub-Signature-256</li>
              </ul>
            </div>

            <div className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
              <div className="font-semibold text-indigo-400 mb-1.5">Query Parameters</div>
              <ul className="text-slate-400 space-y-1 text-[11px]">
                <li>• api_key / apikey</li>
                <li>• token</li>
                <li>• access_token</li>
                <li>• secret</li>
                <li>• signature</li>
                <li>• auth</li>
              </ul>
            </div>

            <div className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
              <div className="font-semibold text-indigo-400 mb-1.5">JSON Field Keys</div>
              <ul className="text-slate-400 space-y-1 text-[11px]">
                <li>• password & passphrase</li>
                <li>• client_secret</li>
                <li>• private_key</li>
                <li>• token & access_token</li>
                <li>• card_number & cvv</li>
                <li>• pin</li>
              </ul>
            </div>

            <div className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
              <div className="font-semibold text-indigo-400 mb-1.5">Regex Scanners</div>
              <ul className="text-slate-400 space-y-1 text-[11px]">
                <li>• Stripe sk_live / rk_live</li>
                <li>• GitHub ghp / gho / pat</li>
                <li>• OpenAI sk-[a-zA-Z0-9]</li>
                <li>• AWS AKIA / ASIA keys</li>
                <li>• JSON Web Tokens (JWT)</li>
                <li>• Credit Card Sequences</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
