import { expect, request as playwrightRequest, test } from "@playwright/test";

function assistantStatus() {
  return {
    data: {
      available: true,
      provider: "fixture",
      model: "fixture-stream",
      prompt_version: "steam-profit-assistant-v1.1.0",
      native_streaming: true,
      retry_supported: true,
      max_history_messages: 6,
    },
  };
}

function assistantSSE(events) {
  return `${events.map(({ name, payload }) => `event: ${name}\ndata: ${JSON.stringify(payload)}\n\n`).join("")}`;
}

function successfulAssistantSSE(text = "已完成可信回答。") {
  return assistantSSE([
    { name: "status", payload: { message: "正在生成回答…" } },
    { name: "delta", payload: { text } },
    {
      name: "done",
      payload: {
        completion: {
          provider: "fixture",
          model: "fixture-stream",
          prompt_version: "steam-profit-assistant-v1.1.0",
          grounded: true,
          native_streaming: true,
          tool_count: 0,
          elapsed_ms: 12,
          usage: { input_tokens: 8, output_tokens: 4, total_tokens: 12 },
        },
      },
    },
  ]);
}

test.describe("deployable demo", () => {
  test("requires demo credentials while keeping dependency health public", async ({ request }) => {
    const anonymous = await playwrightRequest.newContext({
      baseURL: "http://127.0.0.1:18080",
      httpCredentials: { username: "anonymous", password: "invalid-password", send: "always" },
    });
    try {
      const rejected = await anonymous.get("/");
      expect(rejected.status()).toBe(401);
      expect(rejected.headers()["www-authenticate"]).toContain("Basic");

      const health = await anonymous.get("/api/health");
      expect(health.ok()).toBeTruthy();
      expect(await health.json()).toMatchObject({ data: { status: "ok" } });
    } finally {
      await anonymous.dispose();
    }

    const protectedPage = await request.get("/");
    expect(protectedPage.ok()).toBeTruthy();
    expect(await protectedPage.text()).toContain("Steam Profit Lab");

    const metrics = await request.get("/api/v1/metrics");
    expect(metrics.ok()).toBeTruthy();
    expect((await metrics.json()).data.access.unauthorized).toBeGreaterThanOrEqual(1);
  });

  test("persists a calculation across a browser reload", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("#system-status")).toContainText("计算引擎已连接");

    await page.locator("#item-name").fill("E2E Persisted Case");
    await page.locator("#buy-price").fill("0.48");
    await page.locator("#quantity").fill("2");
    await page.locator("#multiplier").fill("0.837");
    await page.locator("#sell-prices").fill("0.45, 0.80");
    await page.getByRole("button", { name: "计算并保存" }).click();

    await expect(page.locator("#calculation-message")).toContainText("已验证并保存 2 个场景");
    await expect(page.locator("#calculation-results tbody tr")).toHaveCount(2);
    await expect(page.locator("#result-summary")).toContainText("共 2 个场景");

    await page.reload();
    await page.locator("#history-item").fill("E2E Persisted Case");
    await page.getByRole("button", { name: "应用筛选" }).click();
    await expect(page.locator("#history-message")).toContainText("服务端共找到 2 条匹配记录");
    await expect(page.locator("#history-results tbody tr")).toHaveCount(2);
    await expect(page.locator("#history-results")).toContainText("E2E Persisted Case");
  });

  test("filters and paginates stable seed history", async ({ page }) => {
    await page.goto("/");
    await page.locator("#history-item").fill("Demo Capsule");
    await page.locator("#history-limit").selectOption("10");
    await page.getByRole("button", { name: "应用筛选" }).click();

    await expect(page.locator("#history-page-summary")).toHaveText("第 1–10 条 · 共 12 条");
    await expect(page.locator("#history-results tbody tr")).toHaveCount(10);
    await expect(page.locator("#history-next")).toBeEnabled();
    await page.locator("#history-next").click();
    await expect(page.locator("#history-page-summary")).toHaveText("第 11–12 条 · 共 12 条");
    await expect(page.locator("#history-results tbody tr")).toHaveCount(2);

    await page.locator("#history-outcome").selectOption("loss");
    await page.getByRole("button", { name: "应用筛选" }).click();
    await expect(page.locator("#history-page-summary")).toHaveText("第 1–1 条 · 共 1 条");
    await expect(page.locator("#history-results tbody tr")).toHaveCount(1);
    await expect(page.locator("#history-results")).toContainText("-0.207");
  });

  test("shows validation errors without creating history", async ({ page }) => {
    await page.goto("/");
    await page.locator("#item-name").fill("Invalid E2E Case");
    await page.locator("#buy-price").fill("0.48");
    await page.locator("#sell-prices").fill("invalid");
    await page.getByRole("button", { name: "计算并保存" }).click();
    await expect(page.locator("#calculation-message")).toContainText("请填写一个或多个有效的正数卖价");

    await page.locator("#history-item").fill("Invalid E2E Case");
    await page.getByRole("button", { name: "应用筛选" }).click();
    await expect(page.locator("#history-page-summary")).toHaveText("第 0–0 条 · 共 0 条");
    await expect(page.locator("#history-results")).toContainText("当前筛选下没有记录");
  });

  test("keeps deterministic workflows available when AI is disabled", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("#assistant-status")).toContainText("AI 策略助手未启用");
    await expect(page.locator("#assistant-input")).toBeDisabled();
    await expect(page.locator("#assistant-submit")).toBeDisabled();
    await expect(page.locator("#calculation-form button[type='submit']")).toBeEnabled();
    await expect(page.locator("#assistant-form-hint")).toHaveText("AI 为可选增强：接入 Provider / API Key 后即可启用，查询、计算和历史不受影响");

    await page.route("**/api/v1/items?**", (route) => route.fulfill({
      status: 502,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "official_provider_rate_limited", message: "Steam official market API is rate limited" } }),
    }));
    await page.locator("#lookup-query").fill("AK-47 Redline");
    await page.getByRole("button", { name: "查询市场", exact: true }).click();
    await expect(page.locator("#lookup-message")).toHaveText("Steam 官方市场 API 当前处于限流状态；建议接入可用的第三方 Steam Market Provider（如 SteamApis），以有效查询虚拟交易物品信息");
    await expect(page.locator("#lookup-results")).toContainText("可接入第三方 Steam Market Provider 继续查询");
    await expect(page.locator("#calculation-form button[type='submit']")).toBeEnabled();
  });

  test("keeps bounded page-local context and explicitly retries a failed turn", async ({ page }) => {
    await page.route("**/api/v1/assistant/status", (route) => route.fulfill({ json: assistantStatus() }));
    const requests = [];
    await page.route("**/api/v1/assistant/stream", async (route) => {
      const payload = route.request().postDataJSON();
      requests.push(payload);
      if (requests.length === 2) {
        await route.fulfill({
          status: 200,
          contentType: "text/event-stream; charset=utf-8",
          body: assistantSSE([{ name: "error", payload: { error: { code: "assistant_error", message: "临时失败", retryable: true } } }]),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: "text/event-stream; charset=utf-8", body: successfulAssistantSSE() });
    });

    await page.goto("/");
    await expect(page.locator("#assistant-status")).toContainText("fixture-stream");
    await page.locator("#assistant-input").fill("第一轮问题");
    await page.locator("#assistant-submit").click();
    await expect(page.locator(".assistant-completion").last()).toContainText("Provider 原生流");

    await page.locator("#assistant-input").fill("继续上一轮");
    await page.locator("#assistant-submit").click();
    await expect(page.locator(".error-message").last()).toContainText("临时失败");
    await expect(page.locator("#assistant-retry")).toBeVisible();
    await page.locator("#assistant-retry").click();
    await expect(page.locator(".assistant-completion")).toHaveCount(2);

    expect(requests).toHaveLength(3);
    expect(requests[0].history).toEqual([]);
    expect(requests[1].history).toEqual([
      { role: "user", content: "第一轮问题" },
      { role: "assistant", content: "已完成可信回答。" },
    ]);
    expect(requests[2]).toEqual(requests[1]);
  });

  test("stops an active assistant request without affecting calculation history", async ({ page }) => {
    await page.route("**/api/v1/assistant/status", (route) => route.fulfill({ json: assistantStatus() }));
    let attempts = 0;
    await page.route("**/api/v1/assistant/stream", async (route) => {
      attempts += 1;
      if (attempts === 1) {
        await new Promise((resolve) => setTimeout(resolve, 1200));
      }
      try {
        await route.fulfill({ status: 200, contentType: "text/event-stream; charset=utf-8", body: successfulAssistantSSE() });
      } catch {
        // The first intercepted request is expected to be aborted by the page.
      }
    });

    await page.goto("/");
    const initialSummary = await page.locator("#history-page-summary").textContent();
    await page.locator("#assistant-input").fill("停止这个请求");
    await page.locator("#assistant-submit").click();
    await expect(page.locator("#assistant-stop")).toBeVisible();
    await page.locator("#assistant-stop").click();
    await expect(page.locator(".error-message").last()).toContainText("请求已停止");
    await expect(page.locator("#assistant-retry")).toBeVisible();
    await expect(page.locator("#assistant-input")).toBeEnabled();

    await page.locator("#refresh-history").click();
    await expect(page.locator("#history-page-summary")).toHaveText(initialSummary);
  });
});
