import React, { useEffect, useRef } from 'react';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { Place } from '../../types';

interface MapExplorerProps {
  hits: Place[];
  selectedPlace: Place | null;
  onSelectPlace: (place: Place) => void;
  pinLocation: { lat: number; lng: number } | null;
  onSetPin: (coords: { lat: number; lng: number }) => void;
  onBBoxChange?: (bbox: [number, number, number, number]) => void;
  syncViewport: boolean;
  mode: string;
}

const VECTOR_STYLE_URL = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';

const DARK_RASTER_STYLE: maplibregl.StyleSpecification = {
  version: 8,
  sources: {
    'carto-dark-raster': {
      type: 'raster',
      tiles: [
        'https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
        'https://b.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
        'https://c.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
        'https://d.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png',
      ],
      tileSize: 256,
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
    },
  },
  layers: [
    {
      id: 'carto-dark-raster-layer',
      type: 'raster',
      source: 'carto-dark-raster',
      minzoom: 0,
      maxzoom: 19,
    },
  ],
};

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function getClassBadgeColor(placeClass: string) {
  switch (placeClass?.toLowerCase()) {
    case 'airport':
      return 'bg-amber-500';
    case 'hotel':
      return 'bg-purple-500';
    case 'restaurant':
      return 'bg-emerald-500';
    case 'trainstation':
      return 'bg-blue-500';
    case 'touristattraction':
    case 'historicsite':
      return 'bg-rose-500';
    case 'city':
    case 'region':
      return 'bg-cyan-500';
    case 'shopping':
      return 'bg-pink-500';
    default:
      return 'bg-sky-400';
  }
}

