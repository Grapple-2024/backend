/* eslint-disable */
'use client';

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { readFile, useGetUserProfile, useMutateProfile } from "./profile";
import axios from "axios";
import { useAuth } from "@clerk/nextjs";
import { usePathname, useRouter } from "next/navigation";
import { gymApi } from "./base-apis";
import { useToken } from "./user";
import { useUserContext } from "@/context/user";
import { Gym, GymSchedule } from "@/types/gym";

export const getGymById = async (gymId: string, token: string = '') => {
  if (!gymId ) {
    return null;
  }

  const { data } = await gymApi.get<any>(`/${gymId}`, {
    ...(
      token === '' ? {} : {
        headers: {
          Authorization: `Bearer ${token}`,
        }
      }
    )
  });
  
  return data;
}

export const getBannerImage = async (gym: any, idToken: string) => {
  const { data } = await gymApi.get<any>(`/s3-presign-url`, {
    params: {
      gym: gym?.pk,
      operation: 'download',
      key: gym?.banner_image,
    },
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });
  
  return data[0];
};

export const getAllGyms = async () => {
  const { data: data } = await gymApi.get<any>('/gyms');
  
  return data;
}

const getRole = (group: string) => {
  if (group.includes('owners')) {
    return 'Owner';
  }

  if (group.includes('coaches')) {
    return 'Coach';
  }

  return 'Student';
};

export const useGetAllAvailableGyms = (profile: any = null) => {
  // Transform the data directly
  const gyms = profile?.gyms?.map(({ gym, group }: any) => ({
    ...gym,
    role: getRole(group),
  })) || [];

  // Return in same format as useQuery for consistency
  return { ...profile, data: gyms };
};

