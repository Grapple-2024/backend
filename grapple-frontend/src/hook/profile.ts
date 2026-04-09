import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useAuth } from "@clerk/nextjs";
import { Profile } from "@/types/profile";
import { profileApi } from "./base-apis";
import axios from "axios";
import { useToken } from "./user";

export const useGetUserProfile = (initialProfile: any = null) => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId, isSignedIn } = useAuth();

  return useQuery({
    queryKey: ['profile', userId],
    queryFn: async () => {
      const { data: profile } = await profileApi.get('', {
        params: { current_user: true },
        headers: { Authorization: `Bearer ${token}` }
      });

      return profile;
    },
    initialData: initialProfile,
    enabled: isSignedIn === true && !!token && !!userId,
  });
};

export const useMutateProfile = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  
  const mutation = useMutation({
    mutationKey: ['profile', userId],
    mutationFn: async (): Promise<Profile> => {
      const { data: profile } = await profileApi.get('', {
        params: { current_user: true },
        headers: { Authorization: `Bearer ${token}` }
      });
      
      return profile;
    },
    onSuccess: (data: Profile) => {
      queryClient.setQueriesData({ queryKey: ['profile', userId] }, data);
    },
  });

  return mutation;
};

export const readFile = (file: Blob): Promise<ArrayBuffer> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.onloadend = () => {
      resolve(reader.result as ArrayBuffer);
    };

    reader.onerror = reject;

    reader.readAsArrayBuffer(file);
  });
};

export const useUpdateAvatar = () => {
  const token = useToken();
  const queryClient = useQueryClient();
  const profile = useGetUserProfile();
  const { userId } = useAuth();

  return useMutation({
    mutationKey: ['profile', userId],
    mutationFn: async ({ file, name }: {file: File, name: string}): Promise<Profile> => {
      const url = `/avatar?file=${Math.floor(Date.now() / 1000)+ name}`;
      const data = await readFile(file);
      
      const response = await profileApi.put(url, {}, {
        headers: {
          Authorization: `Bearer ${token}`,
        }
      });

      await axios.put(response?.data?.URL, data);
      
      const updatedProfile = await profileApi.put("", {
        ...profile?.data,
        avatar_url: response.data.s3_object_url,
      }, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      
      return updatedProfile.data;
    },
    onSuccess: (data: Profile) => {
      queryClient.setQueriesData({ queryKey: ['profile', userId] }, data);
    },
  })
};

const createProfile = async (profile: Profile, token: string): Promise<Profile> => {
  const { data } = await profileApi.post(
    '/', 
    profile,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  
  return data;
};  

export const useCreateProfile = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['profile', userId],
    mutationFn: async (profile: Profile): Promise<Profile> => {
      return await createProfile(profile, token);
    },
    onSuccess: (data: Profile) => {
      setShow(true);
      setMessage('Profile created successfully');
      setColor('success');
      
      queryClient.setQueriesData({ queryKey: ['profile', userId] }, data);
    },
    onError: () => {
      setShow(true);
      setMessage('Error creating profile');
      setColor('danger');
    },
  });
}

export const useUpdateProfile = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['profile', userId],
    mutationFn: async (profile: Profile): Promise<Profile> => {
      const response = await profileApi.put(
        ``,
        profile,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return response.data
    },
    onSuccess: (data: Profile) => {
      setShow(true);
      setMessage('Profile updated successfully');
      setColor('success');
      
      queryClient.setQueriesData({queryKey: ['profile', userId]}, data);
    },
    onError: () => {
      setShow(true);
      setMessage('Error updating profile');
      setColor('danger');
    },
  });
}