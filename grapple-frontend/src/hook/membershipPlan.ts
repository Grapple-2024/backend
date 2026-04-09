import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import {
  getMembershipPlans,
  createMembershipPlan,
  updateMembershipPlan,
  deleteMembershipPlan,
  MembershipPlan,
} from "@/api-requests/membershipPlan";

export const useGetMembershipPlans = () => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();

  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['membership-plans', userId, gymId],
    queryFn: () => getMembershipPlans(gymId as string, token as string),
    enabled: !!token && !!userId && !!gymId,
  });
};

export const useCreateMembershipPlan = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();

  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (plan: Omit<MembershipPlan, 'id' | 'is_active' | 'created_at' | 'updated_at'>) =>
      createMembershipPlan(plan, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['membership-plans', userId, gymId] });
      setMessage('Plan created');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to create plan');
      setColor('danger');
      setShow(true);
    },
  });
};

export const useUpdateMembershipPlan = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();

  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: ({ planId, updates }: { planId: string; updates: Partial<MembershipPlan> }) =>
      updateMembershipPlan(gymId as string, planId, updates, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['membership-plans', userId, gymId] });
      setMessage('Plan updated');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to update plan');
      setColor('danger');
      setShow(true);
    },
  });
};

export const useDeleteMembershipPlan = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();

  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (planId: string) =>
      deleteMembershipPlan(gymId as string, planId, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['membership-plans', userId, gymId] });
      setMessage('Plan removed');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to remove plan');
      setColor('danger');
      setShow(true);
    },
  });
};
