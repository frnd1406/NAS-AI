import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  HardDrive, Shield, Brain, ChevronRight, ChevronLeft,
  Check, Loader2, FolderOpen, Lock, Unlock, Sparkles,
  Server, AlertCircle
} from 'lucide-react';
import { useToast } from '../components/ui/Toast';
import { apiRequest } from '../lib/api';

const API_BASE = window.location.origin;

// Step indicator component
const StepIndicator = ({ currentStep, steps }) => (
  <div className="flex items-center justify-center gap-2 mb-8">
    {steps.map((step, idx) => (
      <React.Fragment key={step.id}>
        <div className={`flex items-center gap-2 px-4 py-2 rounded-full transition-all ${idx === currentStep
          ? 'bg-blue-500/20 border border-blue-500/30 text-blue-400'
          : idx < currentStep
            ? 'bg-emerald-500/20 border border-emerald-500/30 text-emerald-400'
            : 'bg-slate-800/50 border border-white/10 text-slate-500'
          }`}>
          {idx < currentStep ? (
            <Check size={16} />
          ) : (
            <step.icon size={16} />
          )}
          <span className="text-sm font-medium">{step.label}</span>
        </div>
        {idx < steps.length - 1 && (
          <ChevronRight size={16} className="text-slate-600" />
        )}
      </React.Fragment>
    ))}
  </div>
);

// Card wrapper
const SetupCard = ({ children, title, description }) => (
  <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/60 backdrop-blur-xl shadow-2xl max-w-2xl mx-auto">
    <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-blue-500/30 to-transparent" />
    <div className="p-8">
      {title && (
        <div className="text-center mb-6">
          <h2 className="text-2xl font-bold text-white mb-2">{title}</h2>
          {description && <p className="text-slate-400">{description}</p>}
        </div>
      )}
      {children}
    </div>
  </div>
);

// Step 1: Storage Path
const StorageStep = ({ config, setConfig }) => {
  const [customPath, setCustomPath] = useState(config.storagePath || '/mnt/data');
  const [validating, setValidating] = useState(false);
  const [valid, setValid] = useState(null);

  const validatePath = async (path) => {
    setValidating(true);
    try {
      const res = await apiRequest('/api/v1/system/validate-path', {
        method: 'POST',
        body: { path }
      });
      setValid(!res.error);
    } catch {
      setValid(false);
    }
    setValidating(false);
  };

  const handlePathChange = (path) => {
    setCustomPath(path);
    setConfig({ ...config, storagePath: path });
    setValid(null);
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4">
        {[
          { path: '/mnt/data', label: 'Standard', desc: 'Empfohlen für Produktion' },
          { path: '/media/user/DEMO', label: 'Demo', desc: 'Für Tests und Demos' },
        ].map((opt) => (
          <button
            key={opt.path}
            onClick={() => handlePathChange(opt.path)}
            className={`p-4 rounded-xl border text-left transition-all ${customPath === opt.path
              ? 'bg-blue-500/20 border-blue-500/30'
              : 'bg-slate-800/30 border-white/10 hover:border-white/20'
              }`}
          >
            <div className="flex items-center gap-3 mb-2">
              <FolderOpen size={20} className={customPath === opt.path ? 'text-blue-400' : 'text-slate-500'} />
              <span className="font-medium text-white">{opt.label}</span>
            </div>
            <p className="text-xs text-slate-400">{opt.desc}</p>
            <code className="text-xs text-slate-500 font-mono mt-1 block">{opt.path}</code>
          </button>
        ))}
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">
          Oder eigenen Pfad eingeben
        </label>
        <div className="flex gap-2">
          <div className="relative flex-1">
            <HardDrive size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
            <input
              type="text"
              value={customPath}
              onChange={(e) => handlePathChange(e.target.value)}
              className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white font-mono focus:border-blue-500/50 focus:outline-none"
              placeholder="/pfad/zu/storage"
            />
          </div>
          <button
            onClick={() => validatePath(customPath)}
            disabled={validating}
            className="px-4 py-3 bg-slate-700 hover:bg-slate-600 rounded-xl transition-colors"
          >
            {validating ? <Loader2 size={18} className="animate-spin" /> : 'Prüfen'}
          </button>
        </div>
        {valid === true && (
          <p className="text-emerald-400 text-sm mt-2 flex items-center gap-1">
            <Check size={14} /> Pfad ist gültig und beschreibbar
          </p>
        )}
        {valid === false && (
          <p className="text-rose-400 text-sm mt-2 flex items-center gap-1">
            <AlertCircle size={14} /> Pfad nicht gefunden oder nicht beschreibbar
          </p>
        )}
      </div>
    </div>
  );
};

