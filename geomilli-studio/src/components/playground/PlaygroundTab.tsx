import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Place, SearchResponse, ExplorerMode } from '../../types';
import { API_BASE, fetchWithTiming } from '../../utils/api';
import { FilterControls } from './FilterControls';
import { MapExplorer } from './MapExplorer';
import { ResultCards } from './ResultCards';
import { DevInspector } from './DevInspector';
import { LanguageManager } from '../importer/LanguageManager';

export const PlaygroundTab: React.FC = () => {
  const [mode, setMode] = useState<ExplorerMode>('autocomplete');
  const [query, setQuery] = useState('Brux');
  const [selectedClass, setSelectedClass] = useState('All');
  const [selectedLang, setSelectedLang] = useState('');
  const [country, setCountry] = useState('BE');
  const [radiusMeters, setRadiusMeters] = useState(5000);
  const [cuisine, setCuisine] = useState('');
  const [minStars, setMinStars] = useState(0);
  const [isLangManagerOpen, setIsLangManagerOpen] = useState(false);

  // Pin & Bbox
  const [pinLocation, setPinLocation] = useState<{ lat: number; lng: number }>({
    lat: 50.8503,
    lng: 4.3517, // Brussels
  });
  const [bbox, setBbox] = useState<[number, number, number, number] | null>(null);
  const [syncViewport, setSyncViewport] = useState(false);
  const bboxDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Results & inspector
  const [hits, setHits] = useState<Place[]>([]);
  const [totalHits, setTotalHits] = useState<number | undefined>(undefined);
  const [selectedPlace, setSelectedPlace] = useState<Place | null>(null);
  const [loading, setLoading] = useState(false);
  const [durationMs, setDurationMs] = useState<number | undefined>(undefined);
  const [meiliDurationMs, setMeiliDurationMs] = useState<number | undefined>(undefined);
  const [rawResponse, setRawResponse] = useState<any>(null);
  const [curlCommand, setCurlCommand] = useState('');

  // Construct URL and cURL for current parameters with optional overrides
  const buildQueryUrl = useCallback(
    (overrides?: { pin?: { lat: number; lng: number }; bbox?: [number, number, number, number] }) => {
      let endpoint = '/autocomplete';
      const params = new URLSearchParams();
      const effectivePin = overrides?.pin || pinLocation;
      const effectiveBbox = overrides?.bbox || bbox;

      if (mode === 'autocomplete') {
        endpoint = '/autocomplete';
        if (query) params.set('q', query);
        if (selectedLang) params.set('lang', selectedLang);
        if (country) params.set('country', country);
        if (selectedClass !== 'All') params.set('class', selectedClass);
        if (effectivePin) {
          params.set('lat', effectivePin.lat.toString());
          params.set('lng', effectivePin.lng.toString());
        }
        params.set('limit', '20');
      } else if (mode === 'search') {
        endpoint = '/search';
        if (query) params.set('q', query);
        if (selectedLang) params.set('lang', selectedLang);
        if (country) params.set('country', country);
        if (selectedClass !== 'All') params.set('class', selectedClass);
        if (cuisine) params.set('cuisine', cuisine);
        if (minStars > 0) params.set('star_rating', minStars.toString());
        params.set('limit', '20');
      } else if (mode === 'reverse') {
        endpoint = '/reverse';
        params.set('lat', effectivePin.lat.toString());
        params.set('lng', effectivePin.lng.toString());
        if (selectedLang) params.set('lang', selectedLang);
      } else if (mode === 'nearby') {
        endpoint = '/nearby';
        params.set('lat', effectivePin.lat.toString());
        params.set('lng', effectivePin.lng.toString());
        params.set('radius_meters', radiusMeters.toString());
        if (selectedClass !== 'All') params.set('class', selectedClass);
        if (selectedLang) params.set('lang', selectedLang);
        params.set('limit', '25');
      } else if (mode === 'bbox') {
        endpoint = '/bbox';
        if (effectiveBbox) {
          params.set('min_lat', effectiveBbox[0].toString());
          params.set('min_lng', effectiveBbox[1].toString());
          params.set('max_lat', effectiveBbox[2].toString());
          params.set('max_lng', effectiveBbox[3].toString());
        } else {
          params.set('min_lat', '50.75');
          params.set('min_lng', '4.25');
          params.set('max_lat', '50.95');
          params.set('max_lng', '4.45');
        }
        if (selectedClass !== 'All') params.set('class', selectedClass);
        if (selectedLang) params.set('lang', selectedLang);
        params.set('limit', '50');
      } else if (mode === 'airports') {
        endpoint = '/airports';
        if (query) params.set('q', query);
        if (country) params.set('country', country);
        if (selectedLang) params.set('lang', selectedLang);
        params.set('limit', '25');
      }

      const queryString = params.toString() ? `?${params.toString()}` : '';
      const fullUrl = `${API_BASE}${endpoint}${queryString}`;
      return { fullUrl, endpoint, params };
    },
    [mode, query, selectedClass, selectedLang, country, radiusMeters, cuisine, minStars, pinLocation, bbox]
  );

  const executeQuery = useCallback(
    async (overrides?: { pin?: { lat: number; lng: number }; bbox?: [number, number, number, number] }) => {
      const { fullUrl } = buildQueryUrl(overrides);
      const apiHost = import.meta.env.VITE_API_EXTERNAL_URL || 'http://localhost:8080';
      const targetUrl = fullUrl.startsWith('http') ? fullUrl : `${apiHost}${fullUrl}`;
      const curl = `curl -s "${targetUrl}" | jq .`;
      setCurlCommand(curl);
      setLoading(true);

      try {
        const { data, durationMs } = await fetchWithTiming<any>(fullUrl);
        setDurationMs(durationMs);
        setRawResponse(data);

        if (Array.isArray(data)) {
          setHits(data);
          setTotalHits(data.length);
          setMeiliDurationMs(undefined);
        } else if (data && typeof data === 'object') {
          if (Array.isArray(data.results)) {
            setHits(data.results);
            setTotalHits(data.total_hits ?? data.totalHits ?? data.results.length);
            setMeiliDurationMs(data.took_ms ?? data.processingTimeMs);
          } else if (Array.isArray(data.hits)) {
            setHits(data.hits);
            setTotalHits(data.totalHits ?? data.total_hits ?? data.estimatedTotalHits ?? data.hits.length);
            setMeiliDurationMs(data.processingTimeMs ?? data.took_ms);
          } else if (Array.isArray(data.nearest_places) || data.closest_place) {
            const list: Place[] = [];
            if (data.closest_place) {
              list.push(data.closest_place);
            }
            if (Array.isArray(data.nearest_places)) {
              data.nearest_places.forEach((p: Place) => {
                if (!list.some((existing) => existing.id === p.id)) {
                  list.push(p);
                }
              });
            }
            setHits(list);
            setTotalHits(list.length);
            setMeiliDurationMs(data.took_ms);
          } else if (data.id) {
            setHits([data as Place]);
            setTotalHits(1);
            setMeiliDurationMs(undefined);
          } else {
            setHits([]);
            setTotalHits(0);
          }
        }
      } catch (err: any) {
        console.error('API query error:', err);
        setHits([]);
        setTotalHits(0);
        setDurationMs(undefined);
        setMeiliDurationMs(undefined);
        setRawResponse({
          error: true,
          message: err.message || 'Request failed',
        });
      } finally {
        setLoading(false);
      }
    },
    [buildQueryUrl]
  );

  // Trigger query on parameter changes with debounce
  useEffect(() => {
    const timer = setTimeout(() => {
      executeQuery();
    }, 250);
    return () => clearTimeout(timer);
  }, [mode, query, selectedClass, selectedLang, country, radiusMeters, cuisine, minStars]);

  // Handle map pin set: immediately execute if nearby or reverse mode
  const handleSetPin = useCallback(
    (coords: { lat: number; lng: number }) => {
      setPinLocation(coords);
      if (mode === 'nearby' || mode === 'reverse') {
        executeQuery({ pin: coords });
      }
    },
    [mode, executeQuery]
  );

  // Handle bbox change from map movement: debounce and execute if bbox mode with syncViewport
  const handleBBoxChange = useCallback(
    (newBbox: [number, number, number, number]) => {
      setBbox(newBbox);
      if (mode === 'bbox' && syncViewport) {
        if (bboxDebounceRef.current) {
          clearTimeout(bboxDebounceRef.current);
        }
        bboxDebounceRef.current = setTimeout(() => {
          executeQuery({ bbox: newBbox });
        }, 300);
      }
    },
    [mode, syncViewport, executeQuery]
  );

  useEffect(() => {
    return () => {
      if (bboxDebounceRef.current) {
        clearTimeout(bboxDebounceRef.current);
      }
    };
  }, []);

  return (
    <div className="flex flex-col gap-4">
      {/* Top Filter Controls */}
      <FilterControls
        mode={mode}
        onModeChange={(m) => {
          setMode(m);
          if (m === 'airports' && (query === 'Brux' || query === 'belgium')) {
            setQuery('');
          }
          if (m === 'reverse') setQuery('');
        }}
        query={query}
        onQueryChange={setQuery}
        selectedClass={selectedClass}
        onClassChange={(cls) => {
          setSelectedClass(cls);
          if (cls !== 'All' && (query === 'Brux' || query === 'belgium')) {
            setQuery('');
          }
        }}
        selectedLang={selectedLang}
        onLangChange={setSelectedLang}
        country={country}
        onCountryChange={setCountry}
        radiusMeters={radiusMeters}
        onRadiusChange={setRadiusMeters}
        cuisine={cuisine}
        onCuisineChange={setCuisine}
        minStars={minStars}
        onMinStarsChange={setMinStars}
        onExecute={() => executeQuery()}
        loading={loading}
        onOpenLanguageManager={() => setIsLangManagerOpen(true)}
      />

      {/* Main Grid: Left side Map & Results, Right side Inspector */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* Left Column: Interactive Map (7 cols) */}
        <div className="lg:col-span-7 flex flex-col gap-2">
          <div className="h-[420px] w-full">
            <MapExplorer
              hits={hits}
              selectedPlace={selectedPlace}
              onSelectPlace={(p) => setSelectedPlace(p)}
              pinLocation={pinLocation}
              onSetPin={handleSetPin}
              onBBoxChange={handleBBoxChange}
              syncViewport={syncViewport}
              mode={mode}
            />
          </div>

          {mode === 'bbox' && (
            <div className="flex items-center justify-between px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg text-xs text-slate-300">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={syncViewport}
                  onChange={(e) => setSyncViewport(e.target.checked)}
                  className="rounded text-brand-500 focus:ring-0 bg-slate-950 border-slate-700"
                />
                <span>Sync places on map pan/zoom</span>
              </label>
              <button
                onClick={() => executeQuery()}
                className="px-2 py-0.5 bg-brand-600 hover:bg-brand-500 rounded text-white text-[11px]"
              >
                Refresh Viewport
              </button>
            </div>
          )}
        </div>

        {/* Right Column: Place Results List (5 cols) */}
        <div className="lg:col-span-5 flex flex-col gap-3">
          <ResultCards
            hits={hits}
            selectedPlace={selectedPlace}
            onSelectPlace={(p) => setSelectedPlace(p)}
            loading={loading}
            totalHits={totalHits}
          />
        </div>
      </div>

      {/* Bottom Developer Inspector */}
      <DevInspector
        curlCommand={curlCommand}
        responseJson={rawResponse}
        durationMs={durationMs}
        meiliDurationMs={meiliDurationMs}
      />

      {/* Language Whitelist Modal */}
      <LanguageManager
        isOpen={isLangManagerOpen}
        onClose={() => setIsLangManagerOpen(false)}
      />
    </div>
  );
};
