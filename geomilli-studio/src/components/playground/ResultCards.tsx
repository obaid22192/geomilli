import React, { useEffect, useRef } from 'react';
import { Place } from '../../types';
import {
  ExternalLink,
  MapPin,
  Star,
  Utensils,
  Plane,
  Eye,
  Copy,
  Check,
} from 'lucide-react';

interface ResultCardsProps {
  hits: Place[];
  selectedPlace: Place | null;
  onSelectPlace: (place: Place) => void;
  loading: boolean;
  totalHits?: number;
}

export const ResultCards: React.FC<ResultCardsProps> = ({
  hits,
  selectedPlace,
  onSelectPlace,
  loading,
  totalHits,
}) => {
  const [copiedId, setCopiedId] = React.useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const copyToClipboard = (text: string, id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  // Scroll active card into view when selected from Map
  useEffect(() => {
    if (selectedPlace?.id) {
      const cardEl = document.getElementById(`place-card-${selectedPlace.id}`);
      if (cardEl) {
        cardEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    }
  }, [selectedPlace]);

  if (loading) {
    return (
      <div className="flex flex-col gap-2 p-3">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-24 bg-slate-900/60 rounded-xl animate-pulse border border-slate-800"
          />
        ))}
      </div>
    );
  }

  if (hits.length === 0) {
    return (
      <div className="h-64 flex flex-col items-center justify-center text-center p-6 text-slate-500 border border-dashed border-slate-800 rounded-xl">
        <MapPin className="w-8 h-8 text-slate-600 mb-2 stroke-[1.5]" />
        <p className="text-sm font-medium text-slate-400">No places found</p>
        <p className="text-xs text-slate-600 mt-1">
          Try adjusting your query, removing class filters, or zooming out on the map.
        </p>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="flex flex-col gap-2 overflow-y-auto max-h-[640px] pr-1">
      <div className="text-xs text-slate-400 px-1 py-0.5 flex justify-between items-center">
        <span>
          Showing <strong className="text-slate-200">{hits.length}</strong> results
          {totalHits !== undefined && totalHits > hits.length && (
            <span className="text-slate-500"> of ~{totalHits}</span>
          )}
        </span>
      </div>

      {hits.map((place) => {
        const isSelected = selectedPlace?.id === place.id;
        const lat = place._geo?.lat ?? place.lat;
        const lng = place._geo?.lng ?? place.lng;
        const displayName = place.name_display || place.name || '<Unnamed Place>';
        const locality = place.admin_hierarchy?.[0] || place.city || place.state || place.country_name || '';
        const distance = place.distance_meters ?? place._geoDistance;
        const iata = place.iata_code || place.iata;
        const icao = place.icao_code || place.icao;
        const wikidata = place.wikidata_id || place.wikidata;

        return (
          <div
            key={place.id}
            id={`place-card-${place.id}`}
            onClick={() => onSelectPlace(place)}
            className={`p-3 rounded-xl border transition cursor-pointer flex flex-col gap-2 ${
              isSelected
                ? 'bg-slate-850 border-brand-500/80 shadow-md ring-1 ring-brand-500/40'
                : 'bg-slate-900/80 hover:bg-slate-850 border-slate-800/80 hover:border-slate-700'
            }`}
          >
            {/* Top row: Name & Class pill */}
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1 min-w-0">
                <h4 className="text-sm font-semibold text-slate-100 truncate">
                  {displayName}
                </h4>
                <div className="flex items-center gap-1.5 text-xs text-slate-400 mt-0.5">
                  {locality && <span className="text-slate-300 font-medium truncate max-w-[180px]">{locality}</span>}
                  {place.country_code && (
                    <span className="bg-slate-800 text-slate-400 px-1.5 py-0.2 rounded text-[10px] font-mono">
                      {place.country_code}
                    </span>
                  )}
                  {distance !== undefined && (
                    <span className="text-brand-400 text-[11px] font-medium">
                      {distance < 1000
                        ? `${Math.round(distance)}m`
                        : `${(distance / 1000).toFixed(1)}km`}
                    </span>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-1 flex-shrink-0">
                <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-slate-800 text-brand-300 border border-slate-700/60">
                  {place.class}
                </span>
                {place.subclass && place.subclass !== place.class.toLowerCase() && (
                  <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-950 text-slate-400 border border-slate-800">
                    {place.subclass}
                  </span>
                )}
              </div>
            </div>

            {/* Badges: Wikidata, Wikipedia Pageviews, Stars, Airport code */}
            <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
              {iata && (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-300 border border-amber-500/30 font-mono font-bold">
                  <Plane className="w-3 h-3" />
                  IATA: {iata} {icao ? `(${icao})` : ''}
                </span>
              )}

              {place.star_rating && place.star_rating > 0 && (
                <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded bg-amber-400/10 text-amber-300 border border-amber-400/20">
                  <Star className="w-3 h-3 fill-amber-400 text-amber-400" />
                  {place.star_rating}★
                </span>
              )}

              {place.cuisine && (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-300 border border-emerald-500/20">
                  <Utensils className="w-3 h-3" />
                  {place.cuisine}
                </span>
              )}

              {wikidata && (
                <a
                  href={`https://www.wikidata.org/wiki/${wikidata}`}
                  target="_blank"
                  rel="noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-300 border border-indigo-500/30 hover:bg-indigo-500/20 font-mono text-[10px]"
                >
                  <span>{wikidata}</span>
                  <ExternalLink className="w-2.5 h-2.5" />
                </a>
              )}

              {place.wikipedia_pageviews !== undefined && place.wikipedia_pageviews > 0 && (
                <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-300 border border-purple-500/20 text-[10px]">
                  <Eye className="w-3 h-3" />
                  {place.wikipedia_pageviews.toLocaleString()} views
                </span>
              )}
            </div>

            {/* Bottom row: Coordinates & Action */}
            <div className="pt-1 border-t border-slate-800/60 flex items-center justify-between text-[11px] text-slate-500">
              <span className="font-mono">
                {typeof lat === 'number' ? lat.toFixed(5) : '--'}, {typeof lng === 'number' ? lng.toFixed(5) : '--'}
              </span>
              <button
                onClick={(e) => copyToClipboard(`${lat},${lng}`, place.id, e)}
                className="hover:text-slate-300 flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded hover:bg-slate-800"
                title="Copy coordinates"
              >
                {copiedId === place.id ? (
                  <>
                    <Check className="w-3 h-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" />
                    <span>Copy coords</span>
                  </>
                )}
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
};