export const useCreateGym = () => {
  const updateProfile = useMutateProfile();
  const idToken = useToken();
  const router = useRouter();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();
  const setCurrentGym = useChangeGym();
  const { userId } = useAuth();
  const queryClient = useQueryClient();
  
  const mutation = useMutation({
    mutationKey: ['gyms', userId],
    mutationFn: async (gym: Gym) => {
      if (
        gym.name === "" ||
        gym.address_line_1 === "" ||
        gym.city === "" ||
        gym.state === "" ||
        gym.zip === ""
      ) {
        throw new Error('All fields are required');
      }

      const newGym = await gymApi.post<Gym>('/', { 
        ...gym, 
        creator: userId,
      }, {
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
      });

      const gymData = await getGymById(newGym?.data?.id as string, idToken);
      
      setCurrentGym.mutate({
        gym: gymData,
        group: 'owners',
      });

      return gymData?.data;
    },
    onSuccess: async (data: Gym) => {
      setMessage('Gym created successfully');
      setColor('success');
      setShow(true);
      

      // reset the users profile
      updateProfile.mutate();
      queryClient.setQueriesData({ queryKey: ['gyms', userId] }, data);
      router.push(`/coach/profile`);
    },
    onError: (error) => {
      setMessage(`Error creating gym: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
  });

  return mutation;
};

export const useGetGym = () => {
  const token = useToken();
  const queryClient = useQueryClient();
  const { userId, isSignedIn } = useAuth();
  const profile = useGetUserProfile();
  const { currentGym, setCurrentGym, setRole } = useUserContext();

  const gym = useQuery({
    queryKey: ['gyms', userId],
    queryFn: async (): Promise<Gym> => {
      let gym: Gym;

      if (!currentGym) {
        gym = await getGymById(profile?.data?.gyms[0]?.gym?.id as string, token);
        setCurrentGym(gym);
        setRole(getRole(profile?.data?.gyms[0]?.group));
      } else {
        gym = await getGymById(currentGym?.id as string, token);
      }

      return gym;
    },
    enabled: !!userId && isSignedIn === true && !!token && !profile.isPending,
  });

  return gym;
};

export const useChangeGym = () => {
  const queryClient = useQueryClient();
  const { userId } = useAuth();
  const router = useRouter();
  const { setCurrentGym, setRole } = useUserContext();

  return useMutation({
    mutationKey: ['gyms', userId],
    mutationFn: async ({ gym, group }: { gym: Gym; group: string; id?: any }) => {
      setCurrentGym(gym);
      setRole(getRole(group));

      return {
        gym: gym as Gym,
        group: getRole(group) as string,
      };
    },
    onSuccess: ({ gym, group }: {
      gym: Gym;
      group: string;
    }) => {
      queryClient.setQueryData(['gyms', userId], gym);
      queryClient.setQueriesData({ queryKey: ['announcements', userId, gym?.id] }, gym);

      const route = (group === 'Owner' || group === 'Coach') ? '/coach' : '/student';
      router.push(`${route}/my-gym`);
    },
    onError: (error: any) => {
      console.error("ERROR UPDATING SESSION: ", error);
    }
  });
};

export const useUpdateGym = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const mutateProfile = useMutateProfile();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();
  
  const mutation = useMutation({
    mutationKey: ['gyms', userId],
    mutationFn: async (gym: any) => {
      const gymId = gym?.id || gym?._id;
      const value = await gymApi.put(`/${gymId}`, 
      { 
        ...gym, 
        creator: userId 
      }, 
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      return {
        ...value.data,
        id: value.data?._id,
      };
    },
    onSuccess: async (data: Gym) => {
      setMessage('Gym Profile Updated Successfully');
      setColor('success');
      setShow(true);
      
      queryClient.setQueriesData({ queryKey: ['gyms', userId] }, data);
      mutateProfile.mutate();
    },
    onError: (error) => {
      setMessage(`Error updating gym profile: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
  });

  return mutation;
}

export const useUploadGymImage = () => {
  const token = useToken();
  const queryClient = useQueryClient();
  const gym = useGetGym();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();
  const { userId } = useAuth();
  const gymId = gym?.data?.id || gym?.data?._id;
  
  return useMutation({
    mutationKey: ['gyms', userId],
    mutationFn: async ({ file, fileType, name }: {file: Blob, fileType: string, name: string}) => {
      const url = `/${gymId}/presign?type=${fileType}&file=${name}`;
      const data = await readFile(file);
      
      const response = await gymApi.put(url, {}, {
        headers: {
          Authorization: `Bearer ${token}`,
        }
      });
      
      await axios.put(response?.data?.URL, data);
      
      await gymApi.put(`/${gymId}`, {
        ...gym?.data,
        [`${fileType}_url`]: response.data.s3_object_url,
      }, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      
      const newGymData = await getGymById(gymId as string);
      
      return newGymData;
    },
    onSuccess: (data: any) => {
      queryClient.setQueriesData({ queryKey: ['gyms', userId] }, data);
    },
    onError: (error) => {
      setMessage(`Error updating profile: ${error.message}`);
      setColor('danger');
      setShow(true);
    }
  })
};

// export const useSetCurrentGym = (profile: any = null) => {
//   const router = useRouter();
//   const { data: session, update } = useSession();
//   const userId = session?.userSub;

//   return useMutation({
//     mutationKey: ['gyms', userId],
//     mutationFn: async ({ gym, group }: any) => {
//       // Do a single update and await it
//       await update({
//         current_gym: gym,
//         current_role: group,
//       });
      
//       return {
//         gym, 
//         group
//       }
//     },
//     onSuccess: () => {
//       router.push('/coach/profile');
//     },
//     onError: (error: any) => {
//       console.error("ERROR UPDATING SESSION: ", error);
//     }
//   });
// };

export const useDeleteGym = () => {
  const queryClient = useQueryClient();
  const profile = useGetUserProfile();
  const token = useToken();
  const availableGyms = useGetAllAvailableGyms();
  const currentGym = useGetGym();
  const router = useRouter();
  const { userId } = useAuth();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['gyms', userId],
    mutationFn: async () => {
      await gymApi.delete(`/gyms/${currentGym?.data?.id}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      
      return true;
    },
    onSuccess: async () => {
      setMessage('Gym deleted successfully');
      setColor('success');
      setShow(true);

      if (availableGyms?.data?.length === 0) {
        router.push('/student/my-gym');
        return;
      }

      // Invalidate all related queries
      await queryClient.setQueryData(
        ['current-gym', profile.data?.id], 
        (availableGyms as any)?.data[0]
      );
      await queryClient.invalidateQueries({
        queryKey: ['user-gyms', profile.data?.id]
      });
      await queryClient.invalidateQueries({
        queryKey: ['profile', userId]
      });

      router.push('/student/my-gym');
    },
    onError: (error) => {
      console.error("ERROR: ", error);
    },
  });
};

