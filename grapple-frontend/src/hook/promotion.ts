import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import {
  getPromotionHistory,
  getCurrentBelts,
  recordPromotion,
  deletePromotion,
  Promotion,
} from "@/api-requests/promotion";

export const useGetPromotionHistory = (memberId: string) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['promotions', userId, gymId, memberId],
    queryFn: () => getPromotionHistory(gymId as string, memberId, token as string),
    enabled: !!token && !!userId && !!gymId && !!memberId,
  });
};

export const useGetCurrentBelts = () => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['promotions-current', userId, gymId],
    queryFn: () => getCurrentBelts(gymId as string, token as string),
    enabled: !!token && !!userId && !!gymId,
  });
};

export const useRecordPromotion = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (payload: Omit<Promotion, 'id' | 'created_at'>) =>
      recordPromotion(payload, token as string),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['promotions', userId, gymId, variables.member_id] });
      queryClient.invalidateQueries({ queryKey: ['promotions-current', userId, gymId] });
      setMessage('Promotion recorded');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      const msg = error?.response?.data?.error?.[0] ?? error.message ?? 'Failed to record promotion';
      setMessage(msg);
      setColor('danger');
      setShow(true);
    },
  });
};

export const useDeletePromotion = (memberId: string) => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (promotionId: string) =>
      deletePromotion(gymId as string, promotionId, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['promotions', userId, gymId, memberId] });
      queryClient.invalidateQueries({ queryKey: ['promotions-current', userId, gymId] });
      setMessage('Promotion removed');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to remove promotion');
      setColor('danger');
      setShow(true);
    },
  });
};
