export const API_BASE = import.meta.env.VITE_API_BASE_URL || '/v1';

export interface ApiErrorInfo {
  code?: string;
  message: string;
  status: number;
}

export class ApiError extends Error {
  code?: string;
  status: number;
  rawResponse?: any;

  constructor(message: string, status: number, code?: string, rawResponse?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.rawResponse = rawResponse;
  }
}

/**
 * Extracts error message, code, and status from a response body.
 * Gracefully handles RFC-001 error envelopes ({ error: { code, message, status } }),
 * legacy error envelopes ({ error: true, status, message }), and plain text error responses.
 */
export async function parseApiError(res: Response): Promise<ApiErrorInfo & { raw?: any }> {
  const status = res.status;
  try {
    const text = await res.text();
    if (!text) {
      return { message: res.statusText || `HTTP ${status}`, status };
    }
    try {
      const data = JSON.parse(text);
      if (data && typeof data === 'object') {
        // RFC-001 envelope: { error: { code, message, status } }
        if (data.error && typeof data.error === 'object') {
          return {
            code: data.error.code,
            message: data.error.message || `HTTP ${status}`,
            status: data.error.status || status,
            raw: data,
          };
        }
        // Legacy envelope: { error: true, status, message }
        if (data.message) {
          return {
            code: data.code,
            message: data.message,
            status: data.status || status,
            raw: data,
          };
        }
      }
      return { message: text, status, raw: data };
    } catch {
      return { message: text, status };
    }
  } catch {
    return { message: res.statusText || `HTTP ${status}`, status };
  }
}

export async function fetchWithTiming<T>(url: string, options?: RequestInit): Promise<{ data: T; durationMs: number }> {
  const start = performance.now();
  const res = await fetch(url, options);
  const durationMs = Math.round(performance.now() - start);
  if (!res.ok) {
    const errInfo = await parseApiError(res);
    throw new ApiError(errInfo.message, errInfo.status, errInfo.code, errInfo.raw);
  }
  const data = await res.json();
  return { data, durationMs };
}
