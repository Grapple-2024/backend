import { useQuery } from "@tanstack/react-query";
import { useToken } from "./user";
import { useAuth } from "@clerk/nextjs";
import { useGetGym } from "./gym";
import { getMembersRoster, MembersFilters } from "@/api-requests/members";

export const useGetMembersRoster = (filters: MembersFilters = {}) => {
  const token = useToken();
  const gym = useGetGym();
  const { userId } = useAuth();
  const gymId = gym?.data?.id;

  return useQuery({
    queryKey: ['members', userId, gymId, filters],
    queryFn: () => getMembersRoster(gymId as string, token as string, filters),
    enabled: !!token && !!userId && !!gymId,
  });
};
