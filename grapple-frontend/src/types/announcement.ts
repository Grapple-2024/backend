export interface Announcement {
  id: string;
  gym_id: string;
  coach_name: string;
  coach_avatar: string;
  title: string;
  description: string;
  created_at_week: number;
  created_at_year: number;
  updated_at: Date;
  created_at: Date;
}