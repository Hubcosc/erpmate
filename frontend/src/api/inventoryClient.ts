import { createInventoryApi } from './inventory'
import { getApiToken } from './client'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

/** Shared inventory API instance. The token is read from the active session,
 *  so it works on the first paint after a browser refresh. */
export const inventoryApi = createInventoryApi({
  baseUrl: `${API_BASE_URL}/inventory`,
  getToken: getApiToken,
})
