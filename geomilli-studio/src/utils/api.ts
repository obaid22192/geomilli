export const API_BASE = import.meta.env.VITE_API_BASE_URL || '/v1';

export async function fetchWithTiming<T>(url: string, options?: RequestInit): Promise<{ data: T; durationMs: number }> {
  const start = performance.now();
  const res = await fetch(url, options);
  const durationMs = Math.round(performance.now() - start);
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`HTTP ${res.status}: ${errorText || res.statusText}`);
  }
  const data = await res.json();
  return { data, durationMs };
}
