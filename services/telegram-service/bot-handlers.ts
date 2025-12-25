import { Context } from "grammy";
import { Effect } from "effect";
import { Api } from "./api";

export const formatOpportunitiesMessage = (opps: any[]) => {
  if (!opps || opps.length === 0) {
    return "📊 No arbitrage opportunities found right now.";
  }

  const top = opps.slice(0, 5);
  const lines = ["⚡ Top Arbitrage Opportunities", ""];

  top.forEach((opp, index) => {
    lines.push(`${index + 1}. ${opp.symbol}`);
    lines.push(`   Buy: ${opp.buy_exchange} @ ${opp.buy_price}`);
    lines.push(`   Sell: ${opp.sell_exchange} @ ${opp.sell_price}`);
    lines.push(`   Profit: ${Number(opp.profit_percent).toFixed(2)}%`);
    lines.push("");
  });

  return lines.join("\n");
};

export const handleStart = (api: Api) => async (ctx: Context) => {
  const chatId = ctx.chat?.id;
  const userId = ctx.from?.id;

  if (!chatId || !userId) {
    await ctx.reply("Unable to start: missing chat information.");
    return;
  }

  const chatIdStr = String(chatId);

  const userResult = await Effect.runPromise(
    Effect.catchAll(api.getUserByChatId(chatIdStr), () => Effect.succeed(null)),
  );

  if (!userResult) {
    await Effect.runPromise(
      Effect.catchAll(api.registerTelegramUser(chatIdStr, userId), () =>
        Effect.succeed(null),
      ),
    );
  }

  const welcomeMsg =
    "🚀 Welcome to Celebrum AI!\n\n" +
    "✅ You're now registered and ready to receive arbitrage alerts!\n\n" +
    "Use /opportunities to see current opportunities.\n" +
    "Use /help to see available commands.";

  await ctx.reply(welcomeMsg);
};

export const handleHelp = () => async (ctx: Context) => {
  const msg =
    "🤖 Celebrum AI Bot Commands:\n\n" +
    "/start - Register and get started\n" +
    "/opportunities - View current arbitrage opportunities\n" +
    "/settings - Configure your alert preferences\n" +
    "/upgrade - Upgrade to premium subscription\n" +
    "/status - Check your account status\n" +
    "/stop - Pause all notifications\n" +
    "/resume - Resume notifications\n" +
    "/help - Show this help message\n\n" +
    "💡 Tip: You'll receive automatic alerts when profitable opportunities are detected!";

  await ctx.reply(msg);
};

export const handleOpportunities = (api: Api) => async (ctx: Context) => {
  const response = await Effect.runPromise(
    Effect.catchAll(api.getOpportunities(), (error) =>
      Effect.fail(error as Error),
    ),
  ).catch(async (error) => {
    await ctx.reply(
      `❌ Failed to fetch opportunities. Please try again later. (${(error as Error).message})`,
    );
    return null;
  });

  if (!response) {
    return;
  }

  await ctx.reply(formatOpportunitiesMessage(response.opportunities));
};

export const handleStatus = (api: Api) => async (ctx: Context) => {
  const chatId = ctx.chat?.id;
  const userId = ctx.from?.id;
  if (!chatId) {
    await ctx.reply("Unable to lookup status: missing chat information.");
    return;
  }

  const userResult = await Effect.runPromise(
    Effect.catchAll(api.getUserByChatId(String(chatId)), () =>
      Effect.succeed(null),
    ),
  );

  if (!userResult) {
    await ctx.reply("User not found. Please use /start to register.");
    return;
  }

  const preference = userId
    ? await Effect.runPromise(
        Effect.catchAll(api.getNotificationPreference(String(userId)), () =>
          Effect.succeed({
            enabled: true,
            profit_threshold: 0.5,
            alert_frequency: "Every 5 minutes",
          }),
        ),
      )
    : {
        enabled: true,
        profit_threshold: 0.5,
        alert_frequency: "Every 5 minutes",
      };

  const createdAt = new Date(userResult.user.created_at).toLocaleDateString();
  const tier = userResult.user.subscription_tier;
  const notificationStatus = preference.enabled ? "Active" : "Paused";

  const msg =
    "📊 Account Status:\n\n" +
    `💰 Subscription: ${tier}\n` +
    `📅 Member since: ${createdAt}\n` +
    `🔔 Notifications: ${notificationStatus}`;

  await ctx.reply(msg);
};

