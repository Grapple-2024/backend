import { announcementsApi } from "@/hook/base-apis";

export const getAnnouncements = async (
  pk: string, 
  show_by_week: any, 
  token: string
) => {
  if (!pk) {
    return null;
  }

  const { data } = await announcementsApi.get<any>('', {
    params: {
      gym_id: pk, 
      ascending: false,
      show_by_week,
      page: 1,
      limit: 100,
    },
    headers: {
      Authorization: `Bearer ${token}`,
    }
  })
  return data;
}

export const createAnnouncement = async (input: any, token: string) => {
  if (input?.title === '' || input?.content === '') {
    throw new Error('Title and content are required');
  }
  
  return await announcementsApi.post<any>('', input, {
    headers: {
      Authorization: `Bearer ${token}`,
    }
  });
};

export const deleteAnnouncement = async (pk: string, token: string) => {
  return await announcementsApi.delete<any>(`/${pk}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    }
  });
}
