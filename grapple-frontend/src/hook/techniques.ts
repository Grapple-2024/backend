import axios from "axios";
import { useToken } from "./user";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useGetGym } from "./gym";
import { useGetUserProfile } from "./profile";

export interface TechniqueOfTheWeek {
  id?: string;
  title: string;
  description: string;
  gym_id?: string;
  series_id?: string;
  disciplines?: string[];
  display_on_week?: Date;
  created_at_year?: Date;
  created_at?: Date;
  updated_at?: Date;
};

const techniquesApi = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_HOST}/techniques`,
});

export const createTechnique = async (technique: TechniqueOfTheWeek, token: string) => {
  const { data } = await techniquesApi.post(
    '/', 
    {
      series: {
        id: technique.series_id,
      },
      title: technique.title,
      description: technique.description,
      disciplines: technique.disciplines,
      gym_id: technique.gym_id,
      display_on_week: technique.display_on_week,
    },
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  
  return data;
};

export const getTechniques = async (date: Date, gymId: string, token: string) => {
  if (!gymId) {
    return null;
  }

  const { data: { data } } = await techniquesApi.get(
    '/', 
    {
      params: {
        page: 1,
        limit: 100,
        gym_id: gymId,
        show_by_week: date,
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  
  return data;
};

export const getTechnique = async (id: string, token: string) => {
  const { data } = await techniquesApi.get(
    `/${id}`, 
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  return data;
};

export const updateTechnique = async (technique: TechniqueOfTheWeek, token: string) => {
  const { data } = await techniquesApi.put(
    `/${technique.id}`, 
    technique,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  return data;
};

export const deleteTechnique = async (id: string, token: string) => {
  const { data } = await techniquesApi.delete(
    `/${id}`, 
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  return data;
};

export const useDeleteTechnique = () => {
  const queryClient = useQueryClient();
  const gym = useGetGym();
  const token = useToken();
  const {
    setMessage,
    setColor,
    setOpen,
  } = useMessagingContext();
  const profile = useGetUserProfile();

  const gymId = gym?.data?.id;
  const profileId = profile?.data?.cognito_id;

  return useMutation({
    mutationKey: ['techniques', profileId, gymId],
    mutationFn: async (id: string) => {
      await deleteTechnique(id, token);
      
      return await getTechniques(
        new Date(),
        gym?.data?.id as string,
        token,
      );
    },
    onSuccess: (data: any) => {
      // setMessage('Technique of the week sucessfully deleted');
      // setColor('success');
      // setOpen(true);
      queryClient.setQueriesData({ queryKey: ['techniques', profileId, gymId] }, data);
    },
    onError: () => {
      setMessage('Error deleting technique of the week');
      setColor('danger');
      setOpen(true);
    }
  });
}

export const useGetTechniques = () => {
  const token = useToken();
  const currentGym = useGetGym();
  const profile = useGetUserProfile();

  const gymId = currentGym?.data?.id;
  const profileId = profile?.data?.cognito_id;

  return useQuery({
    queryKey: ['techniques', profileId, gymId],
    queryFn: () => getTechniques(
      new Date(),
      gymId as string,
      token,
    ),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    enabled: !!token && !currentGym?.isPending && !profile.isPending,
  });
};

export const useUpdateTechniques = () => {
  const queryClient = useQueryClient();
  const currentGym = useGetGym();
  const token = useToken();
  const {
    setMessage,
    setColor,
    setOpen,
  } = useMessagingContext();
  const profile = useGetUserProfile();

  const gymId = currentGym?.data?.id;
  const profileId = profile?.data?.cognito_id;

  return useMutation({
    mutationKey: ['techniques', profileId, gymId],
    mutationFn: ({ date }: { date: Date }) => {
      return getTechniques(
        date,
        gymId as string,
        token,
      );
    },
    onSuccess: (data) => {
      queryClient.setQueriesData({ queryKey: ['techniques', profileId, gymId] }, data);
    },
    onError: () => {
      setMessage('Error filtering techniques of the week');
      setColor('danger');
      setOpen(true);
    }
  });
};

export const useCreateTechnique = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const currentGym = useGetGym();
  const {
    setMessage,
    setColor,
    setOpen,
  } = useMessagingContext();
  const profile = useGetUserProfile();
  const updateTechniques = useUpdateTechniques();

  const gymId = currentGym?.data?.id;
  const profileId = profile?.data?.cognito_id;
  
  return useMutation({
    mutationKey: ['techniques', profileId, gymId],
    mutationFn: async (technique: any) => {
      await createTechnique(technique, token);
      const values = await getTechniques(
        new Date(),
        gymId as string,
        token,
      );
      updateTechniques.mutate({ date: new Date() });
      return values;
    },
    onSuccess: (data: any) => {
      setMessage('Technique of the week sucessfully added');
      setColor('success');
      setOpen(true);

      queryClient.setQueriesData({ queryKey: ['techniques', profileId, gymId] }, data);
    },
    onError: () => {
      setMessage('Error adding technique of the week');
      setColor('danger');
      setOpen(true);
    }
  });
};