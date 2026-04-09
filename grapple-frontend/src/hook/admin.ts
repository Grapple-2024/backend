'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@clerk/nextjs';
import { useToken } from './user';
import {
  getAdminMetrics,
  getAdminGyms,
  getAdminGymDetail,
  adminUpdateGym,
  adminDeleteGym,
} from '@/api-requests/admin';

export const useAdminMetrics = () => {
  const token = useToken();
  const { userId } = useAuth();
  return useQuery({
    queryKey: ['admin', 'metrics'],
    queryFn: () => getAdminMetrics(token),
    enabled: !!token && !!userId,
    staleTime: 60_000,
  });
};

export const useAdminGyms = (search: string, page: number, pageSize = 25) => {
  const token = useToken();
  const { userId } = useAuth();
  return useQuery({
    queryKey: ['admin', 'gyms', search, page, pageSize],
    queryFn: () => getAdminGyms(token, search, page, pageSize),
    enabled: !!token && !!userId,
    staleTime: 30_000,
  });
};

export const useAdminGymDetail = (id: string | null) => {
  const token = useToken();
  const { userId } = useAuth();
  return useQuery({
    queryKey: ['admin', 'gym', id],
    queryFn: () => getAdminGymDetail(token, id!),
    enabled: !!token && !!userId && !!id,
  });
};

export const useAdminUpdateGym = () => {
  const token = useToken();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action, payload }: {
      id: string;
      action: 'update_tier' | 'add_note';
      payload: { tier?: number; note?: string };
    }) => adminUpdateGym(token, id, action, payload),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['admin', 'gyms'] });
      qc.invalidateQueries({ queryKey: ['admin', 'gym', vars.id] });
    },
  });
};

export const useAdminDeleteGym = () => {
  const token = useToken();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      adminDeleteGym(token, id, password),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'gyms'] });
      qc.invalidateQueries({ queryKey: ['admin', 'metrics'] });
    },
  });
};
