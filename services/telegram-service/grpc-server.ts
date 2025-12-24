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

const SEND_TIMEOUT = 30000;

const sendWithTimeout = <T>(promise: Promise<T>): Promise<T> => {
  return Promise.race([
    promise,
    new Promise<T>((_, reject) =>
      setTimeout(() => reject(new Error("Telegram API timeout")), SEND_TIMEOUT),
    ),
  ]);
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
      callback(
        {
          code: grpc.status.INVALID_ARGUMENT,
          details: "Chat ID and Text are required",
        },
        null,
      );
      return;
    }

    sendWithTimeout(
      this.bot.api.sendMessage(chatId, text, {
        parse_mode: parseMode as any,
      }),
    )
      .then((sent) => {
        callback(null, {
          ok: true,
          messageId: sent.message_id.toString(),
          error: "",
        });
      })
      .catch((error) => {
        console.error("Failed to send message via gRPC:", error);
        callback(null, {
          ok: false,
          messageId: "",
          error: error.message || "Unknown error",
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
