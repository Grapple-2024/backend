import gymApi from "@/util/gym-api";
import { AxiosError } from "axios";

export const getSeries = async (id: string, token: string) => {
  try {
    const { data: { data } } = await gymApi.get<any>(`/gym-series`, {
      params: {
        gym_id: id,
        page_size: 6,
        page: 1
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    
    return data;
  } catch (error: any) {
    console.error(error);
  }
};