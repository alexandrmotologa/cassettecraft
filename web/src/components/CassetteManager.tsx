import React, { useState, useEffect } from 'react';
import { 
  FileCode2, 
  Plus, 
  Trash2, 
  Check, 
  Save, 
  Database, 
  Edit3, 
  Code2
} from 'lucide-react';
import { CassetteSummary, Cassette, Interaction } from '../types/api';

interface CassetteManagerProps {
  activeCassette: string;
  onActiveCassetteChange: (name: string) => void;
}

export const CassetteManager: React.FC<CassetteManagerProps> = ({
  activeCassette,
  onActiveCassetteChange,
}) => {
  const [cassettes, setCassettes] = useState<CassetteSummary[]>([]);
  const [selectedCassetteName, setSelectedCassetteName] = useState<string>(activeCassette || '');
  const [currentCassette, setCurrentCassette] = useState<Cassette | null>(null);
  const [selectedInteraction, setSelectedInteraction] = useState<Interaction | null>(null);
  const [newCassetteName, setNewCassetteName] = useState('');
  const [showNewModal, setShowNewModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Edit mode state for response
  const [editStatus, setEditStatus] = useState<number>(200);
  const [editBody, setEditBody] = useState<string>('');

  const fetchCassettes = async () => {
    try {
      const res = await fetch('/api/v1/cassettes');
      if (res.ok) {
        const data: CassetteSummary[] = await res.json();
        setCassettes(data);
        if (!selectedCassetteName && data.length > 0) {
          const active = data.find((c) => c.active) || data[0];
          setSelectedCassetteName(active.name);
        }
      }
    } catch (err) {
      console.error('Failed to fetch cassettes:', err);
    }
  };

  const loadCassette = async (name: string) => {
    if (!name) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/cassettes/${encodeURIComponent(name)}`);
      if (res.ok) {
        const data = await res.json();
        setCurrentCassette(data.cassette);
        if (data.cassette.interactions && data.cassette.interactions.length > 0) {
          const first = data.cassette.interactions[0];
          setSelectedInteraction(first);
          setEditStatus(first.response.status);
          setEditBody(first.response.body || '');
        } else {
          setSelectedInteraction(null);
        }
      }
    } catch (err) {
      console.error('Failed to load cassette details:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCassettes();
  }, []);

  useEffect(() => {
    if (selectedCassetteName) {
      loadCassette(selectedCassetteName);
    }
  }, [selectedCassetteName]);

  const handleSelectInteraction = (inter: Interaction) => {
    setSelectedInteraction(inter);
    setEditStatus(inter.response.status);
    setEditBody(inter.response.body || '');
  };

  const handleCreateCassette = async () => {
    if (!newCassetteName.trim()) return;
    let name = newCassetteName.trim();
    if (!name.endsWith('.yaml') && !name.endsWith('.json')) {
      name += '.yaml';
    }

    const newCassetteData: Cassette = {
      version: 1,
      recorded_at: new Date().toISOString(),
      recorded_by: 'cassettecraft/v1.0',
      interactions: [],
    };

    try {
      const res = await fetch(`/api/v1/cassettes/${encodeURIComponent(name)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newCassetteData),
      });

      if (res.ok) {
        setNewCassetteName('');
        setShowNewModal(false);
        await fetchCassettes();
        setSelectedCassetteName(name);
      }
    } catch (err) {
      console.error('Failed to create cassette:', err);
    }
  };

  const handleSaveInteractionChanges = async () => {
    if (!currentCassette || !selectedInteraction || !selectedCassetteName) return;

    // Update interaction in memory
    const updatedInteractions = currentCassette.interactions.map((item) => {
      if (item.id === selectedInteraction.id) {
        return {
          ...item,
          response: {
            ...item.response,
            status: editStatus,
            body: editBody,
          },
        };
      }
      return item;
    });

    const updatedCassette: Cassette = {
      ...currentCassette,
      interactions: updatedInteractions,
    };

    try {
      const res = await fetch(`/api/v1/cassettes/${encodeURIComponent(selectedCassetteName)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updatedCassette),
      });

      if (res.ok) {
        setCurrentCassette(updatedCassette);
        setSelectedInteraction({
          ...selectedInteraction,
          response: {
            ...selectedInteraction.response,
            status: editStatus,
            body: editBody,
          },
        });
        setSaveSuccess(true);
        setTimeout(() => setSaveSuccess(false), 2000);
      }
    } catch (err) {
      console.error('Failed to save interaction:', err);
    }
  };

  const handleDeleteInteraction = async (id: string) => {
    if (!currentCassette || !selectedCassetteName) return;
    if (!confirm('Are you sure you want to remove this interaction from the cassette?')) return;

    const filtered = currentCassette.interactions.filter((i) => i.id !== id);
    const updated: Cassette = {
      ...currentCassette,
      interactions: filtered,
    };

    try {
      const res = await fetch(`/api/v1/cassettes/${encodeURIComponent(selectedCassetteName)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated),
      });

      if (res.ok) {
        setCurrentCassette(updated);
        setSelectedInteraction(filtered[0] || null);
        await fetchCassettes();
      }
    } catch (err) {
      console.error('Failed to delete interaction:', err);
    }
  };

  const handleDeleteCassette = async (name: string) => {
    if (!confirm(`Delete cassette file "${name}" completely from disk?`)) return;

    try {
      const res = await fetch(`/api/v1/cassettes/${encodeURIComponent(name)}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        await fetchCassettes();
        if (selectedCassetteName === name) {
          setSelectedCassetteName('');
          setCurrentCassette(null);
        }
      }
    } catch (err) {
      console.error('Failed to delete cassette:', err);
    }
  };

  return (
    <div className="flex h-[calc(100vh-61px)]">
      {/* Column 1: Cassette Files List */}
      <div className="w-72 border-r border-slate-800 bg-[#0b101c] flex flex-col">
        <div className="p-3 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Database className="w-4 h-4 text-indigo-400" />
            <span className="text-xs font-semibold text-slate-200 uppercase tracking-wider">Cassettes</span>
          </div>
          <button
            onClick={() => setShowNewModal(true)}
            className="p-1 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-xs flex items-center gap-1 px-2 font-medium transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New</span>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {cassettes.map((c) => {
            const isSelected = selectedCassetteName === c.name;
            const isActive = activeCassette === c.name;

            return (
              <div
                key={c.name}
                onClick={() => setSelectedCassetteName(c.name)}
                className={`p-2.5 rounded-lg border text-xs cursor-pointer transition-all ${
                  isSelected
                    ? 'bg-indigo-950/40 border-indigo-500/40 text-slate-100 shadow-sm'
                    : 'bg-slate-900/40 border-slate-800/80 text-slate-300 hover:bg-slate-800/50'
                }`}
              >
                <div className="flex items-center justify-between font-mono font-medium">
                  <div className="flex items-center gap-2 truncate">
                    <FileCode2 className="w-3.5 h-3.5 text-indigo-400 shrink-0" />
                    <span className="truncate">{c.name}</span>
                  </div>
                  {isActive && (
                    <span className="text-[10px] bg-emerald-950 text-emerald-300 border border-emerald-800/50 px-1.5 py-0.2 rounded font-semibold">
                      ACTIVE
                    </span>
                  )}
                </div>

                <div className="mt-2 flex items-center justify-between text-[11px] text-slate-400">
                  <span>{c.interactions} interaction{c.interactions === 1 ? '' : 's'}</span>
                  <div className="flex items-center gap-2">
                    {!isActive && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onActiveCassetteChange(c.name);
                        }}
                        className="text-indigo-400 hover:text-indigo-300 hover:underline"
                        title="Set as active cassette"
                      >
                        Set Active
                      </button>
                    )}
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDeleteCassette(c.name);
                      }}
                      className="text-slate-500 hover:text-rose-400"
                      title="Delete cassette"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Column 2: Interactions in Selected Cassette */}
      <div className="w-80 border-r border-slate-800 bg-[#0e1424] flex flex-col">
        <div className="p-3 border-b border-slate-800 flex items-center justify-between">
          <div className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
            Interactions ({currentCassette?.interactions?.length || 0})
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          {loading ? (
            <div className="p-8 text-center text-xs text-slate-500">Loading cassette interactions...</div>
          ) : !currentCassette || currentCassette.interactions.length === 0 ? (
            <div className="p-8 text-center text-xs text-slate-500">No interactions recorded in this cassette yet.</div>
          ) : (
            currentCassette.interactions.map((inter, idx) => {
              const isSelected = selectedInteraction?.id === inter.id;
              return (
                <div
                  key={inter.id || idx}
                  onClick={() => handleSelectInteraction(inter)}
                  className={`p-2.5 rounded-lg border text-xs cursor-pointer font-mono transition-all ${
                    isSelected
                      ? 'bg-indigo-950/50 border-indigo-500/50 text-slate-100'
                      : 'bg-slate-900/30 border-slate-800/60 text-slate-300 hover:bg-slate-800/40'
                  }`}
                >
                  <div className="flex items-center justify-between gap-1 mb-1">
                    <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-slate-800 text-indigo-300 border border-slate-700">
                      {inter.request.method}
                    </span>
                    <span className="text-[10px] font-bold text-emerald-400">
                      {inter.response.status}
                    </span>
                  </div>
                  <div className="truncate text-slate-200 font-medium">{inter.request.url}</div>
                  <div className="mt-1 flex items-center justify-between text-[10px] text-slate-500">
                    <span>{inter.id}</span>
                    <span>{inter.response.latency_ms || 0} ms</span>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Column 3: Interaction Detail & Inline Response Editor */}
      <div className="flex-1 bg-[#090d16] flex flex-col overflow-hidden">
        {selectedInteraction ? (
          <div className="flex-1 flex flex-col h-full overflow-hidden">
            {/* Action Bar */}
            <div className="p-3 border-b border-slate-800 bg-slate-900/60 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Edit3 className="w-4 h-4 text-indigo-400" />
                <span className="text-xs font-semibold text-slate-200">Interactive Mock Editor</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleDeleteInteraction(selectedInteraction.id)}
                  className="px-2.5 py-1 text-xs text-rose-400 hover:bg-rose-500/10 rounded border border-rose-500/20 transition-colors flex items-center gap-1"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Delete</span>
                </button>
                <button
                  onClick={handleSaveInteractionChanges}
                  className="px-3 py-1 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded font-medium transition-colors flex items-center gap-1.5 shadow-sm"
                >
                  {saveSuccess ? <Check className="w-3.5 h-3.5 text-emerald-300" /> : <Save className="w-3.5 h-3.5" />}
                  <span>{saveSuccess ? 'Saved to Disk' : 'Save Changes'}</span>
                </button>
              </div>
            </div>

            {/* Editor Workspace */}
            <div className="flex-1 overflow-y-auto p-4 space-y-4 font-mono text-xs">
              {/* Request Info Card */}
              <div className="bg-slate-900/50 border border-slate-800 rounded-lg p-3">
                <div className="text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-2">Request Signature</div>
                <div className="flex items-center gap-2 text-slate-200 mb-1">
                  <span className="font-bold text-indigo-400">{selectedInteraction.request.method}</span>
                  <span className="truncate">{selectedInteraction.request.url}</span>
                </div>
                {selectedInteraction.request.body && (
                  <pre className="mt-2 bg-slate-950 p-2.5 rounded border border-slate-800/80 text-slate-300 whitespace-pre-wrap max-h-36 overflow-y-auto">
                    {selectedInteraction.request.body}
                  </pre>
                )}
              </div>

              {/* Status Code Modifier */}
              <div>
                <label className="block text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-1.5">
                  HTTP Status Code
                </label>
                <div className="flex items-center gap-3">
                  <input
                    type="number"
                    value={editStatus}
                    onChange={(e) => setEditStatus(parseInt(e.target.value) || 200)}
                    className="w-32 bg-slate-900 border border-slate-800 rounded-lg px-3 py-1.5 text-xs text-slate-100 font-mono focus:outline-hidden focus:border-indigo-500"
                  />
                  <div className="flex gap-1.5">
                    {[200, 201, 400, 401, 404, 429, 500].map((code) => (
                      <button
                        key={code}
                        type="button"
                        onClick={() => setEditStatus(code)}
                        className={`px-2 py-1 rounded text-[11px] border font-semibold ${
                          editStatus === code
                            ? 'bg-indigo-600 text-white border-indigo-500'
                            : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
                        }`}
                      >
                        {code}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* Response Payload Editor */}
              <div className="flex-1 flex flex-col">
                <label className="block text-[11px] uppercase tracking-wider text-slate-400 font-semibold mb-1.5">
                  Response Body (JSON or Text)
                </label>
                <textarea
                  value={editBody}
                  onChange={(e) => setEditBody(e.target.value)}
                  rows={16}
                  className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-xs text-slate-200 font-mono leading-relaxed focus:outline-hidden focus:border-indigo-500"
                  placeholder="Enter JSON response payload..."
                />
              </div>
            </div>
          </div>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center text-slate-500 p-8 text-center">
            <Code2 className="w-10 h-10 mb-2 opacity-30 text-indigo-400" />
            <div className="text-sm font-medium text-slate-400">Select an interaction to inspect and edit</div>
          </div>
        )}
      </div>

      {/* New Cassette Modal */}
      {showNewModal && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-xs flex items-center justify-center z-50 p-4">
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 max-w-sm w-full shadow-2xl">
            <h3 className="text-sm font-semibold text-slate-100 mb-1">Create New Cassette</h3>
            <p className="text-xs text-slate-400 mb-4">Enter a file name for the new cassette (e.g. stripe_payments.yaml)</p>
            <input
              type="text"
              placeholder="e.g. github_auth.yaml"
              value={newCassetteName}
              onChange={(e) => setNewCassetteName(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-100 font-mono focus:outline-hidden focus:border-indigo-500 mb-4"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowNewModal(false)}
                className="px-3 py-1.5 rounded-lg text-xs font-medium text-slate-400 hover:text-slate-200 hover:bg-slate-800"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateCassette}
                className="px-3 py-1.5 rounded-lg text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
