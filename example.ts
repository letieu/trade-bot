// Supabase Edge Function for Bybit signals (Deno

import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { EMA, RSI } from "npm:technicalindicators@3.0.0";

const INTERVALS: Record<string, string | number> = {
  "1h": 60,
  "2h": 120,
  "4h": 240,
  "1d": "D",
};

const TELEGRAM_TOKEN = Deno.env.get("TELEGRAM_BOT_TOKEN")!;
const TELEGRAM_CHAT_ID = Deno.env.get("TELEGRAM_CHAT_ID")!;

/** Align timestamp to interval (UTC) */
export function getAlignedTimestamps(intervalLabel: string) {
  const now = new Date();
  let unitMS: number;

  if (intervalLabel === "1d" || intervalLabel === "D") {
    now.setUTCHours(0, 0, 0, 0);
    unitMS = 86400000;
  } else if (intervalLabel.endsWith("h")) {
    const hours = parseInt(intervalLabel);
    unitMS = hours * 60 * 60 * 1000;
    now.setUTCMinutes(0, 0, 0);
    const mod = now.getTime() % unitMS;
    now.setTime(now.getTime() - mod);
  } else {
    throw new Error("Unsupported interval format");
  }

  const nowTS = now.getTime();
  return [
    nowTS - 4 * unitMS,
    nowTS - 3 * unitMS,
    nowTS - 2 * unitMS,
    nowTS - 1 * unitMS,
  ];
}

/** Find nearest candles by timestamp (tolerant to misalignment) */
export function findNearestCandles(
  targetTSList: number[],
  allCandles: any[],
  intervalLabel: string,
) {
  const sorted = allCandles.sort((a, b) => parseInt(a[0]) - parseInt(b[0]));
  let tolerance = 60000; // default 1 min
  if (intervalLabel.endsWith("h")) {
    const hours = parseInt(intervalLabel);
    tolerance = (hours * 60 * 60 * 1000) / 2; // half interval
  } else if (intervalLabel === "1d" || intervalLabel === "D") {
    tolerance = 12 * 60 * 60 * 1000; // 12h tolerance
  }

  const result: any[] = [];
  for (const ts of targetTSList) {
    const candleIndex = sorted.findIndex(
      (c) => Math.abs(parseInt(c[0]) - ts) <= tolerance,
    );
    if (candleIndex !== -1) {
      result.push(sorted[candleIndex]);
      sorted.splice(candleIndex, 1);
    } else {
      console.warn(`⚠️ Missing candle near ${new Date(ts).toISOString()}`);
    }
  }
  return result.sort((a, b) => parseInt(a[0]) - parseInt(b[0]));
}

export async function getLinearSymbols() {
  const res = await fetch(
    "https://api.bybit.com/v5/market/instruments-info?category=linear",
  );
  const data = await res.json();
  return data.result.list
    .filter((i: any) => i.symbol.endsWith("USDT"))
    .map((i: any) => i.symbol);
}

export async function getExtraInfo(symbol: string, interval: string | number) {
  try {
    const [tickerRes, klineRes] = await Promise.all([
      fetch(
        `https://api.bybit.com/v5/market/tickers?category=linear&symbol=${symbol}`,
      ),
      fetch(
        `https://api.bybit.com/v5/market/kline?category=linear&symbol=${symbol}&interval=${interval}&limit=50`,
      ),
    ]);

    const [tickerData, klineData] = await Promise.all([
      tickerRes.json(),
      klineRes.json(),
    ]);

    if (tickerData.retCode !== 0 || !tickerData.result?.list?.length) {
      return { trend: "?", volume: "?", ema: "?", rsi: "?" };
    }
    if (klineData.retCode !== 0 || !klineData.result?.list?.length) {
      return { trend: "?", volume: "?", ema: "?", rsi: "?" };
    }

    const item = tickerData.result.list[0];
    const candles = klineData.result.list.sort(
      (a: any, b: any) => parseInt(a[0]) - parseInt(b[0]),
    );

    const closes = candles.map((c: any) => parseFloat(c[4]));
    const ema20 = EMA.calculate({ period: 20, values: closes });
    const rsi14 = RSI.calculate({ period: 14, values: closes });

    return {
      trend:
        parseFloat(item.lastPrice) > parseFloat(item.prevPrice24h)
          ? "⬆️"
          : "⬇️",
      volume: Number(item.turnover24h).toFixed(1),
      ema: ema20.length ? ema20.at(-1)!.toFixed(2) : "N/A",
      rsi: rsi14.length ? rsi14.at(-1)!.toFixed(2) : "N/A",
    };
  } catch (e) {
    console.error(`❌ getExtraInfo failed for ${symbol}:`, e);
    return { trend: "?", volume: "?", ema: "?", rsi: "?" };
  }
}

