const elements = {
  status: document.querySelector("#system-status"),
  lookupForm: document.querySelector("#lookup-form"),
  lookupQuery: document.querySelector("#lookup-query"),
  lookupLimit: document.querySelector("#lookup-limit"),
  lookupMessage: document.querySelector("#lookup-message"),
  lookupResults: document.querySelector("#lookup-results"),
  calculationForm: document.querySelector("#calculation-form"),
  itemName: document.querySelector("#item-name"),
  buyPrice: document.querySelector("#buy-price"),
  quantity: document.querySelector("#quantity"),
  multiplier: document.querySelector("#multiplier"),
  sellPrices: document.querySelector("#sell-prices"),
  calculationMessage: document.querySelector("#calculation-message"),
  calculationResults: document.querySelector("#calculation-results"),
  resultSummary: document.querySelector("#result-summary"),
  refreshHistory: document.querySelector("#refresh-history"),
  historyForm: document.querySelector("#history-form"),
  historyItem: document.querySelector("#history-item"),
  historyOutcome: document.querySelector("#history-outcome"),
  historyOrder: document.querySelector("#history-order"),
  historyLimit: document.querySelector("#history-limit"),
  historyPrevious: document.querySelector("#history-previous"),
  historyNext: document.querySelector("#history-next"),
  historyPageSummary: document.querySelector("#history-page-summary"),
  historyMessage: document.querySelector("#history-message"),
  historyResults: document.querySelector("#history-results"),
  assistantStatus: document.querySelector("#assistant-status"),
  assistantMessages: document.querySelector("#assistant-messages"),
  assistantEvidence: document.querySelector("#assistant-evidence"),
  assistantForm: document.querySelector("#assistant-form"),
  assistantInput: document.querySelector("#assistant-input"),
  assistantSubmit: document.querySelector("#assistant-submit"),
  assistantStop: document.querySelector("#assistant-stop"),
  assistantRetry: document.querySelector("#assistant-retry"),
  assistantFormHint: document.querySelector("#assistant-form-hint"),
};

const profitColumns = [
  { label: "商品", value: (record) => record.item_name },
  { label: "买入价", value: (record) => formatPrice(record.buy_price), numeric: true },
  { label: "卖出价", value: (record) => formatPrice(record.sell_price), numeric: true },
  { label: "数量", value: (record) => String(record.quantity), numeric: true },
  { label: "单件收益", value: (record) => formatPrice(record.net_profit), numeric: true, signed: true },
  { label: "总收益", value: (record) => formatPrice(record.total_profit), numeric: true, signed: true },
  { label: "挂刀比例", value: (record) => Number(record.knife_ratio).toFixed(6), numeric: true },
];

const historyColumns = [
  { label: "记录", value: (record) => `#${record.id}`, numeric: true },
  { label: "保存时间", value: (record) => formatTimestamp(record.created_at) },
  ...profitColumns,
];

const historyState = {
  offset: 0,
  total: 0,
  limit: 20,
};

const assistantState = {
  history: [],
  maxHistory: 6,
  controller: undefined,
  retry: undefined,
  available: false,
};

async function requestJSON(path, options = {}) {
  const response = await fetch(path, options);
  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new Error("服务返回了无法识别的响应");
  }
  if (!response.ok) {
    const requestError = new Error(payload?.error?.message || `请求失败（${response.status}）`);
    requestError.code = payload?.error?.code || "request_error";
    throw requestError;
  }
  return payload.data;
}

async function readErrorResponse(response) {
  try {
    const payload = await response.json();
    return payload?.error?.message || `请求失败（${response.status}）`;
  } catch {
    return `请求失败（${response.status}）`;
  }
}

function setMessage(element, message = "", tone = "") {
  element.textContent = message;
  element.className = `message${tone ? ` ${tone}` : ""}`;
}

function setButtonBusy(button, busy, busyLabel) {
  if (!button.dataset.label) {
    button.dataset.label = button.textContent;
  }
  button.disabled = busy;
  button.textContent = busy ? busyLabel : button.dataset.label;
}

function formatPrice(value) {
  return Number(value).toFixed(3);
}

function formatTimestamp(value) {
  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return value || "—";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    dateStyle: "medium",
    timeStyle: "medium",
  }).format(timestamp);
}

function emptyState(message) {
  const state = document.createElement("div");
  state.className = "empty-state";
  state.textContent = message;
  return state;
}

