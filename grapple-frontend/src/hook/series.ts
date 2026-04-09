import axios from 'axios';
import qs from 'qs';
import { useToken } from './user';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMessagingContext } from '@/context/message';
import { useEditSeriesContext } from '@/context/edit-series';
import { useGetGym } from './gym';
import { useGetUserProfile } from './profile';
export interface Video {
  id?: string;
  series_id?: string;
  title: string;
  description: string;
  presigned_url: string;
  difficulty: string;
  disciplines: string[];
  created_at: Date;
};

export interface VideoSeries {
  id?: string;
  gym_id?: string;
  title: string;
  description: string;
  videos?: Video[];
  disciplines: string[];
  difficulty: string[];
  created_at?: Date;
};

export interface FetchSeriesDTO {
  gym?: string;
  discipline?: string,
  difficulty?: string;
  ascending?: boolean;
  url?: any;
};


export const seriesApi = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_HOST}/gym-series`,
});

seriesApi.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      window.location.href = '/auth';
    }
    return Promise.reject(error);
  }
);

export const createSeries = async (series: VideoSeries, token: string) => {
  const { data } = await seriesApi.post(
    '/', 
    series,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  
  return data;
};



export const getSeries = async (query: any, token: string) => {
  const { data } = await seriesApi.get(
    '', 
    {
      params: {
        ...query
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
      paramsSerializer: params => {
        return qs.stringify(params, { arrayFormat: 'repeat' });
      }
    }
  );
  
  return data;
};

export const useGetSeries = (initialSeries: any) => {
  const gym = useGetGym();
  const token = useToken();
  const profile = useGetUserProfile();
  
  const profileId = profile?.data?.id;
  const gymId = gym?.data?.id || gym?.data?._id;
  
  const series = useQuery({
    queryKey: ['series', gymId, profileId],
    queryFn: () => getSeries({
      gym_id: gymId,
      page_size: 6,
    }, token),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    initialData: initialSeries,
    enabled: !!profileId && !!token && !!gymId,
  });

  return series;
};

export const useUpdateDisplaySeries = () => {
  const gym = useGetGym();
  const token = useToken();
  const profile = useGetUserProfile();
  const queryClient = useQueryClient();
  
  const profileId = profile?.data?.id;
  const gymId = gym?.data?.id;
  
  const mutation = useMutation({
    mutationKey: ['series', gymId, profileId],
    mutationFn: (event: any = null) => getSeries({
      gym_id: gymId,
      page_size: 6,
      ...(event ? event: {}),
    }, token),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: (data: any) => {
      queryClient.setQueriesData({ queryKey: ['series', gymId, profileId] }, data);
    },
  });

  return mutation;
};

export const useCreateSeries = () => {
  const gym = useGetGym();
  const { setCurrentSeries } = useEditSeriesContext();
  const token = useToken();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();
  const profile = useGetUserProfile();
  
  const profileId = profile?.data?.id;
  const gymId = gym?.data?.id;

  const mutation = useMutation({
    mutationKey: ['series', gymId, profileId],
    mutationFn: async (event: any) => {
      return await createSeries({
        gym_id: gym?.data?.id,
        ...event,
      }, token);
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: (data: any) => {
      setCurrentSeries(data);
    },
    onError: (error: any) => {
      setShow(true);
      setMessage('Error creating series');
      setColor('danger');
    }
  });

  return mutation;
};

export const useDeleteSeries = () => {
  const gym = useGetGym();
  const token = useToken();
  const profile = useGetUserProfile();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();
  const updateDisplaySeries = useUpdateDisplaySeries();
  
  const profileId = profile?.data?.id;
  const gymId = gym?.data?.id;

  const mutation = useMutation({
    mutationKey: ['series', gymId, profileId],
    mutationFn: async (pk: string) => {
      return await seriesApi.delete(
        `/${pk}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: ({ data }: any) => {
      // if (DeletedCount > 0) {
      //   setShow(true);
      //   setMessage('Series deleted successfully');
      //   setColor('success');
      // }
      updateDisplaySeries.mutate(null);
    },
    onError: (error: any) => {
      console.error("ERROR: ", error);
      setShow(true);
      setMessage('Error deleting series');
      setColor('danger');
    }
  });

  return mutation;
};

/**
 * Below is the state for the series view page
 */

export const useGetSeriesView = (seriesId: string) => {
  const gym = useGetGym();
  const token = useToken();
  const profile = useGetUserProfile();
  
  const profileId = profile?.data?.cognito_id;
  const gymId = gym?.data?.id;
  
  const series = useQuery({
    queryKey: ['series-view', gymId, profileId],
    queryFn: async () => {
      const { data } = await seriesApi.get(
        `/${seriesId}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return data;
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    // initialData: initialData,
    enabled: !!profileId && !!seriesId && !!gymId && !!token,
  });

  return series;
};

export const useUpdateSeriesView = (seriesId: string) => {
  const gym = useGetGym();
  const token = useToken();
  const profile = useGetUserProfile();
  const queryClient = useQueryClient();
  
  const profileId = profile?.data?.cognito_id;
  const gymId = gym?.data?.id;

  const mutation = useMutation({
    mutationKey: ['series-view', gymId, profileId],
    mutationFn: async () => {
      const { data } = await seriesApi.get(
        `/${seriesId}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return data;
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: (data: any) => {
      queryClient.setQueriesData({ queryKey: ['series-view', gymId, profileId] }, data);
    },
  });

  return mutation;
};

export const useUpdateSeries = (seriesId: string) => {
  const gym = useGetGym();
  const token = useToken();
  const queryClient = useQueryClient();
  const profile = useGetUserProfile();
  const updateSeriesView = useUpdateSeriesView(seriesId);
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();
  
  const profileId = profile?.data?.id;
  const gymId = gym?.data?.id;

  const mutation = useMutation({
    mutationKey: ['series-view', gymId, profileId],
    mutationFn: async (event: any) => {
      const { data } = await seriesApi.put(
        `/${event.id}`,
        event,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
    
      return data;
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: (data: any) => {
      setShow(true);
      setMessage('Series updated successfully');
      setColor('success');
      
      updateSeriesView.mutate();
    },
    onError: (error: any) => {
      setShow(true);
      setMessage('Error updating series');
      setColor('danger');
    }
  });

  return mutation;
};

export const useQueryCoachSeries = () => {  
  const token = useToken();
  const gym = useGetGym();
  const queryClient = useQueryClient();

  const query = useMutation({
    mutationKey: ['series', gym?.data?.id],
    mutationFn: (event: any) => {
      return getSeries({
        gym_id: gym?.data?.id,
        page_size: 6,
        ...event,
      }, token);
    },
    onSuccess: (data: any) => {
      queryClient.setQueryData(['series', gym?.data?.id], data);
    }
  });

  return query;
};