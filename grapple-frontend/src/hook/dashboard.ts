import { useQuery } from "@tanstack/react-query";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import { getDashboard } from "@/api-requests/dashboard";

export const useGetDashboard = () => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['dashboard', userId, gymId],
    queryFn: () => getDashboard(gymId as string, token as string),
    enabled: !!token && !!userId && !!gymId,
    staleTime: 60_000,
  });
};
