import { Effect } from "effect";

export type TelegramConfig = {
    botToken: string;
    webhookUrl: string | null;
    webhookPath: string;
    webhookSecret: string | null;
    usePolling: boolean;
    port: number;
    apiBaseUrl: string;
    adminApiKey: string;
};

export type TelegramConfigPartial = TelegramConfig & {
    botTokenMissing: boolean;
    configError: string | null;
};

const resolvePort = (raw: string | undefined, fallback: number) => {
    if (!raw) {
        return fallback;
    }
    const numericPort = Number(raw);
    if (!Number.isNaN(numericPort) && numericPort > 0 && numericPort < 65536) {
        return numericPort;
    }
    console.warn(
        `Invalid port value provided (${raw}). Falling back to default (${fallback}).`,
    );
    return fallback;
};

export const loadConfig = (): TelegramConfigPartial => {
    // Support both TELEGRAM_BOT_TOKEN and TELEGRAM_TOKEN for compatibility with different platforms
    const botToken =
        process.env.TELEGRAM_BOT_TOKEN || process.env.TELEGRAM_TOKEN || "";
    const botTokenMissing = !botToken;

    const adminApiKey = process.env.ADMIN_API_KEY || "";
    const isProduction =
        process.env.NODE_ENV === "production" ||
        process.env.SENTRY_ENVIRONMENT === "production";

    let configError: string | null = null;

    // Validate ADMIN_API_KEY only in production
    if (isProduction) {
        if (!adminApiKey) {
            configError =
                "ADMIN_API_KEY environment variable must be set in production";
        } else if (
            adminApiKey === "admin-secret-key-change-me" ||
            adminApiKey === "admin-dev-key-change-in-production"
        ) {
            configError =
                "ADMIN_API_KEY cannot use default/example values. Please set a secure API key.";
        } else if (adminApiKey.length < 32) {
            configError =
                "ADMIN_API_KEY must be at least 32 characters long for security";
        }
    } else if (!adminApiKey) {
        console.warn(
            "⚠️ WARNING: ADMIN_API_KEY is not set. Admin endpoints will be disabled.",
        );
    }

    if (botTokenMissing) {
        console.error(
            "❌ CRITICAL: TELEGRAM_BOT_TOKEN environment variable is not set!",
        );
        console.error(
            "   The service will start in degraded mode (health check only).",
        );
        console.error(
            "   Bot functionality will be disabled until the token is configured.",
        );
    }

    const apiBaseUrl = (
        process.env.TELEGRAM_API_BASE_URL || "http://localhost:8080"
    ).replace(/\/$/, "");

    const webhookUrlRaw = (process.env.TELEGRAM_WEBHOOK_URL || "").trim();
    const webhookUrl = webhookUrlRaw.length > 0 ? webhookUrlRaw : null;
    const webhookPath = (process.env.TELEGRAM_WEBHOOK_PATH || "").trim();
    const resolvedWebhookPath = webhookPath
        ? webhookPath
        : webhookUrl
            ? new URL(webhookUrl).pathname
            : "/telegram/webhook";

    const usePollingEnv = (process.env.TELEGRAM_USE_POLLING || "").toLowerCase();
    const usePolling =
        usePollingEnv === "true" || usePollingEnv === "1" || webhookUrl === null;

    return {
        botToken,
        botTokenMissing,
        configError,
        webhookUrl,
        webhookPath: resolvedWebhookPath.startsWith("/")
            ? resolvedWebhookPath
            : `/${resolvedWebhookPath}`,
        webhookSecret: process.env.TELEGRAM_WEBHOOK_SECRET || null,
        usePolling,
        port: resolvePort(process.env.TELEGRAM_PORT, 3002),
        apiBaseUrl,
        adminApiKey,
    };
};
