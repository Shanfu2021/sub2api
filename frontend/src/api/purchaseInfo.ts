import { apiClient } from './client'
import type { PurchaseInfoCard, PurchaseInfoCardInput } from '@/types'

const BASE_PATH = '/purchase-info'

export async function listVisibleCards(): Promise<PurchaseInfoCard[]> {
  const { data } = await apiClient.get<PurchaseInfoCard[]>(BASE_PATH)
  return data
}

export async function listManagedCards(): Promise<PurchaseInfoCard[]> {
  const { data } = await apiClient.get<PurchaseInfoCard[]>(`${BASE_PATH}/manage`)
  return data
}

export async function createCard(payload: PurchaseInfoCardInput): Promise<PurchaseInfoCard> {
  const { data } = await apiClient.post<PurchaseInfoCard>(`${BASE_PATH}/manage`, payload)
  return data
}

export async function updateCard(cardId: number, payload: PurchaseInfoCardInput): Promise<PurchaseInfoCard> {
  const { data } = await apiClient.put<PurchaseInfoCard>(`${BASE_PATH}/manage/${cardId}`, payload)
  return data
}

export async function deleteCard(cardId: number): Promise<{ id: number }> {
  const { data } = await apiClient.delete<{ id: number }>(`${BASE_PATH}/manage/${cardId}`)
  return data
}

export const purchaseInfoAPI = {
  listVisibleCards,
  listManagedCards,
  createCard,
  updateCard,
  deleteCard,
}

export default purchaseInfoAPI
