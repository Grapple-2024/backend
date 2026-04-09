import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMessagingContext } from "@/context/message";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import {
  getMemberBilling,
  assignPlan,
  updateBillingStatus,
  getPaymentRecords,
  updatePaymentStatus,
} from "@/api-requests/memberBilling";

export const useGetMemberBilling = (memberId?: string) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['member-billing', userId, gymId, memberId],
    queryFn: () => getMemberBilling(gymId as string, token as string, memberId),
    enabled: !!token && !!userId && !!gymId,
  });
};

export const useAssignPlan = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: (payload: { member_id: string; plan_id: string; member_name: string; start_date?: string }) =>
      assignPlan({ ...payload, gym_id: gymId as string }, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['member-billing', userId, gymId] });
      queryClient.invalidateQueries({ queryKey: ['payment-records', userId, gymId] });
      setMessage('Plan assigned');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to assign plan');
      setColor('danger');
      setShow(true);
    },
  });
};

export const useUpdateBillingStatus = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: ({ billingId, status }: { billingId: string; status: 'active' | 'paused' | 'cancelled' }) =>
      updateBillingStatus(gymId as string, billingId, status, token as string),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['member-billing', userId, gymId] });
      setMessage('Billing updated');
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to update billing');
      setColor('danger');
      setShow(true);
    },
  });
};

export const useGetPaymentRecords = (memberId?: string, status?: string) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['payment-records', userId, gymId, memberId, status],
    queryFn: () => getPaymentRecords(gymId as string, token as string, memberId, status),
    enabled: !!token && !!userId && !!gymId,
  });
};

export const useUpdatePaymentStatus = () => {
  const queryClient = useQueryClient();
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const { setShow, setColor, setMessage } = useMessagingContext();
  const gymId = gym?.data?.id;

  return useMutation({
    mutationFn: ({ recordId, status, notes }: { recordId: string; status: 'paid' | 'unpaid' | 'overdue'; notes?: string }) =>
      updatePaymentStatus(gymId as string, recordId, { status, notes }, token as string),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['payment-records', userId, gymId] });
      const label = vars.status === 'paid' ? 'Marked as paid' : vars.status === 'overdue' ? 'Marked as overdue' : 'Marked as unpaid';
      setMessage(label);
      setColor('success');
      setShow(true);
    },
    onError: (error: any) => {
      setMessage(error.message || 'Failed to update payment');
      setColor('danger');
      setShow(true);
    },
  });
};
