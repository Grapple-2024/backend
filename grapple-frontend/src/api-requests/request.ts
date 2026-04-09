import { profileApi, requestApi } from "@/hook/base-apis";
import gymApi from "@/util/gym-api";
// utils/auth-session.ts
import { unstable_cache } from 'next/cache';


export const approveRequest = async (id: string, token: string) => {
  return await requestApi.put<any>(`/${id}`, 
  {
    status: 'Accepted'
  },
  {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
};

export const denyRequest = async (pk: any, token: string) => {
  return await requestApi.put<any>(`/${pk}`,
  {
    status: 'Denied',
  },
  {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
};

export const getRequests = async (id: string, token: string, status: any = null) => {
  const { data } = await requestApi.get<any>(``, 
  {
    params: {
      ...(status ? { status }: {}),
      gym_id: id,
    },
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  return data;
};

export interface RequestFilterFields {
  status?: string;
  membership_type?: string;
  search?: string;
  role?: string;
};

export interface RequestSortFields {
  sort_direction?: string;
  sort_column?: string;
  page?: number;
  page_size?: number;
};

export const filterRequests = async (
  id: string, 
  filters: RequestFilterFields, 
  sort: RequestSortFields, 
  token: string
) => { 
  const { data } = await requestApi.get<any>(``, 
    {
      params: {
        ...(filters?.status ? { status: filters?.status }: {}),
        ...(filters?.membership_type ? { membership_type: filters?.membership_type }: {}),
        ...(filters?.search ? { search: filters?.search }: {}),
        ...(filters?.role ? { role: filters?.role }: {}),
        gym_id: id,
        ...(sort?.sort_direction ? { sort_direction: sort?.sort_direction }: {}),
        ...(sort?.sort_column ? { sort_column: sort?.sort_column }: {}),
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  
    return data;
};

export const createRequest = async (input: any, token: string) => {
  return await requestApi.post<any>('', 
  {
    ...input,
    membership_type: "IN-PERSON"
  },
  {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
};

export const getAuthSession = unstable_cache(
  async () => {
    try {
      

      return null;
      // return auth?.tokens?.idToken?.toString();
    } catch (error) {
      console.error('Auth session error:', error);
      return null;
    }
  },
  ['auth-session'],
  {
    revalidate: 3600,
    tags: ['auth-session']
  }
);

export const getStudentRequests = async (token: string, id: string, gymId?: string, status?: string) => {
  const { data: { data } } = await requestApi.get<any>(`/`, 
  {
    params: {
      requestor: id,
      ...(status ? { status: status } : {}),
      ...(gymId ? { gym_id: gymId } : {}),
    },
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  
  return data;
};