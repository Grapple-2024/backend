export interface ScheduleItem {
  title: string;
  start: string;
  end: string;
};

export interface GymSchedule {
  mon: ScheduleItem[];
  tue: ScheduleItem[];
  wed: ScheduleItem[];
  thu: ScheduleItem[];
  fri: ScheduleItem[];
  sat: ScheduleItem[];
  sun: ScheduleItem[];
  [key: string]: ScheduleItem[];  
};

export interface Gym {
  _id?: string;
  id?: string;
  slug: string;
  name: string;
  description: string;
  address_line_1: string;
  address_line_2?: string;
  city: string;
  state: string;
  zip: string;
  country: string;
  longitude: string;
  latitude: string;
  coach_first_name: string;
  coach_last_name: string;
  disciplines: string[];
  schedule: GymSchedule;
  public_email?: string;
  logo_url?: string;
  hero_url?: string;
  banner_url?: string;
  createdAt: Date;
  updatedAt: Date;
}