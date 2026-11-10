export type ProxyMode = 'record' | 'replay' | 'auto' | 'passthrough';

export interface LiveInteraction {
  id: string;
  timestamp: string;
  mode: ProxyMode;
  status: number;
  method: string;
  url: string;
  host: string;
  path: string;
  req_headers: Record<string, string>;
  req_body: string;
  res_headers: Record<string, string>;
  res_body: string;
  latency_ms: number;
  matched: boolean;
  cassette: string;
  error?: string;
}

export interface CassetteSummary {
  name: string;
  interactions: number;
  active: boolean;
}

export interface Interaction {
  id: string;
  recorded_at: string;
  request: {
    method: string;
    url: string;
    headers?: Record<string, string>;
    body?: string;
    body_hash?: string;
  };
  response: {
    status: number;
    status_text?: string;
    headers?: Record<string, string>;
    body?: string;
    body_hash?: string;
    latency_ms?: number;
  };
}

export interface Cassette {
  version: number;
  recorded_at: string;
  recorded_by: string;
  interactions: Interaction[];
}

export interface ChaosConfig {
  latency: number; // nanoseconds or duration string
  jitter: number;
  error_rate: number;
  error_status: number;
  rate_limit_freq: number;
}

export interface StatusResponse {
  mode: ProxyMode;
  active_cassette: string;
  cassettes_count: number;
  interactions_count: number;
  ca_fingerprint: string;
  chaos: ChaosConfig;
}