export const handleSettings = (api: Api) => async (ctx: Context) => {
  const chatId = ctx.chat?.id;
  const userId = ctx.from?.id;
  if (!userId || !chatId) {
    await ctx.reply("Unable to fetch settings right now.");
    return;
  }

  // Fetch user for subscription tier
  const userResult = await Effect.runPromise(
    Effect.catchAll(api.getUserByChatId(String(chatId)), () =>
      Effect.succeed(null),
    ),
  );

  const preference = await Effect.runPromise(
    Effect.catchAll(api.getNotificationPreference(String(userId)), () =>
      Effect.succeed({
        enabled: true,
        profit_threshold: 0.5,
        alert_frequency: "Immediate (Periodic Scan 5m)",
      }),
    ),
  );

  const statusIcon = preference.enabled ? "✅" : "❌";
  const statusText = preference.enabled ? "ON" : "OFF";
  const threshold = preference.profit_threshold ?? 0.5;
  const frequency =
    preference.alert_frequency ?? "Immediate (Periodic Scan 5m)";
  const tier = userResult?.user?.subscription_tier ?? "Free Tier";

  const msg =
    "⚙️ Alert Settings:\n\n" +
    `🔔 Notifications: ${statusIcon} ${statusText}\n` +
    `📊 Min Profit Threshold: ${threshold}%\n` +
    `⏰ Alert Frequency: ${frequency}\n` +
    `💰 Subscription: ${tier}\n\n` +
    "To change settings:\n" +
    "/stop - Pause notifications\n" +
    "/resume - Resume notifications\n" +
    "/upgrade - Upgrade to premium for more options";

  await ctx.reply(msg);
};

export const handleStop = (api: Api) => async (ctx: Context) => {
  const userId = ctx.from?.id;
  if (!userId) {
    await ctx.reply("Unable to update notifications.");
    return;
  }

  await Effect.runPromise(
    Effect.catchAll(api.setNotificationPreference(String(userId), false), () =>
      Effect.succeed(null),
    ),
  );

  const msg =
    "⏸️ Notifications Paused\n\n" +
    "You will no longer receive arbitrage alerts.\n\n" +
    "Use /resume to start receiving alerts again.";

  await ctx.reply(msg);
};

export const handleResume = (api: Api) => async (ctx: Context) => {
  const userId = ctx.from?.id;
  if (!userId) {
    await ctx.reply("Unable to update notifications.");
    return;
  }

  await Effect.runPromise(
    Effect.catchAll(api.setNotificationPreference(String(userId), true), () =>
      Effect.succeed(null),
    ),
  );

  const msg =
    "▶️ Notifications Resumed\n\n" +
    "You will now receive arbitrage alerts again.\n\n" +
    "Use /opportunities to see current opportunities.";

  await ctx.reply(msg);
};

export const handleUpgrade = () => async (ctx: Context) => {
  const msg =
    "🎯 Upgrade to Premium\n\n" +
    "✨ Premium Benefits:\n" +
    "• Unlimited alerts\n" +
    "• Instant notifications\n" +
    "• Custom profit thresholds\n" +
    "• Website dashboard access\n" +
    "• Priority support\n\n" +
    "💰 Price: $29/month\n\n" +
    "To upgrade, please contact support.";

  await ctx.reply(msg);
};

export const handleMessageText = () => async (ctx: Context) => {
  await ctx.reply("Thanks for your message! 👋\n\nTry /help for commands.");
};
