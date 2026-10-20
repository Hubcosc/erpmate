/**
 * Inventory API client + types.
 * Mirrors backend/internal/inventory (model.go, dto.go, handler.go).
 * No dependencies: uses fetch. Works with React Query, SWR, or plain useEffect.
 *
 *   const inventory = createInventoryApi({
 *     baseUrl: "/api/v1/inventory",           // wherever you mounted inventory Routes()
 *     getToken: () => localStorage.getItem("token"),
 *   });
 *   const { items, meta } = await inventory.listStock({ status: ["low_stock"], page: 1 });
 */

// ====================== ENUMS (see GET /lookups) ======================

export const STOCK_STATUSES = ["in_stock", "low_stock", "out_of_stock"] as const;
export const ALERT_TYPES = ["low_stock", "near_expiry", "expired"] as const;
export const ALERT_SEVERITIES = ["critical", "warning", "info"] as const;
export const COST_SOURCES = ["stock_batches", "inbound_movements", "no_cost_recorded"] as const;
export const EXPIRY_STATUSES = ["expired", "near_expiry", "ok", "no_expiry"] as const;

export type StockStatus = (typeof STOCK_STATUSES)[number];
export type AlertType = (typeof ALERT_TYPES)[number];
export type AlertSeverity = (typeof ALERT_SEVERITIES)[number];
export type CostSource = (typeof COST_SOURCES)[number];
export type ExpiryStatus = (typeof EXPIRY_STATUSES)[number];

export type StockSortKey = "name" | "code" | "warehouse" | "quantity" | "reserved" | "available";
export type ValuationSortKey = "name" | "code" | "warehouse" | "quantity" | "value" | "unit_cost";
export type SortOrder = "asc" | "desc";

/** Display labels for badges, tabs and dropdowns. */
export const STOCK_STATUS_LABEL: Record<StockStatus, string> = {
  in_stock: "In stock",
  low_stock: "Low stock",
  out_of_stock: "Out of stock",
};
export const ALERT_TYPE_LABEL: Record<AlertType, string> = {
  low_stock: "Low stock",
  near_expiry: "Near expiry",
  expired: "Expired",
};
export const SEVERITY_LABEL: Record<AlertSeverity, string> = {
  critical: "Critical",
  warning: "Warning",
  info: "Info",
};
export const COST_SOURCE_LABEL: Record<CostSource, string> = {
  stock_batches: "Batch cost",
  inbound_movements: "Receipt cost",
  no_cost_recorded: "No cost recorded",
};
export const EXPIRY_STATUS_LABEL: Record<ExpiryStatus, string> = {
  expired: "Expired",
  near_expiry: "Near expiry",
  ok: "OK",
  no_expiry: "No expiry",
};

// ====================== MODELS ======================
// uuid -> string, timestamps -> ISO-8601 string, dates -> "YYYY-MM-DD".
// Nullable Go pointers are `T | null` (always present in the JSON).

export interface StockBalance {
  item_id: string;
  item_code: string;
  item_name: string;
  item_type: string;
  unit_of_measure: string;
  category_name: string | null;
  warehouse_id: string;
  warehouse_code: string;
  warehouse_name: string;
  warehouse_type: string;
  quantity: number;
  reserved_quantity: number;
  available_quantity: number;
  min_stock_level: number;
  max_stock_level: number | null;
  reorder_level: number | null;
  /** Effective alert level: reorder_level, else min_stock_level. 0 = no alert level. */
  threshold: number;
  /** true when threshold > 0 and available <= threshold (also true at zero stock). Prefer stock_status for tabs. */
  low_stock: boolean;
  stock_status: StockStatus;
}

export interface ItemStock {
  item_id: string;
  item_code: string;
  item_name: string;
  item_type: string;
  unit_of_measure: string;
  category_name: string | null;
  min_stock_level: number;
  max_stock_level: number | null;
  reorder_level: number | null;
  is_batch_tracked: boolean;
  is_expiry_tracked: boolean;
  total_quantity: number;
  total_reserved: number;
  total_available: number;
  warehouses: StockBalance[];
}

export interface StockSummary {
  total_lines: number; // item x warehouse rows
  total_items: number; // distinct items
  in_stock: number;
  low_stock: number;
  out_of_stock: number;
}

export interface StockBatch {
  id: string;
  item_id: string;
  warehouse_id: string;
  warehouse_code: string;
  warehouse_name: string;
  batch_number: string;
  quantity: number;
  /** Earliest-expiry-first allocation of the warehouse's available stock. */
  available_quantity: number;
  manufacture_date: string | null;
  expiry_date: string | null;
  days_until_expiry: number | null; // negative = already expired
  expiry_status: ExpiryStatus;
  unit_cost: number | null;
  inventory_value: number | null;
  created_at: string;
}

export interface StockMovement {
  id: string;
  movement_type: string;
  reference_type: string | null;
  reference_id: string | null;
  item_id: string;
  item_code: string;
  item_name: string;
  unit_of_measure: string;
  warehouse_id: string;
  warehouse_name: string;
  batch_id: string | null;
  batch_number: string | null;
  quantity: number;
  unit_cost: number | null;
  total_value: number | null;
  notes: string | null;
  performed_by: string | null;
  performed_by_name: string | null;
  movement_date: string;
}