export async function matchPattern(
  symbol: string,
  intervalVal: string | number,
  intervalLabel: string,
  trendType: "bullish" | "bearish", // 👈 specify which pattern to look for
) {
  try {
    const res = await fetch(
      `https://api.bybit.com/v5/market/kline?category=linear&symbol=${symbol}&interval=${intervalVal}&limit=100`,
    );
    const data = await res.json();
    if (data.retCode !== 0 || !data.result?.list) return false;

    const targetTS = getAlignedTimestamps(intervalLabel);
    const candles = findNearestCandles(
      targetTS,
      data.result.list,
      intervalLabel,
    ).sort((a, b) => parseInt(a[0]) - parseInt(b[0]));

    if (candles.length !== 4) {
      console.warn(
        `⚠️ ${symbol} skipped, only got ${candles.length}/4 candles`,
      );
      return false;
    }

    // Determine candle colors: "green" (close >= open) or "red"
    const trends = candles.map((c) =>
      parseFloat(c[4]) >= parseFloat(c[1]) ? "green" : "red",
    );

    // Check pattern: first 3 same color, last opposite
    const first3Same = trends[0] === trends[1] && trends[1] === trends[2];
    const lastIsOpposite = trends[3] !== trends[2];

    if (!(first3Same && lastIsOpposite)) return false;

    // Now match only the requested pattern type
    if (
      trendType === "bullish" &&
      trends[0] === "red" &&
      trends[3] === "green"
    ) {
      console.log(`✅ ${symbol}: 🟥🟥🟥🟩 Bullish Reversal`);
      return true;
    }

    if (
      trendType === "bearish" &&
      trends[0] === "green" &&
      trends[3] === "red"
    ) {
      console.log(`✅ ${symbol}: 🟩🟩🟩🟥 Bearish Reversal`);
      return true;
    }

    return false;
  } catch (e) {
    console.error(`❌ matchPattern failed for ${symbol}:`, e);
    return false;
  }
}

export async function notifyTelegram(message: string) {
  const MAX_LENGTH = 4096;

  // Split into chunks of up to 4096 characters
  const chunks: string[] = [];
  for (let i = 0; i < message.length; i += MAX_LENGTH) {
    chunks.push(message.slice(i, i + MAX_LENGTH));
  }

  for (const chunk of chunks) {
    const res = await fetch(
      `https://api.telegram.org/bot${TELEGRAM_TOKEN}/sendMessage`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          chat_id: TELEGRAM_CHAT_ID,
          text: chunk,
          parse_mode: "Markdown",
        }),
      },
    );

    // Optional: log or handle any Telegram API errors
    if (!res.ok) {
      const errorText = await res.text();
      console.error("Telegram error:", errorText);
    }

    // To avoid Telegram flood limits (if many messages)
    await new Promise((r) => setTimeout(r, 300));
  }
}

export async function scan(
  intervalLabel: string,
  intervalVal: string | number,
  trendType: "bullish" | "bearish",
) {
  const symbols = await getLinearSymbols();
  const matchedSymbols: string[] = [];

  const batchSize = 20;
  for (let i = 0; i < symbols.length; i += batchSize) {
    const batch = symbols.slice(i, i + batchSize);
    const results = await Promise.allSettled(
      batch.map((s) => matchPattern(s, intervalVal, intervalLabel, trendType)),
    );
    results.forEach((r, idx) => {
      if (r.status === "fulfilled" && r.value) matchedSymbols.push(batch[idx]);
    });
    await new Promise((r) => setTimeout(r, 300)); // avoid rate limit
  }

  if (!matchedSymbols.length) {
    console.log(`❌ No signals for ${intervalLabel}`);
    return;
  }

  const extras = await Promise.allSettled(
    matchedSymbols.map((s) => getExtraInfo(s, intervalVal)),
  );

  const messages = matchedSymbols.map((symbol, i) => {
    const extra = extras[i].status === "fulfilled" ? extras[i].value : {};
    return `*${symbol}* ${extra.trend || ""} ‣ RSI: ${extra.rsi || "?"} ‣ EMA20: ${extra.ema || "?"} ‣ Vol: ${extra.volume || ""}`;
  });

  const trendIcon = trendType === "bullish" ? "🟥🟥🟥🟩" : "🟩🟩🟩🟥";

  await notifyTelegram(
    `${trendIcon} — ${intervalLabel}\n\n${messages.join("\n")}`,
  );
}

// Edge function entrypoint
Deno.serve(async () => {
  for (const [label, val] of Object.entries(INTERVALS)) {
    await scan(label, val, "bullish");
    await scan(label, val, "bearish");
  }
  return new Response(JSON.stringify({ status: "ok" }), {
    headers: { "Content-Type": "application/json" },
  });
});
