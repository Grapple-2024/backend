import gymApi from "@/util/gym-api";
import axios from "axios";
import { useToken } from "./user";
import { useMessagingContext } from "@/context/message";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Video, getSeries, seriesApi, useDeleteSeries } from "./series";
import { useLoadingContext } from "@/context/loading";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useGetGym } from "./gym";
import { useAuth } from "@clerk/nextjs";

const readFile = async (file: any): Promise<Blob> => {
  return new Blob([file], { type: file.type });
};

const handleFileDrop = async (
  file: File,
  seriesId: string,
  token: string,
) => {
  const blob = await readFile(file);
  
  const { data } = await gymApi.put(`/gym-series/${seriesId}/presign`, {}, {
    params: {
      file: file.name,
      type: 'video',
    },
    headers: {
      'Content-Type': file.type,
      Authorization: `Bearer ${token}`,
    },
  });

  await axios.put(data?.URL, blob, {
    headers: {
      'Content-Type': file.type,
    },
  });
  
  return data?.s3_object_key;
};

const handleThumbnailDrop = async (
  file: File, 
  gymId: string, 
  seriesId: string,
  token: string
) => {
  const arrayBuffer = await readFile(file);
  
  const { data } = await gymApi.put(`/gym-series/${seriesId}/presign`, {}, {
    params: {
      file: file.name,
      gym_id: gymId,
      type: 'thumbnail',
    },
    headers: {
      'Content-Type': file.type,
      Authorization: `Bearer ${token}`,
    },
  });

  await axios.put(data?.URL, arrayBuffer, {
    headers: {
      'Content-Type': file.type,
    },
  });
  
  return data?.s3_object_key;
}

export const useCreateVideo = () => {
  const gym = useGetGym();
  const queryClient = useQueryClient();
  const token = useToken();
  const { setLoading } = useLoadingContext();
  const deleteSeries = useDeleteSeries();
  const [id, setId] = useState<any>(null);
  const router = useRouter();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['series', gym?.data?.id],
    mutationFn: async ({ file, thumbnail, video, seriesId }: any) => {
      setLoading(true);
      setId(seriesId);
      
      const name = await handleFileDrop(file, seriesId, token);
      const thumbnailName = await handleThumbnailDrop(thumbnail.file, gym?.data?.id as string, seriesId, token);

      delete video.url;
      delete video.series_id;
      await gymApi.put(`/gym-series/${seriesId}/videos`,  
        { 
          ...video, 
          thumbnail_s3_object_key: thumbnailName,
          s3_object_key: name, 
          sort_order: 1,
        }, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      
      const series = await getSeries({
        gym_id: gym?.data?.id,
        ascending: false,
      }, token);

      return {
        ...series,
        id: seriesId,
      }
    },
    onSuccess: (data: any) => {
      setShow(true);
      setMessage('Video created successfully');
      setColor('success');
      
      setLoading(false);
      router.push(`/coach/content/${data.id}`);
      queryClient.setQueryData(['series', gym?.data?.id], data);
    },
    onError: (e: any) => {
      console.error('Error creating video: ', e);
      setShow(true);
      setMessage('Error creating video: ', e);
      deleteSeries.mutate(id);
      setColor('danger');
      setLoading(false);
    },
  });
};


export const useAddVideo = () => {
  const gym = useGetGym();
  const queryClient = useQueryClient();
  const token = useToken();
  const { setLoading } = useLoadingContext();
  const { userId } = useAuth();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  const gymId = gym?.data?.id;
  const profileId = userId;

  return useMutation({
    mutationKey: ['series-view', gymId, profileId],
    mutationFn: async ({ file, thumbnail, video, seriesId }: any) => {
      setLoading(true);
      const name = await handleFileDrop(file, seriesId, token);
      const thumbnailName = await handleThumbnailDrop(thumbnail.file, gym?.data?.id as string, seriesId, token);

      delete video.url;
      
      await gymApi.put(`/gym-series/${seriesId}/videos`,  
        { 
          ...video, 
          s3_object_key: name, 
          thumbnail_s3_object_key: thumbnailName,
          sort_order: 1,
        }, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      const { data } = await seriesApi.get(
        `/${seriesId}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return data;
    },
    onSuccess: (data: any) => {
      setShow(true);
      setMessage('Video created successfully');
      setColor('success');
      
      setLoading(false);
      queryClient.setQueriesData({ queryKey: ['series-view', gymId, profileId]}, data);
    },
    onError: (e) => {
      setShow(true);
      console.error("ERROR", e);
      setMessage('Error creating video');
      setColor('danger');
      setLoading(false);
    },
  });
};

export const useDeleteVideo = () => {
  const gym = useGetGym();
  const queryClient = useQueryClient();
  const token = useToken();
  const { userId } = useAuth();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  const gymId = gym?.data?.id;
  const profileId = userId;

  return useMutation({
    mutationKey: ['series-view', gymId, profileId],
    mutationFn: async ({ id, seriesId }: any) => {
      await gymApi.delete(`/gym-series/${seriesId}/videos/${id}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      const { data } = await seriesApi.get(
        `/${seriesId}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return data;
    },
    onSuccess: (data: any) => {
      setShow(true);
      setMessage('Video deleted successfully');
      setColor('success');
      
      queryClient.setQueriesData({ queryKey: ['series-view', gymId, profileId]}, data);
    },
    onError: () => {
      setShow(true);
      setMessage('Error deleting video');
      setColor('danger');
    },
  });
};

export const useUpdateVideo = () => {
  const gym = useGetGym();
  const queryClient = useQueryClient();
  const token = useToken();
  const {
    setShow,
    setMessage,
    setColor,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['single-series', token],
    mutationFn: async (video: Video) => {
      const { 
        series_id,
        ...rest
      } = video;
      
      await gymApi.put(`/gym-series/${series_id}/videos`, {
        ...rest,
        gym_id: gym?.data?.id,
      }, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      const { data } = await seriesApi.get(
        `/${series_id}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      
      return data;
    },
    onSuccess: (data: any) => {
      setShow(true);
      setMessage('Video updated successfully');
      setColor('success');
      queryClient.setQueryData(['single-series', token], data);
    },
    onError: () => {
      setShow(true);
      setMessage('Error updating video');
      setColor('danger');
    },
  });
};