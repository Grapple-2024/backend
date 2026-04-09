import { useMutation } from "@tanstack/react-query";
import axios from "axios";
import { useToken } from "./user";
import { useMessagingContext } from "@/context/message";

const emailApi = axios.create({
  baseURL: `${process.env.NEXT_PUBLIC_API_HOST}/emails`,
});

export const sendEmail = async (email: any) => {
  const { data } = await emailApi.post(
    '/', 
    email,
  );
  
  return data;
};

export const useSendEmail = () => {
  const {
    setMessage,
    setShow,
    setColor,
  } = useMessagingContext();

  const token = useToken();
  const mutation = useMutation({
    mutationKey: ['emails'],
    mutationFn: async (email: any) => {
      return await sendEmail(email);
    },
    onSuccess: () => {
      setMessage('Email sent successfully');
      setColor('success');
      setShow(true);
    },
    onError: (error) => {
      setMessage(`Error sending email: ${error.message}`);
      setColor('danger');
      setShow(true);
    },
  });

  return mutation;
}