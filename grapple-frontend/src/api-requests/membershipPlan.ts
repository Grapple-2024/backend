import { membershipPlansApi } from "@/hook/base-apis";

export interface MembershipPlan {
  id?: string;
  gym_id: string;
  name: string;
  description: string;
  billing_type: 'recurring' | 'one_time';
  interval?: 'monthly' | 'yearly' | 'weekly';
  price: number;
  currency: string;
  class_limit: number | null;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getMembershipPlans = async (gymId: string, token: string): Promise<MembershipPlan[]> => {
  const { data } = await membershipPlansApi.get<MembershipPlan[]>('', {
    params: { gym_id: gymId },
    ...authHeaders(token),
  });
  return data;
};

export const createMembershipPlan = async (plan: Omit<MembershipPlan, 'id' | 'is_active' | 'created_at' | 'updated_at'>, token: string): Promise<MembershipPlan> => {
  const { data } = await membershipPlansApi.post<MembershipPlan>('', plan, authHeaders(token));
  return data;
};

export const updateMembershipPlan = async (gymId: string, planId: string, plan: Partial<MembershipPlan>, token: string): Promise<MembershipPlan> => {
  const { data } = await membershipPlansApi.put<MembershipPlan>(`/${gymId}/${planId}`, plan, authHeaders(token));
  return data;
};

export const deleteMembershipPlan = async (gymId: string, planId: string, token: string): Promise<void> => {
  await membershipPlansApi.delete(`/${gymId}/${planId}`, authHeaders(token));
};