function renderProfitTable(container, records, emptyMessage, columns = profitColumns) {
  container.replaceChildren();
  if (!records.length) {
    container.append(emptyState(emptyMessage));
    return;
  }

  const table = document.createElement("table");
  const head = document.createElement("thead");
  const headRow = document.createElement("tr");
  for (const column of columns) {
    const cell = document.createElement("th");
    cell.scope = "col";
    cell.textContent = column.label;
    headRow.append(cell);
  }
  head.append(headRow);

  const body = document.createElement("tbody");
  for (const record of records) {
    const row = document.createElement("tr");
    for (const column of columns) {
      const cell = document.createElement("td");
      const value = column.value(record);
      cell.textContent = value;
      if (column.numeric) {
        cell.classList.add("numeric-cell");
      }
      if (column.signed) {
        cell.classList.add(Number(value) >= 0 ? "positive-value" : "negative-value");
      }
      row.append(cell);
    }
    body.append(row);
  }

  table.append(head, body);
  container.append(table);
}

function renderLookupTable(items) {
  elements.lookupResults.replaceChildren();
  if (!items.length) {
    elements.lookupResults.append(emptyState("没有找到匹配商品"));
    return;
  }

  const table = document.createElement("table");
  const head = document.createElement("thead");
  const headRow = document.createElement("tr");
  for (const label of ["商品", "应用", "参考售价", "挂牌价", "挂牌数", ""]) {
    const cell = document.createElement("th");
    cell.scope = "col";
    cell.textContent = label;
    headRow.append(cell);
  }
  head.append(headRow);

  const body = document.createElement("tbody");
  for (const item of items) {
    const row = document.createElement("tr");
    const values = [
      item.canonical_name,
      item.app_name,
      item.sale_price_text || "—",
      item.sell_price_text || "—",
      String(item.sell_listings),
    ];
    values.forEach((value, index) => {
      const cell = document.createElement("td");
      cell.textContent = value;
      if (index >= 2) {
        cell.classList.add("numeric-cell");
      }
      row.append(cell);
    });

    const actionCell = document.createElement("td");
    const useButton = document.createElement("button");
    useButton.type = "button";
    useButton.className = "use-item-button";
    useButton.textContent = "用于计算";
    useButton.addEventListener("click", () => {
      elements.itemName.value = item.canonical_name;
      elements.buyPrice.focus();
      setMessage(elements.lookupMessage, `已带入：${item.canonical_name}`, "success");
    });
    actionCell.append(useButton);
    row.append(actionCell);
    body.append(row);
  }

  table.append(head, body);
  elements.lookupResults.append(table);
}

function parseSellPrices(raw) {
  const prices = raw
    .split(/[\s,，]+/)
    .map((value) => value.trim())
    .filter(Boolean)
    .map(Number);
  if (!prices.length || prices.some((value) => !Number.isFinite(value) || value <= 0)) {
    throw new Error("请填写一个或多个有效的正数卖价");
  }
  return prices;
}

function updateResultSummary(records) {
  if (!records.length) {
    elements.resultSummary.textContent = "等待新的计算场景";
    return;
  }
  const best = records.reduce((current, record) => (
    record.total_profit > current.total_profit ? record : current
  ));
  elements.resultSummary.replaceChildren();
  const label = document.createElement("span");
  label.textContent = `共 ${records.length} 个场景 · 最佳总收益 `;
  const value = document.createElement("strong");
  value.textContent = formatPrice(best.total_profit);
  elements.resultSummary.append(label, value);
}

async function checkHealth() {
  try {
    await requestJSON("/api/health");
    elements.status.classList.add("online");
    elements.status.classList.remove("offline");
    elements.status.lastChild.textContent = " 计算引擎已连接";
  } catch {
    elements.status.classList.add("offline");
    elements.status.classList.remove("online");
    elements.status.lastChild.textContent = " 计算引擎不可用";
  }
}

function setAssistantAvailability(status) {
  assistantState.available = Boolean(status.available);
  assistantState.maxHistory = Number(status.max_history_messages) || 6;
  elements.assistantStatus.classList.toggle("available", status.available);
  elements.assistantStatus.classList.toggle("unavailable", !status.available);
  const statusText = elements.assistantStatus.querySelector("span:last-child");
  if (status.available) {
    statusText.textContent = `${status.provider} · ${status.model}`;
    elements.assistantInput.disabled = false;
    elements.assistantSubmit.disabled = false;
    elements.assistantFormHint.textContent = `提示词 ${status.prompt_version}${status.native_streaming ? " · 原生流安全缓冲" : ""}`;
    return;
  }
  statusText.textContent = status.reason || "AI 策略助手未启用";
  elements.assistantInput.disabled = true;
  elements.assistantSubmit.disabled = true;
  elements.assistantStop.hidden = true;
  elements.assistantRetry.hidden = true;
  elements.assistantFormHint.textContent = "AI 为可选增强：接入 Provider / API Key 后即可启用，查询、计算和历史不受影响";
}

