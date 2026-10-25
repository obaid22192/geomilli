import React, { useState, useEffect } from 'react';
import { LanguageItem } from '../../types';
import { API_BASE, parseApiError } from '../../utils/api';
import {
  Languages,
  Search,
  Plus,
  Check,
  AlertTriangle,
  RotateCcw,
  Sparkles,
  X,
  Loader2,
  Info,
} from 'lucide-react';

interface LanguageManagerProps {
  isOpen: boolean;
  onClose: () => void;
  indexedCountryCodes?: string[];
  onTriggerReindex?: (countries: string[]) => void;
}

export const LanguageManager: React.FC<LanguageManagerProps> = ({
  isOpen,
  onClose,
  indexedCountryCodes = [],
  onTriggerReindex,
}) => {
  const [languages, setLanguages] = useState<LanguageItem[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [newCode, setNewCode] = useState('');
  const [newName, setNewName] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const fetchLanguages = async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/languages`);
      if (res.ok) {
        const data = await res.json();
        if (data.languages && Array.isArray(data.languages)) {
          setLanguages(data.languages);
        }
      }
    } catch (err) {
      console.error('Failed to load languages:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) {
      fetchLanguages();
    }
  }, [isOpen]);

  const handleToggle = async (code: string, currentEnabled: boolean) => {
    setSaving(true);
    setStatusMessage(null);
    try {
      const res = await fetch(`${API_BASE}/languages`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code,
          enabled: !currentEnabled,
        }),
      });

      if (!res.ok) {
        const errInfo = await parseApiError(res);
        throw new Error(errInfo.message);
      }

      setLanguages((prev) =>
        prev.map((l) => (l.code.toLowerCase() === code.toLowerCase() ? { ...l, enabled: !currentEnabled } : l))
      );
      setHasChanges(true);
      setStatusMessage(`Language '${code.toUpperCase()}' is now ${!currentEnabled ? 'enabled' : 'disabled'}.`);
    } catch (err: any) {
      setStatusMessage(`Error updating language: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleAddLanguage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCode.trim()) return;

    setSaving(true);
    setStatusMessage(null);
    try {
      const res = await fetch(`${API_BASE}/languages`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code: newCode.trim().toLowerCase(),
          name: newName.trim() || newCode.trim().toUpperCase(),
          enabled: true,
        }),
      });

      if (!res.ok) {
        const errInfo = await parseApiError(res);
        throw new Error(errInfo.message);
      }

      const data = await res.json();
      if (data.languages) {
        setLanguages(data.languages);
      }
      setNewCode('');
      setNewName('');
      setShowAddForm(false);
      setHasChanges(true);
      setStatusMessage(`Added and enabled language '${newCode.trim().toUpperCase()}'.`);
    } catch (err: any) {
      setStatusMessage(`Error adding language: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleReindex = () => {
    if (onTriggerReindex && indexedCountryCodes.length > 0) {
      onTriggerReindex(indexedCountryCodes);
      setHasChanges(false);
      setStatusMessage(`Re-indexing triggered for: ${indexedCountryCodes.join(', ')}.`);
    }
  };

  if (!isOpen) return null;

  const filtered = languages.filter(
    (l) =>
      l.name.toLowerCase().includes(search.toLowerCase()) ||
      l.code.toLowerCase().includes(search.toLowerCase())
  );

  const enabledCount = languages.filter((l) => l.enabled).length;
  const disabledCount = languages.length - enabledCount;

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-2xl max-h-[85vh] shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="p-5 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-brand-500/10 text-brand-400 border border-brand-500/30">
              <Languages className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-100 flex items-center gap-2">
                Multilingual Whitelist & Extraction
              </h2>
              <p className="text-xs text-slate-400 mt-0.5">
                Control which language translations (`name:*`) are extracted from OSM into Meilisearch.
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Reprocess Alert Notice */}
        {hasChanges && indexedCountryCodes.length > 0 && (
          <div className="mx-5 mt-4 p-3.5 bg-amber-500/10 border border-amber-500/30 rounded-xl flex items-start justify-between gap-3 text-xs">
            <div className="flex items-start gap-2.5">
              <AlertTriangle className="w-4 h-4 text-amber-400 flex-shrink-0 mt-0.5" />
              <div>
                <span className="font-semibold text-amber-200 block">
                  Language configuration modified
                </span>
                <span className="text-amber-300/80 mt-0.5 block leading-relaxed">
                  Existing indexed countries need re-processing to extract newly enabled translations from cached OSM extracts into Meilisearch.
                </span>
              </div>
            </div>
            <button
              onClick={handleReindex}
              className="px-3 py-1.5 bg-amber-500 hover:bg-amber-600 text-slate-950 font-semibold rounded-lg flex items-center gap-1.5 transition flex-shrink-0 shadow-sm"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              Re-process {indexedCountryCodes.length} Countries
            </button>
          </div>
        )}

        {statusMessage && (
          <div className="mx-5 mt-3 px-3 py-2 bg-slate-950 border border-slate-800 rounded-lg text-xs text-brand-300 font-mono">
            {statusMessage}
          </div>
        )}

        {/* Controls Bar */}
        <div className="p-5 pb-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-500 absolute left-3 top-2.5" />
            <input
              type="text"
              placeholder="Filter by code or language (e.g. 'es', 'Japanese', 'Arabic')..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-9 pr-3 py-2 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-brand-500 transition"
            />
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-400">
              <strong className="text-emerald-400 font-mono">{enabledCount}</strong> enabled ·{' '}
              <span className="text-slate-500 font-mono">{disabledCount}</span> disabled
            </span>

            <button
              onClick={() => setShowAddForm(!showAddForm)}
              className="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 rounded-lg text-xs font-medium border border-slate-700/80 transition flex items-center gap-1.5"
            >
              <Plus className="w-3.5 h-3.5 text-brand-400" />
              Add Language
            </button>
          </div>
        </div>

        {/* Inline Add Language Form */}
        {showAddForm && (
          <form
            onSubmit={handleAddLanguage}
            className="mx-5 mb-3 p-3 bg-slate-950 border border-slate-800 rounded-xl flex flex-wrap items-center gap-2.5"
          >
            <div className="w-24">
              <input
                type="text"
                placeholder="Code (e.g. th)"
                value={newCode}
                onChange={(e) => setNewCode(e.target.value)}
                maxLength={5}
                required
                className="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-brand-500"
              />
            </div>
            <div className="flex-1 min-w-[140px]">
              <input
                type="text"
                placeholder="Name (e.g. Thai)"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="w-full bg-slate-900 border border-slate-700 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-brand-500"
              />
            </div>
            <button
              type="submit"
              disabled={saving || !newCode.trim()}
              className="px-3 py-1.5 bg-brand-600 hover:bg-brand-500 text-white rounded-lg text-xs font-semibold transition disabled:opacity-50"
            >
              Save & Enable
            </button>
            <button
              type="button"
              onClick={() => setShowAddForm(false)}
              className="px-2.5 py-1.5 text-slate-400 hover:text-slate-200 text-xs"
            >
              Cancel
            </button>
          </form>
        )}

        {/* Language Grid */}
        <div className="flex-1 overflow-y-auto px-5 py-2">
          {loading ? (
            <div className="py-12 flex flex-col items-center justify-center gap-2 text-slate-500 text-xs">
              <Loader2 className="w-6 h-6 animate-spin text-brand-400" />
              Loading languages...
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-12 text-center text-slate-500 text-xs">
              No languages matching "{search}"
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5">
              {filtered.map((lang) => (
                <div
                  key={lang.code}
                  className={`p-2.5 rounded-xl border transition flex items-center justify-between gap-2.5 ${
                    lang.enabled
                      ? 'bg-slate-950/80 border-slate-800 shadow-sm'
                      : 'bg-slate-950/40 border-slate-800/60 opacity-60'
                  }`}
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <span
                      className={`px-2 py-0.5 rounded text-[11px] font-mono font-bold uppercase ${
                        lang.enabled
                          ? 'bg-brand-500/15 text-brand-400 border border-brand-500/30'
                          : 'bg-slate-800 text-slate-500 border border-slate-700'
                      }`}
                    >
                      {lang.code}
                    </span>
                    <span className="text-xs font-medium text-slate-200 truncate">
                      {lang.name}
                    </span>
                  </div>

                  <button
                    onClick={() => handleToggle(lang.code, lang.enabled)}
                    disabled={saving}
                    title={lang.enabled ? 'Click to disable' : 'Click to enable'}
                    className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                      lang.enabled ? 'bg-brand-600' : 'bg-slate-800'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition duration-200 ease-in-out ${
                        lang.enabled ? 'translate-x-4' : 'translate-x-0'
                      }`}
                    />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-800 bg-slate-950/60 flex items-center justify-between text-xs text-slate-400">
          <div className="flex items-center gap-1.5 text-[11px] text-slate-500">
            <Info className="w-3.5 h-3.5" />
            <span>Configured in config/languages.yaml · Cached in raw/osm PBF files</span>
          </div>
          <button
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium transition"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
};