// Step 2: Encryption
const EncryptionStep = ({ config, setConfig }) => {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const handleToggle = () => {
    setConfig({ ...config, encryptionEnabled: !config.encryptionEnabled });
  };

  const handlePasswordChange = (pw) => {
    setPassword(pw);
    setConfig({ ...config, masterPassword: pw });
  };

  return (
    <div className="space-y-6">
      <div
        onClick={handleToggle}
        className={`p-6 rounded-xl border cursor-pointer transition-all ${config.encryptionEnabled
          ? 'bg-amber-500/10 border-amber-500/30'
          : 'bg-slate-800/30 border-white/10 hover:border-white/20'
          }`}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {config.encryptionEnabled ? (
              <Lock size={32} className="text-amber-400" />
            ) : (
              <Unlock size={32} className="text-slate-500" />
            )}
            <div>
              <h3 className="text-lg font-semibold text-white">
                Zero-Knowledge Encryption
              </h3>
              <p className="text-slate-400 text-sm">
                AES-256-GCM + Argon2id
              </p>
            </div>
          </div>
          <div className={`w-14 h-8 rounded-full transition-colors ${config.encryptionEnabled ? 'bg-amber-500/30' : 'bg-slate-700'
            }`}>
            <div className={`w-6 h-6 rounded-full bg-white shadow-lg transition-all mt-1 ${config.encryptionEnabled ? 'ml-7' : 'ml-1'
              }`} />
          </div>
        </div>
      </div>

      {config.encryptionEnabled && (
        <div className="space-y-4 animate-in fade-in slide-in-from-top-2 duration-300">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Master-Passwort
            </label>
            <div className="relative">
              <Shield size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
              <input
                type="password"
                value={password}
                onChange={(e) => handlePasswordChange(e.target.value)}
                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-amber-500/50 focus:outline-none"
                placeholder="Mindestens 8 Zeichen"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Passwort bestätigen
            </label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-amber-500/50 focus:outline-none"
              placeholder="Passwort wiederholen"
            />
          </div>
          {password && confirmPassword && password !== confirmPassword && (
            <p className="text-rose-400 text-sm flex items-center gap-1">
              <AlertCircle size={14} /> Passwörter stimmen nicht überein
            </p>
          )}
          {password && password.length < 8 && (
            <p className="text-amber-400 text-sm flex items-center gap-1">
              <AlertCircle size={14} /> Mindestens 8 Zeichen erforderlich
            </p>
          )}
          <p className="text-slate-500 text-xs">
            🔒 Das Passwort wird NIEMALS gespeichert. Nur Sie kennen es.
          </p>
        </div>
      )}
    </div>
  );
};

