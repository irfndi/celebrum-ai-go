import * as grpc from "@grpc/grpc-js";
import {
  TelegramServiceService,
  TelegramServiceServer,
  SendMessageRequest,
  SendMessageResponse,
  HealthCheckRequest,
  HealthCheckResponse,
} from "./proto/telegram_service";
import { Bot } from "grammy";
import {
  classifyError,
  TelegramErrorCode,
  TelegramErrorInfo,
} from "./telegram-errors";
import { withRetry, RetryConfig } from "./retry";

const SEND_TIMEOUT = 30000;

const sendWithTimeout = <T>(promise: Promise<T>): Promise<T> => {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) =>
      setTimeout(() => reject(new Error("Telegram API timeout")), SEND_TIMEOUT),
    ),
  ]);
};

// Retry configuration for gRPC message sending
const GRPC_RETRY_CONFIG: Partial<RetryConfig> = {
  maxRetries: 2,
  initialDelayMs: 500,
  maxDelayMs: 5000,
};

export class TelegramGrpcServer {
  private bot: Bot;

  constructor(bot: Bot) {
    this.bot = bot;
  }

  sendMessage = (
    call: grpc.ServerUnaryCall<SendMessageRequest, SendMessageResponse>,
    callback: grpc.sendUnaryData<SendMessageResponse>,
  ): void => {
    const { chatId, text, parseMode } = call.request;

    if (!chatId || !text) {
      callback(null, {
        ok: false,
        messageId: "",
        error: "Chat ID and Text are required",
        errorCode: TelegramErrorCode.INVALID_REQUEST,
        retryAfter: 0,
      });
      return;
    }

    // Use retry logic for sending messages
    withRetry(
      () =>
        sendWithTimeout(
          this.bot.api.sendMessage(chatId, text, {
            parse_mode: parseMode as any,
          }),
        ),
      classifyError,
      GRPC_RETRY_CONFIG,
    )
      .then((result) => {
        if (result.success && result.data) {
          callback(null, {
            ok: true,
            messageId: result.data.message_id.toString(),
            error: "",
            errorCode: "",
            retryAfter: 0,
          });
        } else {
          const errorInfo = result.error as TelegramErrorInfo;
          console.error(
            `[gRPC] Failed to send message after ${result.attempts} attempts:`,
            errorInfo?.code,
            errorInfo?.message,
          );
          callback(null, {
            ok: false,
            messageId: "",
            error: errorInfo?.message || "Unknown error",
            errorCode: errorInfo?.code || TelegramErrorCode.UNKNOWN,
            retryAfter: errorInfo?.retryAfter || 0,
          });
        }
      })
      .catch((error) => {
        // This shouldn't happen as withRetry doesn't throw, but handle just in case
        console.error("[gRPC] Unexpected error in sendMessage:", error);
        const errorInfo = classifyError(error);
        callback(null, {
          ok: false,
          messageId: "",
          error: errorInfo.message,
          errorCode: errorInfo.code,
          retryAfter: errorInfo.retryAfter || 0,
        });
      });
  };

  healthCheck = (
    call: grpc.ServerUnaryCall<HealthCheckRequest, HealthCheckResponse>,
    callback: grpc.sendUnaryData<HealthCheckResponse>,
  ): void => {
    callback(null, {
      status: "serving",
      version: "1.0.0",
      service: "telegram-service",
    });
  };
}

export function startGrpcServer(bot: Bot, port: number) {
  const server = new grpc.Server();
  const service = new TelegramGrpcServer(bot);

  server.addService(
    TelegramServiceService,
    service as unknown as TelegramServiceServer,
  );

  const bindAddr = `0.0.0.0:${port}`;
  server.bindAsync(
    bindAddr,
    grpc.ServerCredentials.createInsecure(),
    (err, port) => {
      if (err) {
        console.error(`Failed to bind gRPC server: ${err}`);
        throw err; // Propagate error to caller
      }
      console.log(`🚀 Telegram gRPC Service listening on ${bindAddr}`);
    },
  );

  return server;
}
