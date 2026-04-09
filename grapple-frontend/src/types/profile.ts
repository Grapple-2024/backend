import { Gym } from "./gym";

export interface ProfileGym {
  gym_id: string;
  email: string;
  group: string;
  email_preferences: {
    notify_on_announcements: boolean;
    notify_on_requests: boolean;
  };
  gym: Gym;
}

export interface Profile {
  id: string;
  cognito_id: string;
  email: string;
  first_name: string;
  last_name: string;
  phone_number: string;
  notify_on_request_accepted: boolean;
  avatar_url?: string;
  gyms?: ProfileGym[];
  created_at: Date;
  updated_at: Date;
}