import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useToken } from "./user";
import { approveRequest, createRequest, denyRequest, filterRequests, getRequests, getStudentRequests } from "@/api-requests/request";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import { gymApi } from "./base-apis";
import { useGetUserProfile } from "./profile";
import { useRouter } from "next/navigation";

export const useCreateRequest = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const router = useRouter();



  const mutation = useMutation({
    mutationFn: async (event: any) => {
      const { isFromQRCode = false, ...rest } = event;
      
      const data = await createRequest({
        ...rest,
        role: "student",
      }, token as string);

      return {
        ...data,
        isFromQRCode,
      };
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    onSuccess: (values: any) => {
      const {
        data,
        isFromQRCode,
      } = values;
      
      queryClient.setQueriesData({ queryKey: ['requests', userId, data?.gym_id] }, [data]);

      if (isFromQRCode) {
        router.push("/student/my-gym");
      }
    },
  });

  return mutation;
};

export const useGetRequests = (initialData: any) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();


  const gymId = gym?.data?.id;
  
  const requestData = useQuery({
    queryKey: ['requests', userId, gymId],
    queryFn: () => getRequests(gymId as string, token as string),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    ...(initialData ? { initialData }: {}),
    enabled: !!token && !!userId && !!gymId, 
  });

  return requestData;
};

export const useFilterRequests = () => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const queryClient = useQueryClient();


  const gymId = gym?.data?.id;

  return useMutation({
    mutationKey: ['requests', userId, gymId],
    mutationFn: async ({
      filters,
      sort
    }: any) => {
      return await filterRequests(gymId as string, filters, sort, token as string);
    },
    onSuccess: (data: any) => {
      queryClient.setQueriesData({ queryKey: ['requests', userId, gymId] }, data);
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
  })
};

export const useGetNotifications = (type: any = null) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();


  const gymData = gym?.data;
  
  const requestData = useQuery({
    queryKey: ['notifications', userId, gymData?.id],
    queryFn: () => getRequests(gymData?.id as string, token, type),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    enabled: !!token && !!gymData?.id,
  });
  
  return requestData;
}

export const useUpdateUserRole = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();


  const gymId = gym?.data?.id;

  const mutation = useMutation({
    mutationKey: ['requests', userId, gymId],
    mutationFn: async ({
      username,
      role,
      cognito_id
    }: any) => {
      try {
        await gymApi.put<any>(`/${gymId}/assign-role`, {
          username,
          role,
          cognito_id,
        }, {
          headers: {
            Authorization: `Bearer ${token}`,
          }
        });

        const requests = await getRequests(gymId as string, token);

        return requests;
      } catch (error) {
        console.error(error);
      }
    },
    onSuccess: async (data: any) => {
      queryClient.setQueriesData({ queryKey: ['requests', userId, gymId] }, data);
      queryClient.invalidateQueries({ queryKey: ['members', userId, gymId] });
      setShow(true);
      setColor('success');
      setMessage('Role updated successfully');
    },
    retry: false,
  });

  return mutation;
};

export const useApproveRequest = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();

  const {
    setShow: showModal,
    setColor,
    setMessage,
  } = useMessagingContext();


  const gymId = gym?.data?.id;

  return useMutation({
    mutationKey: ['requests', userId, gymId],
    mutationFn: (id: string) => approveRequest(id, token),
    onSuccess: async () => {
      setMessage('Request approved');
      setColor('success');
      showModal(true);

      const data = await getRequests(gymId as string, token);
      const pendingRequests = await getRequests(gymId as string, token, 'Pending');
      queryClient.setQueriesData({ queryKey: ['requests', userId, gymId] }, data);
      queryClient.setQueriesData({ queryKey: ['notifications', gymId] }, pendingRequests);
      queryClient.invalidateQueries({ queryKey: ['members', userId, gymId] });
    },
    onError: (error: any) => {
      setMessage(error.message);
      setColor('danger');
      showModal(true);
    }
  });
};

export const useKickMember = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();

  const {
    setShow: showModal,
    setColor,
    setMessage,
  } = useMessagingContext();

  const gymId = gym?.data?.id;

  return useMutation({
    mutationKey: ['requests', userId, gymId],
    mutationFn: (id: string) => denyRequest(id, token),
    onSuccess: async () => {
      setMessage('Member removed');
      setColor('success');
      showModal(true);

      const data = await getRequests(gymId as string, token);
      queryClient.setQueriesData({ queryKey: ['requests', userId, gymId] }, data);
      queryClient.invalidateQueries({ queryKey: ['members', userId, gymId] });
    },
    onError: (error: any) => {
      setMessage(error.message);
      setColor('danger');
      showModal(true);
    }
  });
};

export const useDenyRequest = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();

  const {
    setShow: showModal,
    setColor,
    setMessage,
  } = useMessagingContext();


  const gymId = gym?.data?.id;

  return useMutation({
    mutationKey: ['requests', userId, gymId],
    mutationFn: (id: string) => denyRequest(id, token),
    onSuccess: async () => {
      setMessage('Request denied');
      setColor('success');
      showModal(true);

      const data = await getRequests(gymId as string, token);
      const pendingRequests = await getRequests(gymId as string, token, 'Pending');
      queryClient.setQueriesData({ queryKey: ['requests', userId, gymId] }, data);
      queryClient.setQueriesData({ queryKey: ['notifications', gymId] }, pendingRequests);
      queryClient.invalidateQueries({ queryKey: ['members', userId, gymId] });
    },
    onError: (error: any) => {
      setMessage(error.message);
      setColor('danger');
      showModal(true);
    }
  });
};

export const useGetByRequestor = (gymId: any = null) => {
  const profile = useGetUserProfile();
  const token = useToken();
  const { userId } = useAuth();

  
  const request = useQuery({
    queryKey: ['requests', userId, gymId],
    queryFn: async () => {
      return await getStudentRequests(
        token, 
        profile?.data?.cognito_id as string,
        gymId,
      )
    },
    retry(failureCount, error) {
      return failureCount < 0;
    },
    enabled: !!profile?.data,
  });

  return request;
};

export const useGetStudentsRequest = () => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();


  const gymId = gym?.data?.id;
  
  const requestData = useQuery({
    queryKey: ['requests', userId, gymId],
    queryFn: () => getStudentRequests(
      token, 
      userId as string,
      gymId,
    ),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    enabled: !!token && !!userId, 
  });

  return requestData;
};