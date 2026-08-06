import { expect, test } from "@playwright/test";

test.describe("deployable demo", () => {
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
    await expect(page.locator("#assistant-form-hint")).toHaveText("现有查询和确定性计算仍可正常使用");
  });
});
