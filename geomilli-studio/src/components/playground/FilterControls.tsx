import React, { useState, useEffect } from 'react';
import { ExplorerMode, LanguageItem } from '../../types';
import { Search, Navigation, Compass, MapPin, Plane, Layers, Settings2, X } from 'lucide-react';
import { API_BASE } from '../../utils/api';

interface FilterControlsProps {
  mode: ExplorerMode;
  onModeChange: (mode: ExplorerMode) => void;
  query: string;
  onQueryChange: (q: string) => void;
  selectedClass: string;
  onClassChange: (cls: string) => void;
  selectedLang: string;
  onLangChange: (lang: string) => void;
  country: string;
  onCountryChange: (c: string) => void;
  radiusMeters: number;
  onRadiusChange: (r: number) => void;
  cuisine: string;
  onCuisineChange: (c: string) => void;
  minStars: number;
  onMinStarsChange: (s: number) => void;
  onExecute: () => void;
  loading: boolean;
  onOpenLanguageManager?: () => void;
}

const DEFAULT_TAXONOMY_CLASSES = [
  'All',
  'City',
  'TouristAttraction',
  'Restaurant',
  'Hotel',
  'HistoricSite',
  'Park',
  'TrainStation',
  'Airport',
  'Shopping',
  'NaturalFeature',
  'Entertainment',
  'Beach',
  'BusStation',
  'Stadium',
  'Neighborhood',
  'Region',
  'Country',
  'Island',
  'FerryTerminal',
];

const DEFAULT_LANGUAGES: LanguageItem[] = [
  { code: 'en', name: 'English', enabled: true },
  { code: 'nl', name: 'Dutch', enabled: true },
  { code: 'fr', name: 'French', enabled: true },
  { code: 'de', name: 'German', enabled: true },
  { code: 'es', name: 'Spanish', enabled: true },
];

