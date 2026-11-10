import React, { useState } from 'react';
import { Download, Shield, Copy, Check, Terminal } from 'lucide-react';

interface CAModalProps {
  isOpen: boolean;
  onClose: () => void;
  caFingerprint?: string;
}

export const CAModal: React.FC<CAModalProps> = ({ isOpen, onClose, caFingerprint }) => {
  const [copiedSnippet, setCopiedSnippet] = useState<string | null>(null);

  if (!isOpen) return null;

  const copySnippet = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedSnippet(id);
    setTimeout(() => setCopiedSnippet(null), 1500);
  };

  const guides = [
    {
      id: 'node',
      title: 'Node.js / Jest / Vitest',
      cmd: 'export NODE_EXTRA_CA_CERTS="$(pwd)/ca.crt"',
    },
    {
      id: 'python',
      title: 'Python (requests / httpx / pytest)',
      cmd: 'export REQUESTS_CA_BUNDLE="$(pwd)/ca.crt"\nexport SSL_CERT_FILE="$(pwd)/ca.crt"',
    },
    {
      id: 'curl',
      title: 'cURL / Shell',
      cmd: 'curl --cacert ./ca.crt -x http://localhost:8585 https://api.stripe.com/v1/charges',
    },
    {
      id: 'runner',
      title: 'Zero-Config with cassettecraft exec',
      cmd: '# The exec command sets all CA and proxy variables automatically:\ncassettecraft exec -- pytest tests/',
    },
  ];

  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-xs flex items-center justify-center z-50 p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-xl w-full p-6 shadow-2xl relative">
        <button
          onClick={onClose}
          className="absolute right-4 top-4 text-slate-500 hover:text-slate-300 text-sm"
        >
          ✕
        </button>

        <div className="flex items-center gap-3 mb-3">
          <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/30 flex items-center justify-center text-indigo-400">
            <Shield className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-slate-100">Root Certificate Authority (CA)</h3>
            <p className="text-xs text-slate-400">Used for local HTTPS MITM TLS interception</p>
          </div>
        </div>

        {caFingerprint && (
          <div className="mb-4 bg-slate-950 p-2.5 rounded-lg border border-slate-800 text-[11px] font-mono flex items-center justify-between">
            <span className="text-slate-500">SHA-256 Fingerprint:</span>
            <span className="text-indigo-300 truncate max-w-xs">{caFingerprint}</span>
          </div>
        )}

        <div className="mb-5">
          <a
            href="/ca.crt"
            download="cassettecraft-ca.crt"
            className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl text-xs font-semibold flex items-center justify-center gap-2 transition-colors shadow-sm"
          >
            <Download className="w-4 h-4" />
            <span>Download ca.crt Certificate File</span>
          </a>
        </div>

        <div className="space-y-3">
          <div className="text-[11px] uppercase tracking-wider text-slate-400 font-semibold flex items-center gap-1.5">
            <Terminal className="w-3.5 h-3.5 text-indigo-400" />
            <span>Environment Setup Guides</span>
          </div>

          <div className="space-y-2 max-h-64 overflow-y-auto pr-1">
            {guides.map((g) => (
              <div key={g.id} className="bg-slate-950 p-3 rounded-lg border border-slate-800 text-xs font-mono">
                <div className="flex items-center justify-between text-[11px] text-slate-400 mb-1.5">
                  <span className="font-semibold text-slate-300">{g.title}</span>
                  <button
                    onClick={() => copySnippet(g.cmd, g.id)}
                    className="flex items-center gap-1 text-slate-500 hover:text-indigo-400 transition-colors"
                  >
                    {copiedSnippet === g.id ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
                    <span>{copiedSnippet === g.id ? 'Copied' : 'Copy'}</span>
                  </button>
                </div>
                <pre className="text-slate-300 whitespace-pre-wrap text-[11px]">{g.cmd}</pre>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