export interface StockValuation {
  item_id: string;
  item_code: string;
  item_name: string;
  item_type: string;
  unit_of_measure: string;
  warehouse_id: string;
  warehouse_name: string;
  warehouse_type: string;
  quantity: number;
  reserved_quantity: number;
  available_quantity: number;
  reorder_level: number | null;
  low_stock: boolean;
  stock_status: StockStatus;
  average_unit_cost: number;
  inventory_value: number;
  cost_source: CostSource;
}

export interface ValuationGroup {
  warehouse_type: string;
  lines: number;
  value: number;
}

export interface ValuationSummary {
  total_value: number;
  total_lines: number;
  lines_without_cost: number;
  by_warehouse_type: ValuationGroup[];
}

export interface InventoryAlert {
  id: string;
  alert_type: AlertType;
  severity: AlertSeverity;
  item_id: string;
  item_code: string;
  item_name: string;
  unit_of_measure: string;
  warehouse_id: string;
  warehouse_code: string;
  warehouse_name: string;
  batch_number: string | null;
  quantity: number | null;
  threshold: number | null;
  expiry_date: string | null;
  days_until_expiry: number | null;
  message: string;
  created_at: string;
}

export interface AlertSummary {
  total: number;
  low_stock: number;
  near_expiry: number;
  expired: number;
  critical: number;
  warning: number;
  info: number;
}

export interface Lookups {
  stock_statuses: StockStatus[];
  alert_types: AlertType[];
  alert_severities: AlertSeverity[];
  cost_sources: CostSource[];
  expiry_statuses: ExpiryStatus[];
}

// ====================== FILTERS ======================

export interface Paging {
  page?: number; // default 1
  limit?: number; // default 20, max 100
}

export interface StockFilter extends Paging {
  search?: string;
  item_type?: string;
  warehouse_type?: string;
  warehouse_id?: string;
  category_id?: string;
  status?: StockStatus[];
  sort?: StockSortKey;
  order?: SortOrder;
}

export interface BatchFilter {
  warehouse_type?: string;
  warehouse_id?: string;
  expiry_status?: ExpiryStatus[];
}

export interface MovementFilter extends Paging {
  item_id?: string;
  warehouse_id?: string;
  movement_type?: string[];
  reference_type?: string;
  search?: string;
  date_from?: string; // YYYY-MM-DD
  date_to?: string; // YYYY-MM-DD (inclusive)
}

export interface ValuationFilter extends Paging {
  search?: string;
  item_type?: string;
  warehouse_type?: string;
  warehouse_id?: string;
  cost_source?: CostSource[];
  low_stock?: boolean;
  sort?: ValuationSortKey;
  order?: SortOrder;
}

export interface AlertFilter extends Paging {
  alert_type?: AlertType[];
  severity?: AlertSeverity[];
  warehouse_id?: string;
  search?: string;
}

// ====================== RESULTS & ERRORS ======================

export interface PageMeta {
  page: number;
  limit: number;
  total: number;
}

export interface Paged<T> {
  items: T[];
  meta: PageMeta;
}

/** Thrown for every non-2xx response and network failure. */
export class ApiError extends Error {
  constructor(
    public status: number, // 0 = network error / aborted
    public code: string, // VALIDATION_ERROR | NOT_FOUND | UNAUTHORIZED | FORBIDDEN | INTERNAL_ERROR | NETWORK_ERROR
    message: string,
    /** Per-field messages for VALIDATION_ERROR, e.g. { status: "must be one of: ..." }. */
    public details: Record<string, string> | null = null,
  ) {
    super(message);
    this.name = "ApiError";
  }
  get isValidation() { return this.code === "VALIDATION_ERROR"; }
  get isNotFound() { return this.status === 404; }
  get isUnauthorized() { return this.status === 401; }
  get isForbidden() { return this.status === 403; }
}

// ====================== CLIENT ======================

export interface InventoryApiOptions {
  /** Where the inventory router is mounted, e.g. "/api/v1/inventory". */
  baseUrl: string;
  /** Return the JWT (or null). Called on every request. */
  getToken?: () => string | null | undefined;
  /** Called on 401 so you can redirect to login / refresh the token. */
  onUnauthorized?: () => void;
  fetchImpl?: typeof fetch;
}

type QueryValue = string | number | boolean | string[] | null | undefined;

