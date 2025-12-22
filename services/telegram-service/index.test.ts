import "./test-setup";
import { test, expect } from "bun:test";

// Cache the service instance
let serviceInstance: {
  port: number | string;
  fetch: (req: Request) => Promise<Response>;
} | null = null;
let initializationPromise: Promise<any> | null = null;

async function getService() {
  if (!serviceInstance) {
    if (!initializationPromise) {
      initializationPromise = (async () => {
        // Force fresh import
        const mod = await import("./index.ts?" + Date.now());
        serviceInstance = mod.default as any; // The Hono app is not exported as default usually, but we are running the server in index.ts
        // Wait, index.ts starts the server directly and doesn't export app by default in the way we might expect for testing if we want to fetch directly.
        // However, Bun.serve returns a server instance. index.ts doesn't export it.
        // But since we are running in Bun test environment, importing index.ts starts the server on the configured port.
        // We can just make requests to http://localhost:PORT

        // Wait for service to be ready
        const port = process.env.PORT || 3003;
        const timeout = 5000;
        const interval = 100;
        const startTime = Date.now();

        while (Date.now() - startTime < timeout) {
          try {
            const healthRes = await fetch(`http://localhost:${port}/health`);
            if (healthRes.status === 200) {
              return { port, fetch: fetch };
            }
          } catch (error) {
            // Service not ready yet
          }
          await new Promise((resolve) => setTimeout(resolve, interval));
        }
        throw new Error("Service failed to start");
      })();
    }
    await initializationPromise;
  }
  return {
    baseUrl: `http://localhost:${process.env.TELEGRAM_PORT || 3003}`,
  };
}

test("health endpoint returns healthy", async () => {
  const { baseUrl } = await getService();
  const res = await fetch(`${baseUrl}/health`);
  expect(res.status).toBe(200);
  const body = await res.json();
  expect(body.status).toBe("healthy");
  expect(body.service).toBe("telegram-service");
});

test("send-message returns 401 without API key", async () => {
  const { baseUrl } = await getService();
  const res = await fetch(`${baseUrl}/send-message`, {
    method: "POST",
    body: JSON.stringify({ chatId: "123", text: "hello" }),
  });
  expect(res.status).toBe(401);
});

test("send-message returns 401 with invalid API key", async () => {
  const { baseUrl } = await getService();
  const res = await fetch(`${baseUrl}/send-message`, {
    method: "POST",
    headers: { "X-API-Key": "invalid-key" },
    body: JSON.stringify({ chatId: "123", text: "hello" }),
  });
  expect(res.status).toBe(401);
});

test("send-message returns 400 with missing parameters", async () => {
  const { baseUrl } = await getService();
  const res = await fetch(`${baseUrl}/send-message`, {
    method: "POST",
    headers: { "X-API-Key": process.env.ADMIN_API_KEY! },
    body: JSON.stringify({ text: "hello" }), // missing chatId
  });
  expect(res.status).toBe(400);
});

test("send-message sends successfully", async () => {
  const { baseUrl } = await getService();
  const res = await fetch(`${baseUrl}/send-message`, {
    method: "POST",
    headers: { "X-API-Key": process.env.ADMIN_API_KEY! },
    body: JSON.stringify({ chatId: "123", text: "test message" }),
  });
  expect(res.status).toBe(200);
  const body = await res.json();
  expect(body.ok).toBe(true);
});

test("webhook endpoint rejects invalid secret token", async () => {
  const { baseUrl } = await getService();
  // Using the path defined in test-setup logic or default logic
  // index.ts: resolvedWebhookPath = ... new URL(webhookUrl).pathname -> /webhook
  const webhookPath = "/webhook";

  const res = await fetch(`${baseUrl}${webhookPath}`, {
    method: "POST",
    headers: { "X-Telegram-Bot-Api-Secret-Token": "wrong-secret" },
    body: JSON.stringify({ update_id: 1, message: { text: "hi" } }),
  });
  expect(res.status).toBe(401);
});

test("webhook endpoint accepts valid secret token", async () => {
  const { baseUrl } = await getService();
  const webhookPath = "/webhook";

  const res = await fetch(`${baseUrl}${webhookPath}`, {
    method: "POST",
    headers: {
      "X-Telegram-Bot-Api-Secret-Token": process.env.TELEGRAM_WEBHOOK_SECRET!,
    },
    body: JSON.stringify({ update_id: 1, message: { text: "hi" } }),
  });
  expect(res.status).toBe(200);
  const body = await res.json();
  expect(body.ok).toBe(true);
});
