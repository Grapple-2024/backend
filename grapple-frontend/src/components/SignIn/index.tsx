'use client';

import { colors } from "@/util/colors";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button, Container, Form, InputGroup, Spinner } from "react-bootstrap";
import { FaEye, FaEyeSlash } from "react-icons/fa";
import { useMobileContext } from "@/context/mobile";
import { useSignIn } from "@/hook/auth";
import { useAuth } from "@clerk/nextjs";

const initialFormState = { 
  email: "", 
  password: "", 
};

interface SignInProps {
  gymId?: string | null;
}

const SignIn = ({ gymId = null }: SignInProps) => {
  const router = useRouter();
  const mutation = useSignIn();
  const [formData, setFormData] = useState(initialFormState);
  const { isMobile } = useMobileContext();
  const [showPassword, setShowPassword] = useState(false);
  const { isSignedIn } = useAuth();

  if (mutation.isPending) {
    return <Spinner />;
  }

  return (
    <Container style={{
      padding: isMobile ? 30 : '40px 100px 50px 100px',
      border: '1px solid rgba(0, 0, 0, 0.1)',
      boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
    }}>
      <Form onSubmit={async (e) => {
        e.preventDefault();
        try {
          mutation.mutate({
            ...formData,
            gymId,
          });
        } catch (e) {
          console.error("Error: ", e);
        }
      }}>
        <h2 style={{ paddingBottom: isMobile ? 10 : 40 }}>Sign in to your account</h2>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          {!isMobile && <InputGroup.Text>Email</InputGroup.Text>}
          <Form.Control
            placeholder="Email"
            aria-label="Email"
            aria-describedby="email"
            value={formData.email}
            onChange={e => setFormData({ ...formData, email: e.target.value })}
            required
          />
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 20 }}>
          {!isMobile && <InputGroup.Text>Password</InputGroup.Text>}
          <Form.Control 
            type={showPassword ? 'text' : 'password'}
            aria-label="password"
            placeholder="Password" 
            value={formData.password}
            onChange={e => setFormData({ ...formData, password: e.target.value })}
            required
          />
          <InputGroup.Text style={{ backgroundColor: colors.black }}>
            {showPassword ? 
              <FaEye color={colors.white} onClick={() => {
                setShowPassword(!showPassword);
              }}/> : 
              <FaEyeSlash color={colors.white} onClick={() => {
                setShowPassword(!showPassword);
              }}/>
            }
          </InputGroup.Text>
        </InputGroup>
        <Button 
          variant="dark" 
          type="submit" 
          style={{ width: '100%' }}
          disabled={mutation.isPending}
        >
          {mutation.isPending ? 'Signing in...' : 'Submit'}
        </Button>
        <a 
          style={{ 
            display: 'flex',
            justifyContent: 'center',
            marginTop: 30,
            cursor: 'pointer', 
          }}
          onClick={() => router.push('/auth/reset-password')}
        >
          Reset Password
        </a>
        <a 
          style={{ 
            display: 'flex',
            justifyContent: 'center',
            marginTop: 30,
            cursor: 'pointer',
          }}
          onClick={() => router.push(isSignedIn ? '/auth/account-type' : '/')}
        >
          Back to home
        </a>
      </Form>
    </Container>
  );
};

export default SignIn;