async function checkAssistantStatus() {
  try {
    const status = await requestJSON("/api/v1/assistant/status");
    setAssistantAvailability(status);
  } catch {
    setAssistantAvailability({ available: false, reason: "无法读取模型状态" });
  }
}

function appendChatMessage(role, text = "") {
  const message = document.createElement("div");
  message.className = `chat-message ${role === "user" ? "user-message" : "assistant-message"}`;
  const label = document.createElement("span");
  label.className = "message-role";
  label.textContent = role === "user" ? "你" : "策略助手";
  const copy = document.createElement("div");
  copy.className = "message-copy";
  copy.textContent = text;
  message.append(label, copy);
  elements.assistantMessages.append(message);
  message.scrollIntoView({ block: "nearest", behavior: "smooth" });
  return { message, copy };
}

function toolLabel(tool) {
  return {
    lookup_items: "市场查询",
    calculate_profit: "收益计算",
    solve_target: "目标反推",
  }[tool] || tool;
}

function appendToolProgress(tool) {
  const progress = document.createElement("div");
  progress.className = "tool-progress";
  progress.textContent = `正在执行 · ${toolLabel(tool)}`;
  elements.assistantMessages.append(progress);
  return progress;
}

function appendEvidence(evidence) {
  const card = document.createElement("article");
  card.className = `evidence-card ${evidence.verified ? "verified" : "failed"}`;

  const header = document.createElement("div");
  header.className = "evidence-header";
  const title = document.createElement("strong");
  title.textContent = `${evidence.id} · ${toolLabel(evidence.tool)}`;
  const state = document.createElement("span");
  state.textContent = evidence.verified ? "程序已验证" : "工具未完成";
  header.append(title, state);

  const body = document.createElement("div");
  body.className = "evidence-body";
  if (Array.isArray(evidence.data?.records)) {
    renderProfitTable(body, evidence.data.records, "工具没有返回记录");
  } else if (Array.isArray(evidence.data?.items)) {
    for (const item of evidence.data.items) {
      const itemRow = document.createElement("div");
      itemRow.className = "evidence-item";
      const itemName = document.createElement("strong");
      itemName.textContent = item.canonical_name;
      const itemMeta = document.createElement("span");
      itemMeta.textContent = `${item.app_name || "未知应用"} · 挂牌 ${item.sell_listings} · ${item.sell_price_text || "暂无挂牌价"}`;
      const queryTime = document.createElement("small");
      queryTime.textContent = `查询时间 ${item.query_time || "未知"}`;
      itemRow.append(itemName, itemMeta, queryTime);
      body.append(itemRow);
    }
  } else if (evidence.data?.error) {
    const error = document.createElement("p");
    error.textContent = evidence.data.error.message || "工具执行失败";
    body.append(error);
  }
  card.append(header, body);
  elements.assistantEvidence.append(card);
}

function renderCompletion(completion) {
  const meta = document.createElement("div");
  meta.className = `assistant-completion ${completion.grounded ? "grounded" : "guarded"}`;
  const tokenText = completion.usage?.total_tokens
    ? ` · ${completion.usage.total_tokens} tokens`
    : "";
  const streamText = completion.native_streaming ? " · Provider 原生流" : "";
  meta.textContent = `${completion.grounded ? "数值溯源通过" : "已启用安全替代"} · ${completion.tool_count} 次工具 · ${completion.elapsed_ms} ms${tokenText}${streamText}`;
  elements.assistantMessages.append(meta);
}

function parseSSEBlock(block) {
  let eventName = "message";
  const data = [];
  for (const line of block.split("\n")) {
    if (line.startsWith("event:")) {
      eventName = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      data.push(line.slice(5).trimStart());
    }
  }
  if (!data.length) {
    return null;
  }
  return { eventName, payload: JSON.parse(data.join("\n")) };
}

async function streamAssistant(message, history, signal, onEvent) {
  const response = await fetch("/api/v1/assistant/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "text/event-stream" },
    body: JSON.stringify({ message, history }),
    signal,
  });
  if (!response.ok) {
    throw new Error(await readErrorResponse(response));
  }
  if (!response.body) {
    throw new Error("当前浏览器无法读取流式响应");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done }).replaceAll("\r\n", "\n");
    let boundary = buffer.indexOf("\n\n");
    while (boundary >= 0) {
      const block = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);
      const event = parseSSEBlock(block);
      if (event) {
        onEvent(event.eventName, event.payload);
      }
      boundary = buffer.indexOf("\n\n");
    }
    if (done) {
      break;
    }
  }
}

