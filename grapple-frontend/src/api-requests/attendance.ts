import { attendanceApi } from "@/hook/base-apis";

export interface CheckIn {
  id?: string;
  gym_id: string;
  member_id: string;
  member_name: string;
  avatar_url?: string;
  checked_in_at: string;
  method: 'manual' | 'qr';
  notes?: string;
  created_at?: string;
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getCheckIns = async (
  gymId: string,
  token: string,
  params: { date?: string; member_id?: string } = {}
): Promise<CheckIn[]> => {
  const { data } = await attendanceApi.get<CheckIn[]>('', {
    params: { gym_id: gymId, ...params },
    ...authHeaders(token),
  });
  return data;
};

export const checkIn = async (
  payload: { gym_id: string; member_id: string; member_name: string; avatar_url?: string; method: 'manual' | 'qr'; notes?: string },
  token: string
): Promise<CheckIn> => {
  const { data } = await attendanceApi.post<CheckIn>('', payload, authHeaders(token));
  return data;
};

export const deleteCheckIn = async (gymId: string, checkInId: string, token: string): Promise<void> => {
  await attendanceApi.delete(`/${gymId}/${checkInId}`, authHeaders(token));
};