function toQuery(params: Record<string, QueryValue>): string {
  const q = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") continue;
    if (Array.isArray(value)) {
      if (value.length > 0) q.set(key, value.join(",")); // backend expects comma-separated
    } else {
      q.set(key, String(value));
    }
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export function createInventoryApi(opts: InventoryApiOptions) {
  const doFetch = opts.fetchImpl ?? fetch;
  const base = opts.baseUrl.replace(/\/+$/, "");

  async function request(path: string, params: Record<string, QueryValue> = {}, signal?: AbortSignal) {
    const headers: Record<string, string> = { Accept: "application/json" };
    const token = opts.getToken?.();
    if (token) headers.Authorization = `Bearer ${token}`;

    let res: Response;
    try {
      res = await doFetch(`${base}${path}${toQuery(params)}`, { headers, signal });
    } catch (e) {
      if ((e as Error)?.name === "AbortError") throw new ApiError(0, "ABORTED", "Request cancelled");
      throw new ApiError(0, "NETWORK_ERROR", "Cannot reach the server. Check your connection.");
    }

    let body: any = null;
    try { body = await res.json(); } catch { /* empty / non-JSON body */ }

    if (!res.ok) {
      if (res.status === 401) opts.onUnauthorized?.();
      const err = body?.error ?? body ?? {};
      throw new ApiError(
        res.status,
        err.code ?? `HTTP_${res.status}`,
        err.message ?? res.statusText ?? "Request failed",
        err.details ?? null,
      );
    }
    return body;
  }

  // The backend wraps payloads as { data, meta? }. If your response helper differs,
  // this is the only place to adjust.
  const unwrap = <T>(body: any): T => (body && "data" in body ? body.data : body) as T;

  async function get<T>(path: string, params?: Record<string, QueryValue>, signal?: AbortSignal): Promise<T> {
    return unwrap<T>(await request(path, params, signal));
  }

  async function getPaged<T>(path: string, params: Record<string, QueryValue>, signal?: AbortSignal): Promise<Paged<T>> {
    const body = await request(path, params, signal);
    const items = (unwrap<T[]>(body) ?? []) as T[];
    const meta: PageMeta = body?.meta ?? { page: params.page as number ?? 1, limit: items.length, total: items.length };
    return { items, meta };
  }

  const enc = encodeURIComponent;

  return {
    lookups: (signal?: AbortSignal) => get<Lookups>("/lookups", {}, signal),

    // ---- stock ----
    listStock: (f: StockFilter = {}, signal?: AbortSignal) =>
      getPaged<StockBalance>("/stock", { ...f }, signal),
    /** Status counts for tabs/KPI cards. The status filter is ignored server-side. */
    stockSummary: (f: Omit<StockFilter, "status" | "page" | "limit" | "sort" | "order"> = {}, signal?: AbortSignal) =>
      get<StockSummary>("/stock/summary", { ...f }, signal),
    getItemStock: (itemId: string, signal?: AbortSignal) =>
      get<ItemStock>(`/stock/${enc(itemId)}`, {}, signal),
    listBatches: (itemId: string, f: BatchFilter = {}, signal?: AbortSignal) =>
      get<StockBatch[]>(`/stock/${enc(itemId)}/batches`, { ...f }, signal),

    // ---- movements ----
    listMovements: (f: MovementFilter = {}, signal?: AbortSignal) =>
      getPaged<StockMovement>("/movements", { ...f }, signal),
    movementTypes: (signal?: AbortSignal) => get<string[]>("/movements/types", {}, signal),

    // ---- valuation ----
    listValuation: (f: ValuationFilter = {}, signal?: AbortSignal) =>
      getPaged<StockValuation>("/valuation", { ...f }, signal),
    valuationSummary: (f: Omit<ValuationFilter, "page" | "limit" | "sort" | "order"> = {}, signal?: AbortSignal) =>
      get<ValuationSummary>("/valuation/summary", { ...f }, signal),

    // ---- alerts ----
    listAlerts: (f: AlertFilter = {}, signal?: AbortSignal) =>
      getPaged<InventoryAlert>("/alerts", { ...f }, signal),
    alertSummary: (f: Pick<AlertFilter, "warehouse_id" | "search"> = {}, signal?: AbortSignal) =>
      get<AlertSummary>("/alerts/summary", { ...f }, signal),
  };
}

export type InventoryApi = ReturnType<typeof createInventoryApi>;

// ====================== UI HELPERS ======================

/** Badge colour hint per status; map these to your design system. */
export const STOCK_STATUS_TONE: Record<StockStatus, "success" | "warning" | "danger"> = {
  in_stock: "success",
  low_stock: "warning",
  out_of_stock: "danger",
};
export const SEVERITY_TONE: Record<AlertSeverity, "danger" | "warning" | "info"> = {
  critical: "danger",
  warning: "warning",
  info: "info",
};

/** "1,234.5" - trims trailing zeros, max 4 decimals (backend precision). */
export function formatQty(n: number | null | undefined): string {
  if (n == null) return "-";
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 }).format(n);
}

/** Money with 2 decimals. Pass your currency code, e.g. "ETB". */
export function formatMoney(n: number | null | undefined, currency = "ETB"): string {
  if (n == null) return "-";
  return new Intl.NumberFormat(undefined, { style: "currency", currency }).format(n);
}

/** "Expired 3 days ago" / "Expires today" / "Expires in 12 days". */
export function describeExpiry(days: number | null | undefined): string {
  if (days == null) return "No expiry";
  if (days < 0) return `Expired ${-days} day${days === -1 ? "" : "s"} ago`;
  if (days === 0) return "Expires today";
  return `Expires in ${days} day${days === 1 ? "" : "s"}`;
}