function boundedAssistantHistory(history) {
  return history.slice(-assistantState.maxHistory).map((message) => ({ ...message }));
}

function setAssistantBusy(busy) {
  elements.assistantInput.disabled = busy || !assistantState.available;
  setButtonBusy(elements.assistantSubmit, busy, "分析中");
  elements.assistantSubmit.disabled = busy || !assistantState.available;
  elements.assistantStop.hidden = !busy;
}

function rememberAssistantTurn(userText, assistantText) {
  assistantState.history = boundedAssistantHistory([
    ...assistantState.history,
    { role: "user", content: userText },
    { role: "assistant", content: assistantText },
  ]);
}

async function runAssistantTurn(message, history) {
  const priorHistory = boundedAssistantHistory(history);
  appendChatMessage("user", message);
  const assistantMessage = appendChatMessage("assistant", "");
  assistantMessage.message.classList.add("streaming");
  assistantMessage.copy.textContent = "正在连接策略模型…";
  elements.assistantEvidence.replaceChildren();
  elements.assistantInput.value = "";
  elements.assistantRetry.hidden = true;
  assistantState.retry = undefined;
  assistantState.controller = new AbortController();
  setAssistantBusy(true);
  let hasDelta = false;
  let completed = false;
  let toolProgress;

  try {
    await streamAssistant(message, priorHistory, assistantState.controller.signal, (eventName, payload) => {
      if (eventName === "status") {
        assistantMessage.copy.textContent = payload.message;
      } else if (eventName === "tool_start") {
        toolProgress?.remove();
        toolProgress = appendToolProgress(payload.tool);
      } else if (eventName === "evidence") {
        toolProgress?.remove();
        toolProgress = undefined;
        appendEvidence(payload.evidence);
        assistantMessage.copy.textContent = "正在验证完整回答与工具依据…";
      } else if (eventName === "delta") {
        if (!hasDelta) {
          assistantMessage.copy.textContent = "";
          hasDelta = true;
        }
        assistantMessage.copy.textContent += payload.text;
      } else if (eventName === "done") {
        completed = true;
        renderCompletion(payload.completion);
      } else if (eventName === "error") {
        const streamError = new Error(payload?.error?.message || "AI 策略助手暂时不可用");
        streamError.retryable = Boolean(payload?.error?.retryable);
        throw streamError;
      }
    });
    if (!completed) {
      throw new Error("AI 策略助手未完成本次请求");
    }
    rememberAssistantTurn(message, assistantMessage.copy.textContent);
  } catch (error) {
    toolProgress?.remove();
    assistantMessage.message.classList.add("error-message");
    const cancelled = error.name === "AbortError";
    assistantMessage.copy.textContent = cancelled ? "请求已停止，未保存对话或计算记录。" : error.message;
    if (cancelled || error.retryable || error instanceof TypeError) {
      assistantState.retry = { message, history: priorHistory };
      elements.assistantRetry.hidden = false;
    }
  } finally {
    assistantMessage.message.classList.remove("streaming");
    assistantState.controller = undefined;
    setAssistantBusy(false);
    elements.assistantInput.focus();
  }
}

async function loadHistory() {
  setButtonBusy(elements.refreshHistory, true, "加载中");
  setMessage(elements.historyMessage, "正在读取服务端记录…");
  try {
    const parameters = new URLSearchParams({
      outcome: elements.historyOutcome.value,
      order: elements.historyOrder.value,
      limit: elements.historyLimit.value,
      offset: String(historyState.offset),
    });
    const item = elements.historyItem.value.trim();
    if (item) {
      parameters.set("item", item);
    }
    const data = await requestJSON(`/api/v1/records?${parameters}`);
    historyState.total = data.pagination.total;
    historyState.limit = data.pagination.limit;
    historyState.offset = data.pagination.offset;
    renderProfitTable(
      elements.historyResults,
      data.records,
      "当前筛选下没有记录。完成一次计算或调整筛选条件后再试。",
      historyColumns,
    );
    updateHistoryPagination(data.pagination, data.records.length);
    setMessage(elements.historyMessage, `服务端共找到 ${data.pagination.total} 条匹配记录`, "success");
  } catch (error) {
    renderProfitTable(elements.historyResults, [], "历史记录暂时不可用", historyColumns);
    elements.historyPrevious.disabled = true;
    elements.historyNext.disabled = true;
    elements.historyPageSummary.textContent = "无法读取分页状态";
    setMessage(elements.historyMessage, error.message, "error");
  } finally {
    setButtonBusy(elements.refreshHistory, false, "");
  }
}