export const MapExplorer: React.FC<MapExplorerProps> = ({
  hits,
  selectedPlace,
  onSelectPlace,
  pinLocation,
  onSetPin,
  onBBoxChange,
  syncViewport,
  mode,
}) => {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<Map<string, { marker: maplibregl.Marker; popup: maplibregl.Popup }>>(new Map());
  const pinMarkerRef = useRef<maplibregl.Marker | null>(null);

  // Initialize Map with vector style and robust fallback to raster
  useEffect(() => {
    if (!mapContainer.current || map.current) return;

    let hasFallbackTriggered = false;

    const mapInstance = new maplibregl.Map({
      container: mapContainer.current,
      style: VECTOR_STYLE_URL,
      center: [4.3517, 50.8503], // Default Brussels, Belgium
      zoom: 11,
      attributionControl: { compact: true },
    });

    const applyFallback = () => {
      if (hasFallbackTriggered) return;
      hasFallbackTriggered = true;
      console.warn('Vector tile style failed or timed out. Falling back to dark raster tiles.');
      try {
        mapInstance.setStyle(DARK_RASTER_STYLE);
      } catch (err) {
        console.error('Failed to set fallback raster style:', err);
      }
    };

    mapInstance.on('error', (e) => {
      // If error occurs before style is loaded, fallback to raster
      if (!mapInstance.isStyleLoaded()) {
        applyFallback();
      }
    });

    // Guard with a 3.5s timeout if external vector style is blocked or hangs
    const styleTimeout = setTimeout(() => {
      if (!mapInstance.isStyleLoaded()) {
        applyFallback();
      }
    }, 3500);

    mapInstance.once('load', () => {
      clearTimeout(styleTimeout);
      mapInstance.resize();
    });

    mapInstance.addControl(new maplibregl.NavigationControl({ visualizePitch: true }), 'top-right');
    mapInstance.addControl(new maplibregl.ScaleControl({ unit: 'metric' }), 'bottom-left');

    mapInstance.on('click', (e) => {
      const lat = parseFloat(e.lngLat.lat.toFixed(5));
      const lng = parseFloat(e.lngLat.lng.toFixed(5));
      onSetPin({ lat, lng });
    });

    mapInstance.on('moveend', () => {
      if (onBBoxChange) {
        const bounds = mapInstance.getBounds();
        const minLat = parseFloat(bounds.getSouth().toFixed(5));
        const minLng = parseFloat(bounds.getWest().toFixed(5));
        const maxLat = parseFloat(bounds.getNorth().toFixed(5));
        const maxLng = parseFloat(bounds.getEast().toFixed(5));
        onBBoxChange([minLat, minLng, maxLat, maxLng]);
      }
    });

    map.current = mapInstance;

    // ResizeObserver to ensure map canvas is always resized accurately
    const resizeObserver = new ResizeObserver(() => {
      if (map.current) {
        map.current.resize();
      }
    });

    if (mapContainer.current) {
      resizeObserver.observe(mapContainer.current);
    }

    return () => {
      clearTimeout(styleTimeout);
      resizeObserver.disconnect();
      mapInstance.remove();
      map.current = null;
    };
  }, []);

  // Update pin marker
  useEffect(() => {
    if (!map.current) return;

    if (pinMarkerRef.current) {
      pinMarkerRef.current.remove();
      pinMarkerRef.current = null;
    }

    if (pinLocation) {
      const el = document.createElement('div');
      el.className =
        'w-6 h-6 bg-rose-500 border-2 border-white rounded-full shadow-lg flex items-center justify-center animate-bounce cursor-pointer';
      el.innerHTML = '<div class="w-2 h-2 bg-white rounded-full"></div>';

      const popup = new maplibregl.Popup({ offset: 14, closeButton: false }).setHTML(`
        <div class="text-[11px] font-sans font-medium text-slate-800 p-1">
          📍 Pin Location<br/>
          <span class="font-mono text-[10px] text-slate-500">${pinLocation.lat}, ${pinLocation.lng}</span>
        </div>
      `);

      const marker = new maplibregl.Marker({ element: el })
        .setLngLat([pinLocation.lng, pinLocation.lat])
        .setPopup(popup)
        .addTo(map.current);

      pinMarkerRef.current = marker;
    }
  }, [pinLocation]);

  // Update result hit markers
  useEffect(() => {
    if (!map.current) return;

    // Clear old markers
    markersRef.current.forEach(({ marker }) => marker.remove());
    markersRef.current.clear();

    hits.forEach((hit) => {
      const lat = hit._geo?.lat ?? hit.lat;
      const lng = hit._geo?.lng ?? hit.lng;
      if (typeof lat !== 'number' || typeof lng !== 'number' || isNaN(lat) || isNaN(lng)) return;

      const isSelected = selectedPlace?.id === hit.id;
      const name = hit.name_display || hit.name || '<Unnamed Place>';
      const placeClass = hit.class || 'Place';
      const placeSubclass = hit.subclass;
      const adminInfo = hit.admin_hierarchy?.[0] || hit.city || hit.state || hit.country_name || '';
      const countryCode = hit.country_code || '';
      const badgeColor = getClassBadgeColor(placeClass);

      const el = document.createElement('div');
      el.className = `cursor-pointer transition-all duration-200 ${
        isSelected
          ? 'w-7 h-7 bg-brand-500 border-2 border-white rounded-full shadow-2xl flex items-center justify-center ring-4 ring-brand-400/40 z-30 scale-125'
          : `w-4 h-4 ${badgeColor} border-2 border-slate-900 rounded-full shadow-md hover:scale-150 z-10 hover:border-white`
      }`;

      if (isSelected) {
        el.innerHTML = '<div class="w-2 h-2 bg-white rounded-full animate-ping"></div>';
      }

      const popup = new maplibregl.Popup({ offset: 14, closeButton: false }).setHTML(`
        <div class="p-2 text-slate-900 font-sans min-w-[160px] max-w-[240px]">
          <div class="font-bold text-xs text-slate-950 leading-tight mb-1">${escapeHtml(name)}</div>
          <div class="flex items-center gap-1.5 mb-1.5 flex-wrap">
            <span class="px-1.5 py-0.5 rounded text-[10px] font-semibold bg-indigo-50 text-indigo-700 border border-indigo-200">
              ${escapeHtml(placeClass)}
            </span>
            ${placeSubclass ? `<span class="text-[10px] text-slate-500 font-medium">· ${escapeHtml(placeSubclass)}</span>` : ''}
          </div>
          ${
            adminInfo || countryCode
              ? `
            <div class="text-[10px] text-slate-600 flex items-center justify-between border-t border-slate-200 pt-1 mt-1">
              <span class="truncate max-w-[150px]">${escapeHtml(adminInfo)}</span>
              ${countryCode ? `<span class="font-mono text-[9px] px-1 py-0.2 bg-slate-100 rounded text-slate-700 font-medium">${escapeHtml(countryCode)}</span>` : ''}
            </div>
          `
              : ''
          }
        </div>
      `);

      const marker = new maplibregl.Marker({ element: el })
        .setLngLat([lng, lat])
        .setPopup(popup)
        .addTo(map.current!);

      el.addEventListener('click', (e) => {
        e.stopPropagation();
        onSelectPlace(hit);
      });

      markersRef.current.set(hit.id, { marker, popup });
    });

    // Auto fit bounds if hits changed and not in bbox mode
    if (hits.length > 0 && mode !== 'bbox' && !syncViewport && map.current) {
      const bounds = new maplibregl.LngLatBounds();
      let count = 0;
      hits.slice(0, 30).forEach((hit) => {
        const lat = hit._geo?.lat ?? hit.lat;
        const lng = hit._geo?.lng ?? hit.lng;
        if (typeof lat === 'number' && typeof lng === 'number' && !isNaN(lat) && !isNaN(lng)) {
          bounds.extend([lng, lat]);
          count++;
        }
      });
      if (count > 0) {
        map.current.fitBounds(bounds, { padding: 60, maxZoom: 15, duration: 800 });
      }
    }
  }, [hits, selectedPlace, mode, syncViewport]);

  // Center on selected place & toggle popup
  useEffect(() => {
    if (!map.current || !selectedPlace) return;
    const lat = selectedPlace._geo?.lat ?? selectedPlace.lat;
    const lng = selectedPlace._geo?.lng ?? selectedPlace.lng;
    if (typeof lat === 'number' && typeof lng === 'number' && !isNaN(lat) && !isNaN(lng)) {
      map.current.flyTo({ center: [lng, lat], zoom: Math.max(map.current.getZoom(), 14), duration: 700 });

      // Open popup for selected marker
      const item = markersRef.current.get(selectedPlace.id);
      if (item) {
        const popup = item.marker.getPopup();
        if (popup && !popup.isOpen()) {
          item.marker.togglePopup();
        }
      }
    }
  }, [selectedPlace]);

  return (
    <div className="relative w-full h-full min-h-[380px] rounded-xl overflow-hidden border border-slate-800 shadow-inner bg-slate-900">
      <div ref={mapContainer} className="w-full h-full absolute inset-0" />

      {/* Map Overlay Controls / Hints */}
      <div className="absolute top-3 left-3 z-10 flex flex-col gap-2">
        <div className="bg-slate-900/85 backdrop-blur-md px-3 py-1.5 rounded-lg border border-slate-700/60 text-xs shadow-md flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span className="text-slate-300 font-medium">MapLibre GL Live</span>
          {hits.length > 0 && (
            <span className="bg-slate-800 text-slate-300 px-1.5 py-0.5 rounded text-[10px]">
              {hits.length} pin{hits.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>

        {pinLocation && (
          <div className="bg-slate-900/90 backdrop-blur-md px-2.5 py-1 rounded border border-rose-500/40 text-[11px] text-rose-300 flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-rose-500"></span>
            Pin: {pinLocation.lat}, {pinLocation.lng}
          </div>
        )}
      </div>

      <div className="absolute bottom-3 right-3 z-10">
        <div className="bg-slate-900/80 backdrop-blur-md px-2 py-1 rounded text-[10px] text-slate-400 border border-slate-800">
          Click map to set (lat, lng) pin
        </div>
      </div>
    </div>
  );
};
