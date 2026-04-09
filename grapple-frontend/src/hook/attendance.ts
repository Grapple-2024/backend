import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import { getCheckIns, checkIn, deleteCheckIn } from "@/api-requests/attendance";

export const useGetCheckIns = (params: { date?: string; member_id?: string } = {}) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['check-ins', userId, gymId, params.date, params.member_id],
    queryFn: () => getCheckIns(gymId as string, token as string, params),
    enabled: !!token && !!userId && !!gymId,
  });
};

export const useCheckIn = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (payload: { member_id: string; member_name: string; avatar_url?: string; method: 'manual' | 'qr'; notes?: string }) =>
      checkIn({ ...payload, gym_id: gymId as string }, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['check-ins', userId, gymId] });
      setMessage('Checked in');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      const msg = error?.response?.data?.error?.[0] ?? error.message ?? 'Failed to check in';
      setMessage(msg);
      setColor('danger');
      setShow(true);
    },
  });
};

export const useDeleteCheckIn = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (checkInId: string) => deleteCheckIn(gymId as string, checkInId, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['check-ins', userId, gymId] });
      setMessage('Check-in removed');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to remove check-in');
      setColor('danger');
      setShow(true);
    },
  });
};