function updateHistoryPagination(pagination, recordCount) {
  const start = recordCount ? pagination.offset + 1 : 0;
  const end = pagination.offset + recordCount;
  elements.historyPageSummary.textContent = `第 ${start}–${end} 条 · 共 ${pagination.total} 条`;
  elements.historyPrevious.disabled = pagination.offset === 0;
  elements.historyNext.disabled = !pagination.has_more;
}

elements.lookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = elements.lookupForm.querySelector("button[type='submit']");
  setButtonBusy(button, true, "查询中");
  setMessage(elements.lookupMessage, "正在查询市场数据 Provider…");
  try {
    const query = encodeURIComponent(elements.lookupQuery.value.trim());
    const limit = encodeURIComponent(elements.lookupLimit.value);
    const data = await requestJSON(`/api/v1/items?query=${query}&limit=${limit}`);
    renderLookupTable(data.items);
    setMessage(elements.lookupMessage, `找到 ${data.items.length} 个候选商品`, "success");
  } catch (error) {
    const officialRateLimited = error.code === "official_provider_rate_limited";
    const providerFailed = officialRateLimited || error.code === "provider_error";
    elements.lookupResults.replaceChildren(emptyState(providerFailed
      ? officialRateLimited
        ? "Steam 官方市场 API 当前处于限流状态；可接入第三方 Steam Market Provider 继续查询"
        : "当前市场数据 Provider 未返回结果；收益计算、目标反推和历史记录仍可继续使用"
      : "查询没有返回可用结果"));
    setMessage(elements.lookupMessage, officialRateLimited
      ? "Steam 官方市场 API 当前处于限流状态；建议接入可用的第三方 Steam Market Provider（如 SteamApis），以有效查询虚拟交易物品信息"
      : providerFailed
        ? "当前市场数据 Provider 未完成查询；请检查 Provider 或 API Key，网站核心功能不受影响"
      : error.message, "error");
  } finally {
    setButtonBusy(button, false, "");
  }
});

elements.calculationForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = elements.calculationForm.querySelector("button[type='submit']");
  setButtonBusy(button, true, "计算中");
  setMessage(elements.calculationMessage, "正在由服务端验证并计算…");
  try {
    const payload = {
      item_name: elements.itemName.value.trim(),
      buy_price: Number(elements.buyPrice.value),
      quantity: Number(elements.quantity.value),
      multiplier: Number(elements.multiplier.value),
      sell_prices: parseSellPrices(elements.sellPrices.value),
    };
    const data = await requestJSON("/api/v1/calculations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    renderProfitTable(elements.calculationResults, data.records, "暂无计算结果");
    updateResultSummary(data.records);
    setMessage(elements.calculationMessage, `已验证并保存 ${data.records.length} 个场景`, "success");
    historyState.offset = 0;
    await loadHistory();
  } catch (error) {
    setMessage(elements.calculationMessage, error.message, "error");
  } finally {
    setButtonBusy(button, false, "");
  }
});

elements.refreshHistory.addEventListener("click", loadHistory);

elements.historyForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  historyState.offset = 0;
  await loadHistory();
});

elements.historyPrevious.addEventListener("click", async () => {
  historyState.offset = Math.max(0, historyState.offset - historyState.limit);
  await loadHistory();
});

elements.historyNext.addEventListener("click", async () => {
  if (historyState.offset + historyState.limit < historyState.total) {
    historyState.offset += historyState.limit;
    await loadHistory();
  }
});

document.querySelectorAll("[data-assistant-prompt]").forEach((button) => {
  button.addEventListener("click", () => {
    elements.assistantInput.value = button.dataset.assistantPrompt;
    elements.assistantInput.focus();
  });
});

elements.assistantForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const message = elements.assistantInput.value.trim();
  if (!message) {
    return;
  }

  await runAssistantTurn(message, assistantState.history);
});

elements.assistantStop.addEventListener("click", () => {
  assistantState.controller?.abort();
});

elements.assistantRetry.addEventListener("click", async () => {
  const retry = assistantState.retry;
  if (!retry || assistantState.controller) {
    return;
  }
  await runAssistantTurn(retry.message, retry.history);
});

renderLookupTable([]);
renderProfitTable(elements.calculationResults, [], "输入条件后，本次计算结果会显示在这里。");
renderProfitTable(elements.historyResults, [], "正在载入历史记录…", historyColumns);
checkHealth();
checkAssistantStatus();
loadHistory();
