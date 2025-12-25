import { Effect } from "effect";
import { TelegramConfigPartial } from "./config";

// Error types for API classification
export type ApiErrorType =
  | "auth_failed"
  | "not_found"
  | "server_error"
  | "network_error"
  | "unknown";

export interface ApiError {
  type: ApiErrorType;
  status?: number;
  message: string;
  code?: string;
}

// Type guard for ApiError
export const isApiError = (error: unknown): error is ApiError => {
  return (
    typeof error === "object" &&
    error !== null &&
    "type" in error &&
    "message" in error
  );
};

export const createApi = (config: TelegramConfigPartial) => {
  const apiFetch = <T>(
    path: string,
    init: RequestInit = {},
    requireAdmin = false,
  ) =>
    Effect.tryPromise(async () => {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...(init.headers as Record<string, string> | undefined),
      };

      if (requireAdmin) {
        headers["X-API-Key"] = config.adminApiKey;
      }

      let response: Response;
      try {
        response = await fetch(`${config.apiBaseUrl}${path}`, {
          ...init,
          headers,
        });
      } catch (networkError) {
        console.error(
          `[API] Network error for ${path}:`,
          networkError instanceof Error ? networkError.message : networkError,
        );
        const apiError: ApiError = {
          type: "network_error",
          message: "Network error: Unable to connect to backend API",
        };
        throw apiError;
      }

      const payload = await response
        .json()
        .catch(() => ({ message: "Failed to parse response" }));

      if (!response.ok) {
        // Classify error type based on status code
        const errorType: ApiErrorType =
          response.status === 401
            ? "auth_failed"
            : response.status === 404
              ? "not_found"
              : response.status >= 500
                ? "server_error"
                : "unknown";

        const message =
          payload?.error ||
          payload?.message ||
          `API request failed (${response.status})`;

        // Log authentication failures with context for debugging
        if (errorType === "auth_failed") {
          console.error(
            `[API] Authentication failed for ${path} - check ADMIN_API_KEY configuration`,
          );
        } else {
          console.error(`[API] ${errorType}: ${response.status} for ${path}`);
        }

        const apiError: ApiError = {
          type: errorType,
          status: response.status,
          message,
          code: payload?.code,
        };
        throw apiError;
      }

      return payload as T;
    });

  const getUserByChatId = (chatId: string) =>
    apiFetch<{
      user: { id: string; subscription_tier: string; created_at: string };
    }>(
      `/api/v1/telegram/internal/users/${encodeURIComponent(chatId)}`,
      {},
      true,
    );

  const getNotificationPreference = (userId: string) =>
    apiFetch<{
      enabled: boolean;
      profit_threshold: number;
      alert_frequency: string;
    }>(
      `/api/v1/telegram/internal/notifications/${encodeURIComponent(userId)}`,
      {},
      true,
    );

  const setNotificationPreference = (userId: string, enabled: boolean) =>
    apiFetch(
      `/api/v1/telegram/internal/notifications/${encodeURIComponent(userId)}`,
      {
        method: "POST",
        body: JSON.stringify({ enabled }),
      },
      true,
    );

  const registerTelegramUser = (chatId: string, userId: number) =>
    apiFetch(
      "/api/v1/users/register",
      {
        method: "POST",
        body: JSON.stringify({
          email: `telegram_${userId}@celebrum.ai`,
          password: `${globalThis.crypto.randomUUID()}${globalThis.crypto.randomUUID()}`,
          telegram_chat_id: chatId,
        }),
      },
      false,
    );

  const getOpportunities = () =>
    apiFetch<{ opportunities: any[] }>(
      "/api/v1/arbitrage/opportunities?limit=5&min_profit=0.5",
    );

  return {
    getUserByChatId,
    getNotificationPreference,
    setNotificationPreference,
    registerTelegramUser,
    getOpportunities,
  };
};

export type Api = ReturnType<typeof createApi>;
