import axios from "axios";

const host = process.env.NEXT_PUBLIC_API_HOST;

export const profileApi = axios.create({
  baseURL: `${host}/profiles`,
});

export const gymApi = axios.create({
  baseURL: `${host}/gyms`,
});

export const announcementsApi = axios.create({
  baseURL: `${host}/announcements`,
});

export const requestApi = axios.create({
  baseURL: `${host}/gym-requests`,
});

export const seriesApi = axios.create({
  baseURL: `${host}/gym-series`,
});

export const techniquesApi = axios.create({
  baseURL: `${host}/techniques`,
});

export const searchApi = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_HOST}/search`,
});

export const membershipPlansApi = axios.create({
  baseURL: `${host}/membership-plans`,
});

export const memberBillingApi = axios.create({
  baseURL: `${host}/member-billing`,
});

export const attendanceApi = axios.create({
  baseURL: `${host}/attendance`,
});

export const promotionsApi = axios.create({
  baseURL: `${host}/promotions`,
});

export const dashboardApi = axios.create({
  baseURL: `${host}/dashboard`,
});

export const membersApi = axios.create({
  baseURL: `${host}/members`,
});

const handle401 = (apiName: string) => ({
  response: (response: any) => response,
  error: async (error: any) => {
    console.error(`ERROR IN ${apiName}`, error);
    return Promise.reject(error);
  },
});

profileApi.interceptors.response.use(handle401('PROFILE API').response, handle401('PROFILE API').error);
gymApi.interceptors.response.use(handle401('GYM API').response, handle401('GYM API').error);
announcementsApi.interceptors.response.use(handle401('ANNOUNCEMENTS API').response, handle401('ANNOUNCEMENTS API').error);
requestApi.interceptors.response.use(handle401('REQUEST API').response, handle401('REQUEST API').error);
seriesApi.interceptors.response.use(handle401('SERIES API').response, handle401('SERIES API').error);
techniquesApi.interceptors.response.use(handle401('TECHNIQUES API').response, handle401('TECHNIQUES API').error);
