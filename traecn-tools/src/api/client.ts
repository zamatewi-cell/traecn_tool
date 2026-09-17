// API Client for TraeCN Tools
import type { ProxyConfig, Account, AppData } from '../types';

const BASE_URL = 'http://127.0.0.1:9090';

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface ModelInfo {
  id: string;
  object: string;
  created: number;
  owned_by: string;
}

export interface ModelsResponse {
  object: string;
  data: ModelInfo[];
}

export interface QueueStatus {
  enabled: boolean;
  current_tokens: number;
  queue_length: number;
  estimated_wait_seconds: number;
}

export interface HealthStatus {
  status: string;
  version: string;
  timestamp: string;
}

/**
 * API Client for communicating with trae-proxy backend
 */
export class ApiClient {
  private baseUrl: string;
  private apiKey: string | null;

  constructor(baseUrl: string = BASE_URL, apiKey: string | null = null) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
  }

  setApiKey(key: string) {
    this.apiKey = key;
  }

  setBaseUrl(url: string) {
    this.baseUrl = url;
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
    try {
      const headers: HeadersInit = {
        'Content-Type': 'application/json',
        ...(this.apiKey ? { 'Authorization': `Bearer ${this.apiKey}` } : {}),
      };

      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        ...options,
        headers: {
          ...headers,
          ...(options?.headers || {}),
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        return {
          success: false,
          error: `HTTP ${response.status}: ${errorText}`,
        };
      }

      const data = await response.json();
      return { success: true, data };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * Health check
   */
  async health(): Promise<ApiResponse<HealthStatus>> {
    return this.request<HealthStatus>('/health');
  }

  /**
   * Get available models
   */
  async getModels(): Promise<ApiResponse<ModelsResponse>> {
    return this.request<ModelsResponse>('/v1/models');
  }

  /**
   * Get queue status
   */
  async getQueueStatus(): Promise<ApiResponse<QueueStatus>> {
    return this.request<QueueStatus>('/v1/queue/status');
  }

  /**
   * Send chat completion request
   */
  async chatCompletion(
    model: string,
    messages: Array<{ role: string; content: string }>,
    stream: boolean = false
  ): Promise<ApiResponse<any>> {
    return this.request<any>('/v1/chat/completions', {
      method: 'POST',
      body: JSON.stringify({
        model,
        messages,
        stream,
      }),
    });
  }

  /**
   * Test API connectivity
   */
  async testConnection(): Promise<ApiResponse<boolean>> {
    const result = await this.health();
    if (result.success) {
      return { success: true, data: true };
    }
    return result;
  }
}

// Singleton instance for app-wide use
export const apiClient = new ApiClient();