// Step 3: AI Models
const AIModelStep = ({ config, setConfig, ollamaStatus }) => {
  const models = ollamaStatus?.models || [];
  const connected = ollamaStatus?.connected;

  const llmModels = ['qwen2.5:7b', 'llama3.2', 'mistral'];
  const embeddingModels = ['mxbai-embed-large', 'nomic-embed-text'];

  return (
    <div className="space-y-6">
      {/* Ollama Status */}
      <div className={`p-4 rounded-xl flex items-center gap-3 ${connected
        ? 'bg-emerald-500/10 border border-emerald-500/20'
        : 'bg-rose-500/10 border border-rose-500/20'
        }`}>
        <Server size={20} className={connected ? 'text-emerald-400' : 'text-rose-400'} />
        <div>
          <p className={`font-medium ${connected ? 'text-emerald-400' : 'text-rose-400'}`}>
            {connected ? 'Ollama verbunden' : 'Ollama nicht erreichbar'}
          </p>
          <p className="text-slate-400 text-sm">
            {connected ? `${models.length} Modelle verfügbar` : 'Starte Ollama auf dem Host'}
          </p>
        </div>
      </div>

      {/* LLM Model */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">
          <span className="flex items-center gap-2">
            <Brain size={16} className="text-violet-400" />
            Antwort-Modell (RAG)
          </span>
        </label>
        <select
          value={config.aiModels?.llm || 'qwen2.5:7b'}
          onChange={(e) => setConfig({
            ...config,
            aiModels: { ...config.aiModels, llm: e.target.value }
          })}
          className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
        >
          {llmModels.map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>
        <p className="text-xs text-slate-500 mt-1">Für AI-Antworten und Chat</p>
      </div>

      {/* Embedding Model */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">
          <span className="flex items-center gap-2">
            <Sparkles size={16} className="text-blue-400" />
            Embedding-Modell
          </span>
        </label>
        <select
          value={config.aiModels?.embedding || 'mxbai-embed-large'}
          onChange={(e) => setConfig({
            ...config,
            aiModels: { ...config.aiModels, embedding: e.target.value }
          })}
          className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-blue-500/50 focus:outline-none"
        >
          {embeddingModels.map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>
        <p className="text-xs text-slate-500 mt-1">Für semantische Suche (1024D Vektoren)</p>
      </div>

      <div className="p-4 bg-slate-800/30 rounded-xl border border-white/5">
        <p className="text-slate-400 text-sm">
          💡 Nach dem Setup werden die Modelle automatisch vorgeladen (Warmup).
        </p>
      </div>
    </div>
  );
};

// Main Setup Component
export default function Setup() {
  const navigate = useNavigate();
  const toast = useToast();
  const [currentStep, setCurrentStep] = useState(0);
  const [saving, setSaving] = useState(false);

  const [config, setConfig] = useState({
    storagePath: '/mnt/data',
    encryptionEnabled: false,
    masterPassword: '',
    aiModels: {
      llm: 'qwen2.5:7b',
      embedding: 'mxbai-embed-large'
    }
  });

  const steps = [
    { id: 'storage', label: 'Speicher', icon: HardDrive },
    { id: 'encryption', label: 'Verschlüsselung', icon: Shield },
  ];

  useEffect(() => {
    checkSetupStatus();
  }, []);

  const checkSetupStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/system/setup-status`, {
        credentials: 'include'
      });
      if (res.ok) {
        const data = await res.json();
        if (data.complete) {
          navigate('/dashboard', { replace: true });
        }
      }
    } catch (err) {
      console.log('Setup status check failed:', err);
    }
  };

  const handleNext = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handleBack = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleComplete = async () => {
    // Validate
    if (config.encryptionEnabled) {
      if (!config.masterPassword || config.masterPassword.length < 8) {
        toast.error('Master-Passwort muss mindestens 8 Zeichen haben');
        return;
      }
    }

    setSaving(true);
    try {
      // 1. Save setup config
      await apiRequest('/api/v1/system/setup', {
        method: 'POST',
        body: {
          storagePath: config.storagePath,
          encryptionEnabled: config.encryptionEnabled,
          aiModels: config.aiModels
        }
      });

      // 2. If encryption enabled, setup vault
      if (config.encryptionEnabled && config.masterPassword) {
        await apiRequest('/api/v1/system/vault/setup', {
          method: 'POST',
          body: { masterPassword: config.masterPassword }
        });
      }

      // AI warmup skipped — AI UI/routes disabled

      toast.success('Setup abgeschlossen! 🚀');
      navigate('/dashboard', { replace: true });
    } catch (err) {
      toast.error(err.message || 'Setup fehlgeschlagen');
    } finally {
      setSaving(false);
    }
  };

  const canProceed = () => {
    if (currentStep === 0) return config.storagePath;
    if (currentStep === 1) {
      if (!config.encryptionEnabled) return true;
      return config.masterPassword && config.masterPassword.length >= 8;
    }
    return true;
  };

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans flex items-center justify-center p-4 relative overflow-hidden">
      {/* Background effects */}
      <div className="fixed inset-0 z-0 pointer-events-none overflow-hidden">
        <div className="absolute top-[-10%] left-[-10%] w-[500px] h-[500px] bg-blue-600/20 rounded-full blur-[120px] animate-pulse" />
        <div className="absolute bottom-[-10%] right-[-5%] w-[600px] h-[600px] bg-violet-600/10 rounded-full blur-[130px]" />
      </div>

      <div className="relative z-10 w-full max-w-2xl">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-white tracking-tight mb-2">
            🚀 NAS.AI Setup
          </h1>
          <p className="text-slate-400">
            Konfiguriere dein System in 2 Schritten
          </p>
        </div>

        {/* Step Indicator */}
        <StepIndicator currentStep={currentStep} steps={steps} />

        {/* Step Content */}
        <SetupCard
          title={steps[currentStep].label}
          description={
            currentStep === 0 ? 'Wo sollen deine Daten gespeichert werden?' :
              'Möchtest du Zero-Knowledge Verschlüsselung aktivieren?'
          }
        >
          {currentStep === 0 && (
            <StorageStep config={config} setConfig={setConfig} />
          )}
          {currentStep === 1 && (
            <EncryptionStep config={config} setConfig={setConfig} />
          )}

          {/* Navigation */}
          <div className="flex items-center justify-between mt-8 pt-6 border-t border-white/10">
            <button
              onClick={handleBack}
              disabled={currentStep === 0}
              className="flex items-center gap-2 px-4 py-2.5 text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft size={18} />
              Zurück
            </button>

            {currentStep < steps.length - 1 ? (
              <button
                onClick={handleNext}
                disabled={!canProceed()}
                className="flex items-center gap-2 px-6 py-2.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Weiter
                <ChevronRight size={18} />
              </button>
            ) : (
              <button
                onClick={handleComplete}
                disabled={saving || !canProceed()}
                className="flex items-center gap-2 px-6 py-2.5 bg-gradient-to-r from-blue-600 to-violet-600 hover:from-blue-500 hover:to-violet-500 text-white rounded-xl shadow-lg shadow-blue-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? (
                  <>
                    <Loader2 size={18} className="animate-spin" />
                    Speichern...
                  </>
                ) : (
                  <>
                    <Check size={18} />
                    Setup abschließen
                  </>
                )}
              </button>
            )}
          </div>
        </SetupCard>

        {/* Version */}
        <p className="text-center text-xs text-slate-600 mt-6">
          NAS.AI v2.1 · Zero-Knowledge Encryption
        </p>
      </div>
    </div>
  );
}
