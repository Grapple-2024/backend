import { gymApi } from "@/hook/base-apis";


export const getCurrentGym = async (gymId: string, token: string) => {
  const { data } = await gymApi.get<any>(`/${gymId}`, {
    ...(
      token === '' ? {} : {
        headers: {
          Authorization: `Bearer ${token}`,
        }
      }
    )
  });
  
  return data;
};