export const FilterControls: React.FC<FilterControlsProps> = ({
  mode,
  onModeChange,
  query,
  onQueryChange,
  selectedClass,
  onClassChange,
  selectedLang,
  onLangChange,
  country,
  onCountryChange,
  radiusMeters,
  onRadiusChange,
  cuisine,
  onCuisineChange,
  minStars,
  onMinStarsChange,
  onExecute,
  loading,
  onOpenLanguageManager,
}) => {
  const [languages, setLanguages] = useState<LanguageItem[]>(DEFAULT_LANGUAGES);
  const [taxonomyClasses, setTaxonomyClasses] = useState<string[]>(DEFAULT_TAXONOMY_CLASSES);

  useEffect(() => {
    fetch(`${API_BASE}/languages`)
      .then((res) => res.json())
      .then((data) => {
        if (data.languages && Array.isArray(data.languages)) {
          setLanguages(data.languages);
        }
      })
      .catch(() => {});

    fetch(`${API_BASE}/taxonomy`)
      .then((res) => res.json())
      .then((data) => {
        if (data.classes && Array.isArray(data.classes)) {
          const names = data.classes.map((c: any) => c.name);
          setTaxonomyClasses(['All', ...names]);
        }
      })
      .catch(() => {});
  }, []);

  const enabledLanguages = languages.filter((l) => l.enabled);
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex flex-col gap-3 shadow-sm">
      {/* Modes Navigation */}
      <div className="flex flex-wrap items-center gap-1.5 p-1 bg-slate-950 rounded-lg border border-slate-800">
        <button
          onClick={() => onModeChange('autocomplete')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'autocomplete'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <Search className="w-3.5 h-3.5" />
          Autocomplete
        </button>
        <button
          onClick={() => onModeChange('search')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'search'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <Layers className="w-3.5 h-3.5" />
          Search & Facets
        </button>
        <button
          onClick={() => onModeChange('nearby')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'nearby'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <Compass className="w-3.5 h-3.5" />
          Nearby POIs
        </button>
        <button
          onClick={() => onModeChange('reverse')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'reverse'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <MapPin className="w-3.5 h-3.5" />
          Reverse Geocode
        </button>
        <button
          onClick={() => onModeChange('bbox')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'bbox'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <Navigation className="w-3.5 h-3.5" />
          Bounding Box
        </button>
        <button
          onClick={() => onModeChange('airports')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition ${
            mode === 'airports'
              ? 'bg-brand-600 text-white shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'
          }`}
        >
          <Plane className="w-3.5 h-3.5" />
          Airports
        </button>
      </div>

      {/* Primary Query Input */}
      {mode !== 'reverse' && mode !== 'bbox' && (
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={query}
              onChange={(e) => onQueryChange(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && onExecute()}
              placeholder={
                mode === 'autocomplete'
                  ? 'Type prefix (e.g. "Brux", "Ghent", "Atom", "Hilton")...'
                  : mode === 'airports'
                  ? 'Search airport name, IATA (BRU, ANR, CRL) or ICAO...'
                  : 'Search places, monuments, restaurants, museums...'
              }
              className="w-full bg-slate-950 border border-slate-700/80 rounded-lg pl-9 pr-9 py-2 text-sm text-slate-100 placeholder-slate-500 focus:outline-none focus:border-brand-500"
            />
            {query && (
              <button
                type="button"
                onClick={() => onQueryChange('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 p-0.5 rounded transition"
                title="Clear search query"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
          <button
            onClick={onExecute}
            disabled={loading}
            className="px-4 py-2 bg-brand-600 hover:bg-brand-500 disabled:bg-slate-800 text-white text-sm font-medium rounded-lg transition shadow-sm flex items-center gap-1.5"
          >
            {loading ? (
              <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            ) : (
              'Query'
            )}
          </button>
        </div>
      )}

      {/* Secondary Filters Bar */}
      <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2 text-xs">
        {/* Language */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-slate-400 text-[10px] uppercase font-semibold">
              Language
            </label>
            {onOpenLanguageManager && (
              <button
                type="button"
                onClick={onOpenLanguageManager}
                title="Manage language whitelist & extraction"
                className="text-[10px] text-brand-400 hover:text-brand-300 flex items-center gap-0.5"
              >
                <Settings2 className="w-2.5 h-2.5" />
                Manage
              </button>
            )}
          </div>
          <select
            value={selectedLang}
            onChange={(e) => onLangChange(e.target.value)}
            className="w-full bg-slate-950 border border-slate-700/80 rounded px-2 py-1.5 text-slate-200 focus:outline-none focus:border-brand-500 font-mono text-xs"
          >
            <option value="">Default (Ground)</option>
            {enabledLanguages.map((l) => (
              <option key={l.code} value={l.code}>
                {l.name} ({l.code})
              </option>
            ))}
          </select>
        </div>

        {/* Country */}
        <div>
          <label className="block text-slate-400 text-[10px] uppercase font-semibold mb-1">
            Country Code
          </label>
          <input
            type="text"
            value={country}
            onChange={(e) => onCountryChange(e.target.value.toUpperCase())}
            placeholder="e.g. BE, NL, FR"
            maxLength={3}
            className="w-full bg-slate-950 border border-slate-700/80 rounded px-2 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-brand-500"
          />
        </div>

        {/* Nearby Radius */}
        {mode === 'nearby' && (
          <div>
            <label className="block text-slate-400 text-[10px] uppercase font-semibold mb-1">
              Radius (m)
            </label>
            <select
              value={radiusMeters}
              onChange={(e) => onRadiusChange(Number(e.target.value))}
              className="w-full bg-slate-950 border border-slate-700/80 rounded px-2 py-1.5 text-slate-200 focus:outline-none focus:border-brand-500"
            >
              <option value={500}>500 m</option>
              <option value={1000}>1,000 m (1km)</option>
              <option value={3000}>3,000 m (3km)</option>
              <option value={5000}>5,000 m (5km)</option>
              <option value={10000}>10,000 m (10km)</option>
              <option value={25000}>25,000 m (25km)</option>
            </select>
          </div>
        )}

        {/* Cuisine Filter (Restaurants) */}
        {(selectedClass === 'Restaurant' || mode === 'search') && (
          <div>
            <label className="block text-slate-400 text-[10px] uppercase font-semibold mb-1">
              Cuisine
            </label>
            <input
              type="text"
              value={cuisine}
              onChange={(e) => onCuisineChange(e.target.value)}
              placeholder="e.g. italian, belgian"
              className="w-full bg-slate-950 border border-slate-700/80 rounded px-2 py-1.5 text-slate-200 placeholder-slate-600 focus:outline-none focus:border-brand-500"
            />
          </div>
        )}

        {/* Star Rating (Hotels) */}
        {(selectedClass === 'Hotel' || mode === 'search') && (
          <div>
            <label className="block text-slate-400 text-[10px] uppercase font-semibold mb-1">
              Min Stars
            </label>
            <select
              value={minStars}
              onChange={(e) => onMinStarsChange(Number(e.target.value))}
              className="w-full bg-slate-950 border border-slate-700/80 rounded px-2 py-1.5 text-slate-200 focus:outline-none focus:border-brand-500"
            >
              <option value={0}>Any Rating</option>
              <option value={3}>3+ Stars ★★★</option>
              <option value={4}>4+ Stars ★★★★</option>
              <option value={5}>5 Stars ★★★★★</option>
            </select>
          </div>
        )}
      </div>

      {/* Class Pills */}
      <div>
        <div className="text-[10px] uppercase tracking-wider text-slate-400 font-semibold mb-1.5 flex items-center justify-between">
          <span>Taxonomy Class Filter</span>
          {selectedClass !== 'All' && (
            <button
              onClick={() => onClassChange('All')}
              className="text-brand-400 hover:text-brand-300 text-[10px] lowercase"
            >
              reset to all
            </button>
          )}
        </div>
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 no-scrollbar">
          {taxonomyClasses.map((cls) => (
            <button
              key={cls}
              onClick={() => onClassChange(cls)}
              className={`px-2.5 py-1 rounded-full text-xs whitespace-nowrap transition border ${
                selectedClass === cls
                  ? 'bg-brand-500/20 text-brand-300 border-brand-500/60 font-medium'
                  : 'bg-slate-950 text-slate-400 border-slate-800 hover:border-slate-700 hover:text-slate-300'
              }`}
            >
              {cls}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};
