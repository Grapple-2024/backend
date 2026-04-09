import { adminApi } from '@/hook/base-apis';

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export interface MonthMRR {
  month: string;
  mrr: number;
}

export interface FeatureAdoption {
  billing_pct: number;
  attendance_pct: number;
  belt_tracking_pct: number;
}

export interface AdminMetrics {
  total_mrr: number;
  mrr_by_month: MonthMRR[];
  active_gyms: number;
  new_gyms_this_month: number;
  total_students: number;
  churn_rate: number;
  avg_students_per_gym: number;
  feature_adoption: FeatureAdoption;
}

export interface AdminGym {
  id: string;
  name: string;
  owner_name: string;
  owner_email: string;
  address: string;
  state: string;
  student_count: number;
  tier: number;
  has_billing: boolean;
  last_activity: string | null;
  created_at: string;
}

export interface AdminGymRosterResponse {
  data: AdminGym[];
  count: number;
  total_count: number;
}

export interface AdminLog {
  id: string;
  action: string;
  target_name: string;
  timestamp: string;
  metadata: Record<string, any>;
}

export interface AdminGymDetail {
  gym: Record<string, any>;
  admin_notes: AdminLog[];
  activity_log: AdminLog[];
  member_count: number;
  revenue_30d: number;
}

export const getAdminMetrics = async (token: string): Promise<AdminMetrics> => {
  const { data } = await adminApi.get<AdminMetrics>('', authHeaders(token));
  return data;
};

export const getAdminGyms = async (
  token: string,
  search = '',
  page = 1,
  pageSize = 25,
): Promise<AdminGymRosterResponse> => {
  const { data } = await adminApi.post<AdminGymRosterResponse>(
    '',
    { search, page, page_size: pageSize },
    authHeaders(token),
  );
  return data;
};

export const getAdminGymDetail = async (token: string, id: string): Promise<AdminGymDetail> => {
  const { data } = await adminApi.get<AdminGymDetail>(`/${id}`, authHeaders(token));
  return data;
};

export const adminUpdateGym = async (
  token: string,
  id: string,
  action: 'update_tier' | 'add_note',
  payload: { tier?: number; note?: string },
): Promise<void> => {
  await adminApi.put(`/${id}`, { action, ...payload }, authHeaders(token));
};

export const adminDeleteGym = async (token: string, id: string, password: string): Promise<void> => {
  await adminApi.delete(`/${id}`, {
    ...authHeaders(token),
    data: { password },
  });
};
