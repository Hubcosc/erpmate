import { apiRequest } from './client'

export interface ReportEnvelope<T> { success: boolean; data: T }
export interface ReportPeriod { from: string; to: string }
export interface ReportSummary extends ReportPeriod {
  sales_revenue: number
  delivery_count: number
  units_delivered: number
  purchase_commitments: number
  pending_purchase_requests: number
  pending_material_requests: number
  low_stock_items: number
  near_expiry_batches: number
  expired_batches: number
}
export interface SalesReportRow { key: string; name: string; units_delivered: number; revenue: number }
export interface ConsumptionReportRow { item_id: string; item_code: string; item_name: string; unit_of_measure: string; quantity_issued: number; issue_count: number }
export interface PurchaseCommitmentRow { purchase_order_id: string; po_number: string; supplier_name?: string | null; status: string; ordered_value: number; received_value: number; outstanding_value: number }

function periodQuery(period: ReportPeriod) { return `from=${encodeURIComponent(period.from)}&to=${encodeURIComponent(period.to)}` }
export const getReportSummary = (period: ReportPeriod) => apiRequest<ReportEnvelope<ReportSummary>>(`/reports/summary?${periodQuery(period)}`)
export const getSalesReport = (period: ReportPeriod, groupBy: 'product' | 'customer' = 'product') => apiRequest<ReportEnvelope<SalesReportRow[]>>(`/reports/sales?${periodQuery(period)}&group_by=${groupBy}`)
export const getMaterialConsumptionReport = (period: ReportPeriod) => apiRequest<ReportEnvelope<ConsumptionReportRow[]>>(`/reports/material-consumption?${periodQuery(period)}`)
export const getPurchaseCommitmentsReport = () => apiRequest<ReportEnvelope<PurchaseCommitmentRow[]>>('/reports/purchase-commitments')