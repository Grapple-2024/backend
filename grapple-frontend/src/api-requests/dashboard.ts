import { dashboardApi } from "@/hook/base-apis";

export interface MonthRevenue {
  month: string;    // "2025-04"
  revenue: number;  // cents
}

export interface WeekAttendance {
  week: string;   // "2025-W14"
  count: number;
}

export interface OverdueMember {
  member_id: string;
  member_name: string;
  amount: number;
  due_date: string;
}

export interface UpcomingRenewal {
  member_id: string;
  member_name: string;
  plan_name: string;
  amount: number;
  due_date: string;
}

export interface DashboardData {
  active_members: number;
  monthly_revenue: number;
  today_attendance: number;
  overdue_count: number;
  revenue_by_month: MonthRevenue[];
  attendance_by_week: WeekAttendance[];
  overdue_list: OverdueMember[];
  pending_requests: number;
  upcoming_renewals: UpcomingRenewal[];
}

const authHeaders = (token: string) => ({
  headers: { Authorization: `Bearer ${token}` },
});

export const getDashboard = async (gymId: string, token: string): Promise<DashboardData> => {
  const { data } = await dashboardApi.get<DashboardData>('', {
    params: { gym_id: gymId },
    ...authHeaders(token),
  });
  return data;
};
