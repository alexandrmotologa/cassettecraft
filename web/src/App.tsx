import React, { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { Waterfall } from './components/Waterfall';
import { CassetteManager } from './components/CassetteManager';
import { DiffViewer } from './components/DiffViewer';
import { RedactionTester } from './components/RedactionTester';
import { ChaosControls } from './components/ChaosControls';
import { CAModal } from './components/CAModal';
import { LiveInteraction, ProxyMode, StatusResponse } from './types/api';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<string>('waterfall');
  const [mode, setMode] = useState<ProxyMode>('record');
  const [activeCassette, setActiveCassette] = useState<string>('default.yaml');
  const [interactions, setInteractions] = useState<LiveInteraction[]>([]);
  const [caFingerprint, setCaFingerprint] = useState<string>('');
  const [caModalOpen, setCaModalOpen] = useState<boolean>(false);

  // Pre-fill diff inspector from waterfall
  const [diffLeft] = useState<string>('');
  const [diffRight, setDiffRight] = useState<string>('');

  const fetchStatus = async () => {
    try {
      const res = await fetch('/api/v1/status');
      if (res.ok) {
        const data: StatusResponse = await res.json();
        setMode(data.mode);
        setActiveCassette(data.active_cassette);
        setCaFingerprint(data.ca_fingerprint);
      }
    } catch {
      // offline or disconnected
    }
  };

  const fetchInteractions = async () => {
    try {
      const res = await fetch('/api/v1/interactions/live');
      if (res.ok) {
        const data: LiveInteraction[] = await res.json();
        setInteractions(data);
      }
    } catch {
      // offline
    }
  };

  useEffect(() => {
    fetchStatus();
    fetchInteractions();

    const interval = setInterval(() => {
      fetchStatus();
      fetchInteractions();
    }, 1500);

    return () => clearInterval(interval);
  }, []);

  const handleModeChange = async (newMode: ProxyMode) => {
    try {
      const res = await fetch('/api/v1/mode', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: newMode }),
      });
      if (res.ok) {
        setMode(newMode);
      }
    } catch (err) {
      console.error('Failed to change proxy mode:', err);
    }
  };

  const handleActiveCassetteChange = async (name: string) => {
    try {
      const res = await fetch('/api/v1/active-cassette', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
      if (res.ok) {
        setActiveCassette(name);
      }
    } catch (err) {
      console.error('Failed to change active cassette:', err);
    }
  };

  const handlePurge = async () => {
    try {
      const res = await fetch('/api/v1/interactions/purge', {
        method: 'POST',
      });
      if (res.ok) {
        setInteractions([]);
      }
    } catch (err) {
      console.error('Failed to purge buffer:', err);
    }
  };

  const handleSelectForDiff = (item: LiveInteraction) => {
    setDiffRight(item.res_body || '');
    setCurrentTab('diff');
  };

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col font-sans">
      <Navbar
        currentTab={currentTab}
        setCurrentTab={setCurrentTab}
        mode={mode}
        onModeChange={handleModeChange}
        activeCassette={activeCassette}
        interactionsCount={interactions.length}
        onPurge={handlePurge}
        onOpenCAModal={() => setCaModalOpen(true)}
      />

      <main className="flex-1 overflow-hidden">
        {currentTab === 'waterfall' && (
          <Waterfall 
            interactions={interactions} 
            onSelectForDiff={handleSelectForDiff} 
          />
        )}
        {currentTab === 'cassettes' && (
          <CassetteManager
            activeCassette={activeCassette}
            onActiveCassetteChange={handleActiveCassetteChange}
          />
        )}
        {currentTab === 'diff' && (
          <DiffViewer 
            initialLeft={diffLeft} 
            initialRight={diffRight} 
          />
        )}
        {currentTab === 'redaction' && <RedactionTester />}
        {currentTab === 'chaos' && <ChaosControls />}
      </main>

      <CAModal
        isOpen={caModalOpen}
        onClose={() => setCaModalOpen(false)}
        caFingerprint={caFingerprint}
      />
    </div>
  );
};

export default App;
