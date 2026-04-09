import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRef } from 'react';
import { useMessagingContext } from "@/context/message";
import { useRouter } from "next/navigation";
import {
  useSignIn as useClerkSignIn,
  useSignUp as useClerkSignUp,
  useClerk,
  useUser,
} from '@clerk/nextjs';
import { useChangeGym } from '@/hook/gym';
import { profileApi } from '@/hook/base-apis';
import { useCreateRequest } from './request';
import { useAuth } from '@clerk/nextjs';

interface SignupData {
  username: string;
  password: string;
  firstName: string;
  lastName: string;
  phone: string;
  gymId?: string;
}

export const useSignup = () => {
  const clerkSignUp = useClerkSignUp() as any;
  const signUpRef = useRef(clerkSignUp);
  signUpRef.current = clerkSignUp; // always fresh reference on every render
  const router = useRouter();
  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['user'],
    mutationFn: async (formData: SignupData) => {
      const { signUp, isLoaded } = signUpRef.current;
      if (!isLoaded || !signUp) throw new Error('Clerk is not ready yet');

      await signUp.create({
        emailAddress: formData.username,
        password: formData.password,
        firstName: formData.firstName,
        lastName: formData.lastName,
      });
      await signUp.prepareEmailAddressVerification({ strategy: 'email_code' });
      return { email: formData.username, gymId: formData.gymId };
    },
    onSuccess: ({ email, gymId }) => {
      router.push(
        `/auth/confirm?email=${encodeURIComponent(email)}${gymId ? `&gymId=${gymId}` : ''}`
      );
    },
    onError: (error: Error) => {
      setMessage(error?.message || 'Sign up failed. Please try again.');
      setColor('danger');
      setShow(true);
    },
  });
};

export const useSignOut = () => {
  const { signOut } = useClerk();
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationKey: ['user'],
    mutationFn: async () => {
      await signOut({ redirectUrl: '/' });
    },
    onSuccess() {
      queryClient.setQueriesData({ queryKey: ['user'] }, null);
    },
    onError(error) {
      console.error('ERROR: ', error);
      router.push('/');
    },
  });
};

export const useSignIn = () => {
  const { signIn, setActive } = useClerkSignIn() as any;
  const { getToken } = useAuth();
  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();
  const createRequest = useCreateRequest();
  const router = useRouter();
  const queryClient = useQueryClient();
  const setGym = useChangeGym();

  return useMutation({
    mutationKey: ['user'],
    mutationFn: async (formData: any) => {
      const si = signIn as any;
      const sa = setActive as any;
      const result = await si.create({
        identifier: formData.email,
        password: formData.password,
      });

      if (result.status !== 'complete') {
        throw new Error('Authentication failed');
      }

      await sa({ session: result.createdSessionId });

      const token = await getToken();

      const { data: profile } = await profileApi.get('', {
        params: { current_user: true },
        headers: { Authorization: `Bearer ${token}` },
      });

      if (profile?.gyms?.length > 0) {
        setGym.mutate({
          gym: profile?.gyms?.[0]?.gym,
          group: profile?.gyms?.[0]?.group,
        });
      }

      return {
        token,
        userId: result.createdSessionId,
        first_name: profile?.first_name,
        last_name: profile?.last_name,
        email: profile?.email,
        gymId: formData?.gymId,
      };
    },
    async onSuccess(data: any) {
      const { gymId } = data;
      queryClient.setQueriesData({ queryKey: ['user'] }, data);

      if (gymId) {
        createRequest.mutate({
          gym_id: gymId,
          status: 'Pending',
          first_name: data?.first_name,
          last_name: data?.last_name,
          requestor_email: data?.email,
          isFromQRCode: true,
        });
        setMessage('Successfully joined your new gym');
        setColor('success');
        setShow(true);
      } else {
        setMessage('Successful sign in');
        setColor('success');
        setShow(true);
        router.push('/student/my-gym');
      }
    },
    onError() {
      setMessage("Username and Password Don't Match");
      setColor('danger');
      setShow(true);
    },
  });
};

interface ChangePasswordInput {
  oldPassword: string;
  newPassword: string;
}

export const useChangePassword = () => {
  const { user } = useUser();
  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['user'],
    mutationFn: async ({ oldPassword, newPassword }: ChangePasswordInput) => {
      await user!.updatePassword({ currentPassword: oldPassword, newPassword });
    },
    onSuccess: () => {
      setColor('success');
      setMessage('Password updated successfully');
      setShow(true);
    },
    onError: () => {
      setColor('danger');
      setMessage('Password update failed');
      setShow(true);
    },
  });
};

interface ConfirmCodeData {
  email: string;
  code: string;
  gymId?: string;
}

export const useConfirmCode = () => {
  const clerkSignUp = useClerkSignUp() as any;
  const signUpRef = useRef(clerkSignUp);
  signUpRef.current = clerkSignUp;
  const router = useRouter();
  const {
    setShow,
    setColor,
    setMessage,
  } = useMessagingContext();

  return useMutation({
    mutationKey: ['user'],
    mutationFn: async (data: ConfirmCodeData) => {
      const { signUp, setActive } = signUpRef.current;
      if (!signUp) throw new Error('Clerk is not ready yet');

      const result = await signUp.attemptEmailAddressVerification({ code: data.code });

      if (result.status !== 'complete') {
        throw new Error('Invalid or expired code. Please try again.');
      }

      await setActive({ session: result.createdSessionId });

      return { gymId: data.gymId };
    },
    onSuccess: ({ gymId }) => {
      if (gymId) {
        router.push(`/student/my-gym?gymId=${gymId}`);
      } else {
        router.push('/auth/account-type');
      }
    },
    onError: (error: Error) => {
      setMessage(error?.message || 'Verification failed. Please try again.');
      setColor('danger');
      setShow(true);
    },
  });
};

interface ResendCodeData {
  email: string;
}

export const useResendCode = () => {
  const clerkSignUp = useClerkSignUp() as any;
  const signUpRef = useRef(clerkSignUp);
  signUpRef.current = clerkSignUp;

  return useMutation({
    mutationFn: async (_data: ResendCodeData) => {
      const { signUp } = signUpRef.current;
      if (!signUp) throw new Error('Clerk is not ready yet');
      await signUp.prepareEmailAddressVerification({ strategy: 'email_code' });
    },
    onError: (error: Error) => {
      console.error('Resend code error:', error);
    },
  });
};
