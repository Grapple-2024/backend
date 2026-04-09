import { memberBillingApi } from "@/hook/base-apis";

export interface MemberBilling {
  id?: string;
  gym_id: string;
  member_id: string;
  plan_id: string;
  plan_name: string;
  member_name: string;
  status: 'active' | 'paused' | 'cancelled';
  start_date: string;
  next_payment_date: string;
  stripe_customer_id?: string;
  stripe_subscription_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PaymentRecord {
  id?: string;
  gym_id: string;
  member_id: string;
  billing_id: string;
  plan_id: string;
  plan_name: string;
  member_name: string;
  amount: number; // cents
  currency: string;
  status: 'unpaid' | 'paid' | 'overdue';
  due_date: string;
  paid_at?: string | null;
  notes?: string;
  stripe_payment_intent_id?: string;
  stripe_invoice_id?: string;
  created_at?: string;
  updated_at?: string;
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getMemberBilling = async (gymId: string, token: string, memberId?: string): Promise<MemberBilling[]> => {
  const { data } = await memberBillingApi.get<MemberBilling[]>('', {
    params: { gym_id: gymId, ...(memberId ? { member_id: memberId } : {}) },
    ...authHeaders(token),
  });
  return data;
};

export const assignPlan = async (
  payload: { gym_id: string; member_id: string; plan_id: string; member_name: string; start_date?: string },
  token: string
): Promise<{ billing: MemberBilling; payment: PaymentRecord }> => {
  const { data } = await memberBillingApi.post('', payload, authHeaders(token));
  return data;
};

export const updateBillingStatus = async (
  gymId: string,
  billingId: string,
  status: 'active' | 'paused' | 'cancelled',
  token: string
): Promise<MemberBilling> => {
  const { data } = await memberBillingApi.put<MemberBilling>(`/${gymId}/${billingId}`, { status }, authHeaders(token));
  return data;
};

export const getPaymentRecords = async (gymId: string, token: string, memberId?: string, status?: string): Promise<PaymentRecord[]> => {
  const { data } = await memberBillingApi.get<PaymentRecord[]>('/payments', {
    params: {
      gym_id: gymId,
      ...(memberId ? { member_id: memberId } : {}),
      ...(status ? { status } : {}),
    },
    ...authHeaders(token),
  });
  return data;
};

export const updatePaymentStatus = async (
  gymId: string,
  recordId: string,
  payload: { status: 'paid' | 'unpaid' | 'overdue'; notes?: string },
  token: string
): Promise<PaymentRecord> => {
  const { data } = await memberBillingApi.put<PaymentRecord>(`/payments/${gymId}/${recordId}`, payload, authHeaders(token));
  return data;
};
