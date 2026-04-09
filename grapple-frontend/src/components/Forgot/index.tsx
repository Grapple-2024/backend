'use client';

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button, Container, Form, InputGroup } from "react-bootstrap";

import { FaEye, FaEyeSlash } from "react-icons/fa";
import { colors } from "@/util/colors";
import { useMessagingContext } from "@/context/message";

const initialFormState = {
  email: "",
  newPassword: "",
  confirmNewPassword: "",
  confirmationCode: "",
};

const Forgot = ({
  profileType,
}: { profileType: string}) => {
  const [formData, setFormData] = useState(initialFormState);
  const [showPassword, setShowPassword] = useState(false);
  const {
    setMessage,
    setShow,
    setColor
  } = useMessagingContext();

  const router = useRouter();

  return (
    <Container style={{
      padding: '40px 100px 50px 100px',
      border: '1px solid rgba(0, 0, 0, 0.1)',
      boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
    }}>
      <Form>
        <h2 style={{ paddingBottom: 40 }}>Reset Password</h2>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          <InputGroup.Text>Email</InputGroup.Text>
          <Form.Control
            placeholder="Email"
            aria-label="Email"
            aria-describedby="email"
            value={formData.email}
            onChange={e => setFormData({ ...formData, 'email': e.target.value })}
          />
          <InputGroup.Text onClick={async () => {
            if (!formData.email) {
              setMessage('Please enter an email address and press "Send Code" to receive a confirmation code.');
              setShow(true);
              setColor('danger');
            } else {
              await fetch('/api/auth/forgot-password', {
                method: 'POST',
                body: JSON.stringify({ email: formData.email }),
              });
              setMessage('A confirmation code has been sent to your email address.');
              setShow(true);
              setColor('success');
            }
          }}>
            <a style={{ 
              display: 'flex',
              justifyContent: 'center',
              cursor: 'pointer',
            }}>
              Send Code
            </a>
          </InputGroup.Text>
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          <InputGroup.Text>New Password</InputGroup.Text>
          <Form.Control
            placeholder="**************"
            type={showPassword ? 'text' : 'password'}
            aria-label="new password"
            aria-describedby="new-password"
            value={formData.newPassword}
            onChange={e => setFormData({ ...formData, 'newPassword': e.target.value })}
          />
          <InputGroup.Text style={{ backgroundColor: colors.black }}>
            {showPassword ? 
              <FaEye color={colors.white} onClick={() => {
                setShowPassword(false);
              }}/> : 
              <FaEyeSlash color={colors.white} onClick={() => {
                setShowPassword(true);
              }}/>
            }
          </InputGroup.Text>
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          <InputGroup.Text>Confirm New Password</InputGroup.Text>
          <Form.Control
            placeholder="**************"
            type={showPassword ? 'text' : 'password'}
            aria-label="confirm new password"
            aria-describedby="confirm-new-password"
            value={formData.confirmNewPassword}
            onChange={e => setFormData({ ...formData, 'confirmNewPassword': e.target.value })}
          />
          <InputGroup.Text style={{ backgroundColor: colors.black }}>
            {showPassword ? 
              <FaEye color={colors.white} onClick={() => {
                setShowPassword(false);
              }}/> : 
              <FaEyeSlash color={colors.white} onClick={() => {
                setShowPassword(true);
              }}/>
            }
          </InputGroup.Text>
        </InputGroup>
        <InputGroup className="mb-3" style={{ paddingBottom: 10 }}>
          <InputGroup.Text>Confirmation Code</InputGroup.Text>
          <Form.Control
            placeholder="Confirmation Code"
            aria-label="Confirmation Code"
            aria-describedby="confirmation-code"
            value={formData.confirmationCode}
            onChange={e => setFormData({ ...formData, 'confirmationCode': e.target.value })}
          />
        </InputGroup>
        <Button 
          variant="dark" 
          type="submit" 
          style={{ 
            width: '100%',
          }}
          onClick={async (e) => {
            e.preventDefault();
            try {
              if (!formData.email || !formData.newPassword || !formData.confirmNewPassword || !formData.confirmationCode) {
                throw new Error('Please fill out all fields.');
              }

              await fetch('/api/auth/reset-password', {
                method: 'POST',
                body: JSON.stringify({
                  email: formData.email,
                  newPassword: formData.newPassword,
                  code: formData.confirmationCode // code from email
                })
              });
  
              setMessage('Password reset successfully');
              setShow(true);
              setColor('success');

              router.push(`/auth`);
            } catch (error: any) {
              setMessage("Error resetting password: ", error.message);
              setShow(true);
              setColor('danger');
            }
          }}
        >
          Reset
        </Button>
        <a 
          style={{ 
            display: 'flex',
            justifyContent: 'center',
            marginTop: 30,
            cursor: 'pointer', 
          }}
          onClick={() => {
            router.push(`/auth`);
          }}
        >
          Back to Sign In
        </a>
      </Form>
    </Container>
  );
};

export default Forgot;