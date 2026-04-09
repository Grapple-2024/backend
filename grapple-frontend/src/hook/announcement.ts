import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { formatDateToUTCString } from "@/util/format-date";
import { useToken } from "./user";
import { useGetGym } from "./gym";
import { useAuth } from "@clerk/nextjs";
import { createAnnouncement, deleteAnnouncement, getAnnouncements } from "@/api-requests/announcements";

export const useCreateAnnouncement = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const gym = useGetGym();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();


  const gymData = gym?.data;

  return useMutation({
    mutationKey: ['announcements', userId, gymData?.id],
    mutationFn: async (input: any) => {
      const { data: announcement } = await createAnnouncement(input, token as string);
      return announcement;  // Just return the new announcement
    },
    onSuccess: (newAnnouncement: any) => {
      queryClient.setQueryData(['announcements', userId, gymData?.id], (oldData: any) => {
        const currentAnnouncements = oldData || [];
        const updatedAnnouncements = [...currentAnnouncements, newAnnouncement];
        
        // Sort by created_at date, most recent first
        return updatedAnnouncements.sort((a, b) => {
          return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        });
      });

      setMessage('Announcement created successfully');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(`Error creating announcement: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
  });
}

export const useDeleteAnnouncement = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const gym = useGetGym();
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();

  const gymData = gym?.data;

  
  return useMutation({
    mutationKey: ['announcements', userId, gymData?.id],
    mutationFn: (id: string) => deleteAnnouncement(id, token as string),
    onSuccess: (_, deletedId) => {
      queryClient.setQueryData(['announcements', userId, gymData?.id], (oldData: any) => {
        // Filter out the deleted announcement
        const updatedAnnouncements = (oldData || []).filter((announcement: any) => 
          announcement.id !== deletedId
        );
        
        // Sort by created_at date, most recent first
        return updatedAnnouncements.sort((a: any, b: any) => 
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
      });

      setMessage('Announcement deleted successfully');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(`Error deleting announcement: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
  });
}
export const useGetAnnouncements = (initialData: any, daySelected: Date = new Date()) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const initialDate = formatDateToUTCString(daySelected);
  
  const gymData = gym?.data;

  
  const announcements = useQuery({
    queryKey: ['announcements', userId, gymData?.id],
    queryFn: async () => {
      const { data } = await getAnnouncements(
        gym?.data?.id as string, 
        initialDate, 
        token as string
      );

      return data;
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    // initialData,
    enabled: !gym?.isPending && !!token && !!userId,
  });

  return announcements;
};

export const useUpdateAnnouncements = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymData = gym?.data;
  const {
    setMessage,
    setColor,
    setShow
  } = useMessagingContext();


  
  return useMutation({
    mutationKey: ['announcements', userId, gymData?.id],
    mutationFn: async ({ date }: any) => {
      const { data } = await getAnnouncements(gymData?.id as string, date, token as string);
      return data;
    },
    onSuccess: (data: any) => {
      queryClient.setQueriesData({ queryKey: ['announcements', userId, gymData?.id] }, data);
    },
    onError: (error: any) => {
      setMessage(`Error Fetching announcements: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
  });
};