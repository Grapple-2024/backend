import { membersApi } from "@/hook/base-apis";

export interface MemberBillingSummary {
  status: string;
  plan_name: string;
  next_payment_date: string;
}

export interface MemberCurrentBelt {
  system: 'adult' | 'kids';
  belt: string;
  stripes: number;
}

export interface MemberProfile {
  avatar_url?: string;
  phone_number?: string;
  // gyms preserved for hooks that still reference it (gym.ts, auth.ts)
  gyms?: any[];
}

export interface RichMember {
  id: string;
  gym_id: string;
  requestor_id: string;
  requestor_email: string;
  first_name: string;
  last_name: string;
  membership_type: 'IN-PERSON' | 'VIRTUAL';
  role: 'Coach' | 'Owner' | 'Student' | string;
  status: string;
  created_at?: string;
  // joined fields
  profile?: MemberProfile;
  billing?: MemberBillingSummary | null;
  current_belt?: MemberCurrentBelt | null;
  last_check_in?: string | null;
  // kept for backward compat with components that read .approved
  approved?: boolean;
  phone?: string;
}

export interface MembersResponse {
  data: RichMember[];
  count: number;
  total_count: number;
  next_page: string | null;
  previous_page: string | null;
}

export interface MembersFilters {
  status?: string;
  role?: string;
  membership_type?: string;
  search?: string;
  page?: number;
  page_size?: number;
  sort_column?: string;
  sort_direction?: string;
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getMembersRoster = async (
  gymId: string,
  token: string,
  filters: MembersFilters = {}
): Promise<MembersResponse> => {
  const params: Record<string, string> = { gym_id: gymId };
  if (filters.status)         params.status         = filters.status;
  if (filters.role)           params.role           = filters.role;
  if (filters.membership_type) params.membership_type = filters.membership_type;
  if (filters.search)         params.search         = filters.search;
  if (filters.page)           params.page           = String(filters.page);
  if (filters.page_size)      params.page_size      = String(filters.page_size);
  if (filters.sort_column)    params.sort_column    = filters.sort_column;
  if (filters.sort_direction) params.sort_direction = filters.sort_direction;

  const { data } = await membersApi.get<MembersResponse>('', {
    params,
    ...authHeaders(token),
  });
  return data;
};
