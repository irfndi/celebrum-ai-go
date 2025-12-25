import { Effect } from "effect";
import { TelegramConfigPartial } from "./config";

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

      const response = await fetch(`${config.apiBaseUrl}${path}`, {
        ...init,
        headers,
      });

      const payload = await response
        .json()
        .catch(() => ({ message: "Failed to parse response" }));

      if (!response.ok) {
        const message =
          payload?.error ||
          payload?.message ||
          `API request failed (${response.status})`;
        throw new Error(message);
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
