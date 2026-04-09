import { promotionsApi } from "@/hook/base-apis";

export const ADULT_BELTS = ['white', 'blue', 'purple', 'brown', 'black', 'coral', 'red/white', 'red'] as const;
export const KIDS_BELTS = [
  'white',
  'grey/white', 'grey', 'grey/black',
  'yellow/white', 'yellow', 'yellow/black',
  'orange/white', 'orange', 'orange/black',
  'green/white', 'green', 'green/black',
] as const;

export type AdultBelt = typeof ADULT_BELTS[number];
export type KidsBelt = typeof KIDS_BELTS[number];
export type BeltSystem = 'adult' | 'kids';

export interface Promotion {
  id?: string;
  gym_id: string;
  member_id: string;
  member_name: string;
  avatar_url?: string;
  system: BeltSystem;
  belt: string;
  stripes: number;
  notes?: string;
  promoted_by?: string;
  promoted_at: string;
  created_at?: string;
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getPromotionHistory = async (gymId: string, memberId: string, token: string): Promise<Promotion[]> => {
  const { data } = await promotionsApi.get<Promotion[]>('', {
    params: { gym_id: gymId, member_id: memberId },
    ...authHeaders(token),
  });
  return data;
};

export const getCurrentBelts = async (gymId: string, token: string): Promise<Record<string, Promotion>> => {
  const { data } = await promotionsApi.get<Record<string, Promotion>>('/current', {
    params: { gym_id: gymId },
    ...authHeaders(token),
  });
  return data;
};

export const recordPromotion = async (
  payload: Omit<Promotion, 'id' | 'created_at'>,
  token: string
): Promise<Promotion> => {
  const { data } = await promotionsApi.post<Promotion>('', payload, authHeaders(token));
  return data;
};

export const deletePromotion = async (gymId: string, promotionId: string, token: string): Promise<void> => {
  await promotionsApi.delete(`/${gymId}/${promotionId}`, authHeaders(token));
};
