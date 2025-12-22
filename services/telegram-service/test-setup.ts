import { mock } from "bun:test";

// Ensure environment variables are set for testing
process.env.TELEGRAM_BOT_TOKEN = "test-bot-token";
process.env.ADMIN_API_KEY =
  "test-admin-key-that-is-at-least-32-characters-long-for-security";
process.env.TELEGRAM_WEBHOOK_SECRET = "test-webhook-secret";
process.env.TELEGRAM_PORT = "3003";
process.env.TELEGRAM_USE_POLLING = "false";
process.env.TELEGRAM_WEBHOOK_URL = "https://example.com/webhook";

// Mock Grammy
mock.module("grammy", () => {
  class MockApi {
    async sendMessage(chatId: string | number, text: string, options?: any) {
      // Mock implementation
      return { message_id: 123, chat: { id: chatId }, text };
    }

    async deleteWebhook(options?: any) {
      return true;
    }

    async setWebhook(url: string, options?: any) {
      return true;
    }
  }

  class MockBot {
    api: MockApi;

    constructor(token: string) {
      this.api = new MockApi();
    }

    async handleUpdate(update: any) {
      return true;
    }

    async start() {
      // Do nothing
    }

    command(cmd: string, handler: any) {
      // Do nothing
    }

    on(event: string, handler: any) {
      // Do nothing
    }

    catch(handler: any) {
      // Do nothing
    }
  }

  return {
    Bot: MockBot,
    GrammyError: class GrammyError extends Error {},
    HttpError: class HttpError extends Error {},
  };
});

export {};
